package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBedrockInvocationTarget(t *testing.T) {
	t.Run("legacy target uses existing model support when routing disabled", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		assert.True(t, target.Legacy)
		assert.Nil(t, target.RouteKey)
		assert.Equal(t, "anthropic.claude-opus-4-6-v1", target.Support.CanonicalModel)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "us-east-1", target.RuntimeRegion)
	})

	t.Run("single route selects explicit scope and preferred region", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":                "us-east-1",
				"aws_route_mode":            "single_route",
				"aws_route_scope":           "eu",
				"aws_route_preferred_region": "eu-central-1",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "eu", target.RouteKey.Scope)
		assert.Equal(t, "eu-central-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "eu.anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "eu-central-1", target.RuntimeRegion)
	})

	t.Run("all routes returns first route from filtered pool surface", func(t *testing.T) {
		account := &Account{
			ID:       42,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "us-east-1",
				"aws_route_mode": "all_routes",
				"aws_route_scope": "us",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "anthropic.claude-opus-4-6-v1", target.Support.CanonicalModel)
		assert.Equal(t, "us", target.RouteKey.Scope)
		assert.Equal(t, "us-east-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
	})

	t.Run("route enabled non catalog model returns error", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "us-east-1",
				"aws_route_mode": "all_routes",
			},
		}

		_, err := ResolveBedrockInvocationTarget(account, "deepseek-r1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "route catalog")
	})

	t.Run("route mode conflicts with force global", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":       "us-east-1",
				"aws_force_global": "true",
				"aws_route_mode":   "all_routes",
			},
		}

		_, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws_force_global")
	})
}

func TestBedrockRoutePool(t *testing.T) {
	routes := []BedrockRoute{
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-opus-4-6-v1"},
	}
	pool := NewBedrockRoutePool(routes)

	first, ok := pool.SelectNextRoute(100)
	require.True(t, ok)
	assert.Equal(t, "us-east-1", first.Key.RuntimeRegion)

	pool.MarkCooldown(first.Key, 200)

	second, ok := pool.SelectNextRoute(100)
	require.True(t, ok)
	assert.Equal(t, "us-east-2", second.Key.RuntimeRegion)

	pool.MarkCooldown(second.Key, 200)
	_, ok = pool.SelectNextRoute(100)
	assert.False(t, ok)
}

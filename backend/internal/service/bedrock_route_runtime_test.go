package service

import (
	"testing"
	"time"

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

	t.Run("legacy target aligns direct canonical on-demand ids with catalog runtime", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "anthropic.claude-opus-4-6-v1")
		require.NoError(t, err)
		assert.True(t, target.Legacy)
		assert.Nil(t, target.RouteKey)
		assert.Equal(t, "anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "eu-west-2", target.RuntimeRegion)
	})

	t.Run("single route selects explicit scope and preferred region", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":                 "us-east-1",
				"aws_route_mode":             "single_route",
				"aws_route_scope":            "eu",
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

	t.Run("single route on_demand locks to unprefixed model", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":      "us-east-1",
				"aws_route_mode":  "single_route",
				"aws_route_scope": "on_demand",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "", target.RouteKey.Scope)
		assert.Equal(t, "anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "eu-west-2", target.RuntimeRegion)
	})

	t.Run("all routes returns first route from filtered pool surface", func(t *testing.T) {
		account := &Account{
			ID:       42,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":      "us-east-1",
				"aws_route_mode":  "all_routes",
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

	t.Run("all routes empty scope preserves runtime region baseline first", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       142,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "us-east-1",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "us-east-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
	})

	t.Run("all routes empty scope ignores unprefixed direct canonical baseline", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       149,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "us-east-1",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "anthropic.claude-opus-4-6-v1")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "us-east-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
	})

	t.Run("all routes empty scope preserves non-east baseline runtime first", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       143,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "us-west-2",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "us-west-2", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
	})

	t.Run("all routes empty scope preserves baseline family for non-catalog eu caller region", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       145,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "eu-west-1",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "eu.anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "eu-central-1", target.RouteKey.RuntimeRegion)
	})

	t.Run("all routes empty scope prefers apac fallback route over on-demand drift", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       146,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "ap-south-1",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "au.anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "ap-southeast-2", target.RouteKey.RuntimeRegion)
	})

	t.Run("all routes empty scope applies apac fallback for direct canonical model ids", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       147,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":     "ap-south-1",
				"aws_route_mode": "all_routes",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "anthropic.claude-opus-4-6-v1")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "au.anthropic.claude-opus-4-6-v1", target.InvocationModel)
		assert.Equal(t, "ap-southeast-2", target.RouteKey.RuntimeRegion)
	})

	t.Run("all routes preferred region preserves baseline invocation inside same region", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       144,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":                 "us-east-1",
				"aws_route_mode":             "all_routes",
				"aws_route_preferred_region": "us-east-1",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
		assert.Equal(t, "us-east-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", target.InvocationModel)
	})

	t.Run("all routes preferred region ignores unprefixed direct canonical baseline", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		account := &Account{
			ID:       148,
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":                 "us-east-1",
				"aws_route_mode":             "all_routes",
				"aws_route_preferred_region": "us-east-1",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "anthropic.claude-opus-4-6-v1")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.False(t, target.Legacy)
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

		_, err := ResolveBedrockInvocationTarget(account, "deepseek.r1-v1:0")
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

	t.Run("single route jp scope uses tokyo route semantics", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":      "ap-northeast-1",
				"aws_route_mode":  "single_route",
				"aws_route_scope": "jp",
			},
		}

		target, err := ResolveBedrockInvocationTarget(account, "claude-sonnet-4-20250514")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		assert.Equal(t, "jp", target.RouteKey.Scope)
		assert.Equal(t, "ap-northeast-1", target.RouteKey.RuntimeRegion)
		assert.Equal(t, "jp.anthropic.claude-sonnet-4-20250514-v1:0", target.InvocationModel)
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

func TestSelectAllRoutesBedrockTarget_ExpiredCooldownDoesNotBlockRoute(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	account := &Account{ID: 99}
	policy := BedrockRoutePolicy{Mode: "all_routes", Scope: "us"}
	routes := filterBedrockRoutesByScope(LookupBedrockRoutes("anthropic.claude-opus-4-6-v1"), policy.Scope)
	require.Len(t, routes, 3)

	pool := runtimeBedrockRoutePools.getOrCreate(routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "us.anthropic.claude-opus-4-6-v1"), routes)
	pool.MarkCooldown(routes[0].Key, time.Now().Add(-time.Minute).Unix())

	selected, err := selectAllRoutesBedrockTarget(account, "anthropic.claude-opus-4-6-v1", policy, routes, "us-east-1", "us.anthropic.claude-opus-4-6-v1", true)
	require.NoError(t, err)
	assert.Equal(t, routes[0].Key.RuntimeRegion, selected.Key.RuntimeRegion)
}

func TestSelectAllRoutesBedrockTarget_PreferredRegionCooldownFallsThroughToNextHealthyRoute(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	account := &Account{ID: 100}
	policy := BedrockRoutePolicy{Mode: "all_routes", Scope: "us", PreferredRegion: "us-east-1"}
	routes := filterBedrockRoutesByScope(LookupBedrockRoutes("anthropic.claude-opus-4-6-v1"), policy.Scope)
	require.Len(t, routes, 3)

	pool := runtimeBedrockRoutePools.getOrCreate(routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "us.anthropic.claude-opus-4-6-v1"), routes)
	pool.MarkCooldown(routes[0].Key, time.Now().Add(time.Hour).Unix())

	selected, err := selectAllRoutesBedrockTarget(account, "anthropic.claude-opus-4-6-v1", policy, routes, "us-east-1", "us.anthropic.claude-opus-4-6-v1", true)
	require.NoError(t, err)
	assert.Equal(t, "us-east-2", selected.Key.RuntimeRegion)
}

func TestPreviewBedrockInvocationTarget_DoesNotAdvanceRoutePool(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	account := &Account{
		ID:       101,
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region":      "us-east-1",
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "us",
		},
	}

	preview, err := PreviewBedrockInvocationTarget(account, "claude-opus-4-6")
	require.NoError(t, err)
	require.NotNil(t, preview.RouteKey)
	assert.Equal(t, "us-east-1", preview.RouteKey.RuntimeRegion)

	selected, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
	require.NoError(t, err)
	require.NotNil(t, selected.RouteKey)
	assert.Equal(t, "us-east-1", selected.RouteKey.RuntimeRegion)
}

func TestRoutePoolRegistryKey_IncludesPreferredRegion(t *testing.T) {
	account := &Account{ID: 102}
	keyA := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", BedrockRoutePolicy{Mode: "all_routes", Scope: "us", PreferredRegion: "us-east-1"}, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	keyB := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", BedrockRoutePolicy{Mode: "all_routes", Scope: "us", PreferredRegion: "us-east-2"}, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	assert.NotEqual(t, keyA, keyB)
}

func TestRoutePoolRegistryKey_PreferredRegionIgnoresBaselineInvocationModel(t *testing.T) {
	account := &Account{ID: 103}
	policy := BedrockRoutePolicy{Mode: "all_routes", Scope: "us", PreferredRegion: "us-east-1"}

	keyA := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-west-2", "us.anthropic.claude-opus-4-6-v1")
	keyB := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-west-2", "global.anthropic.claude-opus-4-6-v1")
	assert.Equal(t, keyA, keyB)
}

func TestRoutePoolRegistryKey_PreferredRegionKeepsBaselineInvocationWithinSameRegion(t *testing.T) {
	account := &Account{ID: 104}
	policy := BedrockRoutePolicy{Mode: "all_routes", PreferredRegion: "us-east-1"}

	keyA := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	keyB := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "global.anthropic.claude-opus-4-6-v1")
	assert.NotEqual(t, keyA, keyB)
}

func TestRoutePoolRegistryKey_EmptyScopeIgnoresUnprefixedBaselineInvocation(t *testing.T) {
	account := &Account{ID: 105}
	policy := BedrockRoutePolicy{Mode: "all_routes"}

	keyA := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "anthropic.claude-opus-4-6-v1")
	keyB := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "")
	assert.Equal(t, keyA, keyB)
}

func TestRoutePoolRegistryKey_UsesEffectiveRouteOrdering(t *testing.T) {
	account := &Account{ID: 106}
	policy := BedrockRoutePolicy{Mode: "all_routes", Scope: "us"}

	keyAlias := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	keyCanonical := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "anthropic.claude-opus-4-6-v1")
	assert.Equal(t, keyAlias, keyCanonical)
}

func TestRoutePoolRegistryKey_IgnoresEquivalentPolicyHints(t *testing.T) {
	account := &Account{ID: 107}

	keyA := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", BedrockRoutePolicy{Mode: "all_routes", Scope: "us"}, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	keyB := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", BedrockRoutePolicy{Mode: "all_routes", Scope: "us", PreferredRegion: "us-east-1"}, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	assert.Equal(t, keyA, keyB)
}

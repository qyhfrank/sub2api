package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBedrockRoutePolicy(t *testing.T) {
	canonicalModel := "anthropic.claude-opus-4-6-v1"

	t.Run("empty credentials disable routing", func(t *testing.T) {
		account := &Account{}

		policy, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.NoError(t, err)
		assert.Equal(t, "", policy.Mode)
		assert.Equal(t, "", policy.Scope)
		assert.Equal(t, "", policy.PreferredRegion)
	})

	t.Run("single route requires explicit scope", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{"aws_route_mode": "single_route"}}

		_, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws_route_scope")
	})

	t.Run("all routes accepts empty scope and preferred region", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":             "all_routes",
			"aws_route_preferred_region": "us-east-1",
		}}

		policy, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.NoError(t, err)
		assert.Equal(t, "all_routes", policy.Mode)
		assert.Equal(t, "", policy.Scope)
		assert.Equal(t, "us-east-1", policy.PreferredRegion)
	})

	t.Run("on demand scope normalizes to empty scope", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":  "single_route",
			"aws_route_scope": "on_demand",
		}}

		policy, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.NoError(t, err)
		assert.Equal(t, "single_route", policy.Mode)
		assert.Equal(t, "", policy.Scope)
	})

	t.Run("invalid scope on catalog model returns error", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":  "single_route",
			"aws_route_scope": "mars",
		}}

		_, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws_route_scope")
	})

	t.Run("invalid preferred region returns error", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":             "all_routes",
			"aws_route_preferred_region": "us-central-9",
		}}

		_, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws_route_preferred_region")
	})

	t.Run("route mode conflicts with force global", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":   "all_routes",
			"aws_force_global": "true",
		}}

		_, err := ResolveBedrockRoutePolicy(account, canonicalModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aws_force_global")
	})

	t.Run("route enabled non catalog model returns error", func(t *testing.T) {
		account := &Account{Credentials: map[string]any{
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "us",
		}}

		_, err := ResolveBedrockRoutePolicy(account, "deepseek.r1-v1:0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "route catalog")
	})
}

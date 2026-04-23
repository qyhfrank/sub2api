package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBedrockRouteCatalog(t *testing.T) {
	t.Run("lookup resolves canonical routes and keeps on demand scope empty", func(t *testing.T) {
		routes := LookupBedrockRoutes("us.anthropic.claude-opus-4-6-v1")
		require.NotEmpty(t, routes)

		var foundOnDemand bool
		var foundUS bool
		for _, route := range routes {
			assert.Equal(t, "anthropic.claude-opus-4-6-v1", route.Key.CanonicalModel)
			switch {
			case route.Key.Scope == "" && route.Key.RuntimeRegion == "eu-west-2":
				foundOnDemand = true
				assert.Equal(t, "anthropic.claude-opus-4-6-v1", route.InvocationModel)
			case route.Key.Scope == "us" && route.Key.RuntimeRegion == "us-east-1":
				foundUS = true
				assert.Equal(t, "us.anthropic.claude-opus-4-6-v1", route.InvocationModel)
			}
		}

		assert.True(t, foundOnDemand)
		assert.True(t, foundUS)
	})

	t.Run("non catalog model returns no routes", func(t *testing.T) {
		assert.Nil(t, LookupBedrockRoutes("deepseek.r1-v1:0"))
	})

	t.Run("covers current rewrite anthropic canonical set", func(t *testing.T) {
		for _, modelID := range []string{
			"anthropic.claude-opus-4-7-v1",
			"anthropic.claude-opus-4-6-v1",
			"anthropic.claude-opus-4-5-20251101-v1:0",
			"anthropic.claude-opus-4-1-20250805-v1:0",
			"anthropic.claude-opus-4-20250514-v1:0",
			"anthropic.claude-sonnet-4-6",
			"anthropic.claude-sonnet-4-5-20250929-v1:0",
			"anthropic.claude-sonnet-4-20250514-v1:0",
			"anthropic.claude-haiku-4-5-20251001-v1:0",
		} {
			t.Run(modelID, func(t *testing.T) {
				assert.NotEmpty(t, LookupBedrockRoutes(modelID))
			})
		}
	})

	t.Run("tokyo route uses jp scope and invocation prefix", func(t *testing.T) {
		routes := LookupBedrockRoutes("anthropic.claude-sonnet-4-20250514-v1:0")
		require.NotEmpty(t, routes)

		var foundJP bool
		for _, route := range routes {
			if route.Key.RuntimeRegion == "ap-northeast-1" {
				foundJP = true
				assert.Equal(t, "jp", route.Key.Scope)
				assert.Equal(t, "jp.anthropic.claude-sonnet-4-20250514-v1:0", route.InvocationModel)
			}
		}
		assert.True(t, foundJP)
	})
}

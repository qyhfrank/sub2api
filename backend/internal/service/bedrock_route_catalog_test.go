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
}

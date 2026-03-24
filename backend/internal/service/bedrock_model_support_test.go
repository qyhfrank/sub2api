package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBedrockModelSupport(t *testing.T) {
	t.Run("default alias resolves anthropic support contract", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "eu-west-1",
			},
		}

		support, ok := ResolveBedrockModelSupport(account, "claude-sonnet-4-5")
		require.True(t, ok)
		assert.Equal(t, "claude-sonnet-4-5", support.RequestedModel)
		assert.Equal(t, "claude-sonnet-4-5", support.MappedModel)
		assert.Equal(t, "anthropic.claude-sonnet-4-5-20250929-v1:0", support.CanonicalModel)
		assert.Equal(t, "eu.anthropic.claude-sonnet-4-5-20250929-v1:0", support.InvocationModel)
		assert.Equal(t, "eu-west-1", support.RuntimeRegion)
		assert.Equal(t, BedrockModelFamilyAnthropic, support.Family)
		assert.False(t, support.NeedsRequestAdapter)
		assert.False(t, support.NeedsResponseAdapter)
		assert.False(t, support.NeedsStreamAdapter)
	})

	t.Run("custom wildcard mapping keeps mapping precedence", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "ap-southeast-2",
				"model_mapping": map[string]any{
					"claude-*": "claude-opus-4-6",
				},
			},
		}

		support, ok := ResolveBedrockModelSupport(account, "claude-opus-4-6-thinking")
		require.True(t, ok)
		assert.Equal(t, "claude-opus-4-6", support.MappedModel)
		assert.Equal(t, "anthropic.claude-opus-4-6-v1", support.CanonicalModel)
		assert.Equal(t, "au.anthropic.claude-opus-4-6-v1", support.InvocationModel)
	})

	t.Run("force global only rewrites regional bedrock model ids", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region":       "us-east-1",
				"aws_force_global": "true",
				"model_mapping": map[string]any{
					"claude-sonnet-4-6": "us.anthropic.claude-sonnet-4-6",
				},
			},
		}

		support, ok := ResolveBedrockModelSupport(account, "claude-sonnet-4-6")
		require.True(t, ok)
		assert.Equal(t, "anthropic.claude-sonnet-4-6", support.CanonicalModel)
		assert.Equal(t, "global.anthropic.claude-sonnet-4-6", support.InvocationModel)
	})

	t.Run("third party direct model requires adapters", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		support, ok := ResolveBedrockModelSupport(account, "deepseek.r1-v1:0")
		require.True(t, ok)
		assert.Equal(t, "deepseek.r1-v1:0", support.CanonicalModel)
		assert.Equal(t, "deepseek.r1-v1:0", support.InvocationModel)
		assert.Equal(t, BedrockModelFamilyThirdParty, support.Family)
		assert.True(t, support.NeedsRequestAdapter)
		assert.True(t, support.NeedsResponseAdapter)
		assert.True(t, support.NeedsStreamAdapter)
	})

	t.Run("snapshot vendor prefixes pass through directly", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		for _, requested := range []string{
			"amazon.nova-pro-v1:0",
			"meta.llama3-3-70b-instruct-v1:0",
			"moonshotai.kimi-k2.5",
			"moonshot.kimi-k2-thinking",
			"zai.glm-4.7-flash",
			"openai.gpt-oss-120b-1:0",
			"deepseek.r1-v1:0",
		} {
			t.Run(requested, func(t *testing.T) {
				support, ok := ResolveBedrockModelSupport(account, requested)
				require.True(t, ok)
				assert.Equal(t, requested, support.InvocationModel)
				assert.Equal(t, CanonicalBedrockModelID(requested), support.CanonicalModel)
			})
		}
	})

	t.Run("new aliases resolve to third party model ids", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		for requested, expected := range map[string]string{
			"kimi-k2.5":        "moonshotai.kimi-k2.5",
			"kimi-k2-thinking": "moonshot.kimi-k2-thinking",
			"glm-4.7":          "zai.glm-4.7",
			"glm-4.7-flash":    "zai.glm-4.7-flash",
			"glm-5":            "zai.glm-5",
			"deepseek-v3.2":    "deepseek.v3.2",
			"gpt-oss-120b":     "openai.gpt-oss-120b-1:0",
		} {
			t.Run(requested, func(t *testing.T) {
				support, ok := ResolveBedrockModelSupport(account, requested)
				require.True(t, ok)
				assert.Equal(t, expected, support.InvocationModel)
				assert.Equal(t, CanonicalBedrockModelID(expected), support.CanonicalModel)
			})
		}
	})

	t.Run("unsupported alias returns false", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		_, ok := ResolveBedrockModelSupport(account, "claude-3-5-sonnet-20241022")
		assert.False(t, ok)
	})

	t.Run("unknown vendor prefix stays rejected", func(t *testing.T) {
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeBedrock,
			Credentials: map[string]any{
				"aws_region": "us-east-1",
			},
		}

		_, ok := ResolveBedrockModelSupport(account, "unknownvendor.some-model")
		assert.False(t, ok)
	})
}

func TestCanonicalBedrockModelID(t *testing.T) {
	assert.Equal(t, "anthropic.claude-opus-4-6-v1", CanonicalBedrockModelID("us.anthropic.claude-opus-4-6-v1"))
	assert.Equal(t, "anthropic.claude-sonnet-4-6", CanonicalBedrockModelID("global.anthropic.claude-sonnet-4-6"))
	assert.Equal(t, "anthropic.claude-haiku-4-5-20251001-v1:0", CanonicalBedrockModelID("anthropic.claude-haiku-4-5-20251001-v1:0"))
	assert.Equal(t, "deepseek.r1-v1:0", CanonicalBedrockModelID("deepseek.r1-v1:0"))
}

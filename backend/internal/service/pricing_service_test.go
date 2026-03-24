package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParsePricingData_ParsesPriorityAndServiceTierFields(t *testing.T) {
	svc := &PricingService{}
	body := []byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_creation_input_token_cost": 0.0000025,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"supports_prompt_caching": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`)

	data, err := svc.parsePricingData(body)
	require.NoError(t, err)
	pricing := data["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 3e-5, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 5e-7, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestGetModelPricing_Gpt53CodexSparkUsesGpt51CodexPricing(t *testing.T) {
	sparkPricing := &LiteLLMModelPricing{InputCostPerToken: 1}
	gpt53Pricing := &LiteLLMModelPricing{InputCostPerToken: 9}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": sparkPricing,
			"gpt-5.3":       gpt53Pricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex-spark")
	require.Same(t, sparkPricing, got)
}

func TestGetModelPricing_Gpt53CodexFallbackStillUsesGpt52Codex(t *testing.T) {
	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)
}

func TestGetModelPricing_OpenAIFallbackMatchedLoggedAsInfo(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)

	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "info"))
	require.False(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "warn"))
}

func TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": &LiteLLMModelPricing{InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, got)
	require.InDelta(t, 2.5e-6, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-5, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2.5e-7, got.CacheReadInputTokenCost, 1e-12)
	require.Equal(t, 272000, got.LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, got.LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, got.LongContextOutputCostMultiplier, 1e-12)
}

func TestGetModelPricing_BedrockDeepSeekAliasesResolveToCanonicalPricingKeys(t *testing.T) {
	deepseekV32Pricing := &LiteLLMModelPricing{InputCostPerToken: 3}
	deepseekR1Pricing := &LiteLLMModelPricing{InputCostPerToken: 7}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"deepseek-v3-2-251201": deepseekV32Pricing,
			"deepseek-reasoner":    deepseekR1Pricing,
		},
	}

	require.Same(t, deepseekV32Pricing, svc.GetModelPricing("deepseek-v3.2"))
	require.Same(t, deepseekV32Pricing, svc.GetModelPricing("deepseek.v3.2"))
	require.Same(t, deepseekR1Pricing, svc.GetModelPricing("deepseek-r1"))
	require.Same(t, deepseekR1Pricing, svc.GetModelPricing("deepseek.r1-v1:0"))
}

func TestGetModelPricing_BedrockThirdPartyAliasesResolveToCanonicalPricingKeys(t *testing.T) {
	kimiPricing := &LiteLLMModelPricing{InputCostPerToken: 6e-7}
	glmPricing := &LiteLLMModelPricing{InputCostPerToken: 1e-6}
	gptOss20bPricing := &LiteLLMModelPricing{InputCostPerToken: 1e-7}
	gptOss120bPricing := &LiteLLMModelPricing{InputCostPerToken: 1.5e-7}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"kimi-k2.5":    kimiPricing,
			"glm-5":        glmPricing,
			"gpt-oss-20b":  gptOss20bPricing,
			"gpt-oss-120b": gptOss120bPricing,
		},
	}

	require.Same(t, kimiPricing, svc.GetModelPricing("moonshotai.kimi-k2.5"))
	require.Same(t, glmPricing, svc.GetModelPricing("zai.glm-5"))
	require.Same(t, gptOss20bPricing, svc.GetModelPricing("openai.gpt-oss-20b-1:0"))
	require.Same(t, gptOss120bPricing, svc.GetModelPricing("openai.gpt-oss-120b-1:0"))
}

func TestLoadPricingData_MergesOverrideFile(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "base.json")
	overridePath := filepath.Join(tmpDir, "override.json")

	require.NoError(t, os.WriteFile(basePath, []byte(`{
		"deepseek-v3-2-251201": {
			"input_cost_per_token": 0.00000026,
			"output_cost_per_token": 0.00000038,
			"cache_read_input_token_cost": 0.00000013,
			"litellm_provider": "bedrock",
			"mode": "chat"
		}
	}`), 0644))
	require.NoError(t, os.WriteFile(overridePath, []byte(`{
		"kimi-k2.5": {
			"input_cost_per_token": 0.0000006,
			"output_cost_per_token": 0.000003,
			"cache_read_input_token_cost": 0.0000001,
			"litellm_provider": "bedrock",
			"mode": "chat"
		},
		"deepseek-v3-2-251201": {
			"input_cost_per_token": 0.0000003,
			"output_cost_per_token": 0.0000004,
			"cache_read_input_token_cost": 0.00000015,
			"litellm_provider": "bedrock",
			"mode": "chat"
		}
	}`), 0644))

	svc := &PricingService{cfg: &config.Config{Pricing: config.PricingConfig{OverrideFile: overridePath}}}
	require.NoError(t, svc.loadPricingData(basePath))

	kimi := svc.GetModelPricing("kimi-k2.5")
	require.NotNil(t, kimi)
	require.InDelta(t, 6e-7, kimi.InputCostPerToken, 1e-12)
	require.InDelta(t, 3e-6, kimi.OutputCostPerToken, 1e-12)

	deepseek := svc.GetModelPricing("deepseek-v3-2-251201")
	require.NotNil(t, deepseek)
	require.InDelta(t, 3e-7, deepseek.InputCostPerToken, 1e-12)
	require.InDelta(t, 4e-7, deepseek.OutputCostPerToken, 1e-12)
}

func TestParsePricingData_PreservesPriorityAndServiceTierFields(t *testing.T) {
	raw := map[string]any{
		"gpt-5.4": map[string]any{
			"input_cost_per_token":                 2.5e-6,
			"input_cost_per_token_priority":        5e-6,
			"output_cost_per_token":                15e-6,
			"output_cost_per_token_priority":       30e-6,
			"cache_read_input_token_cost":          0.25e-6,
			"cache_read_input_token_cost_priority": 0.5e-6,
			"supports_service_tier":                true,
			"supports_prompt_caching":              true,
			"litellm_provider":                     "openai",
			"mode":                                 "chat",
		},
	}
	body, err := json.Marshal(raw)
	require.NoError(t, err)

	svc := &PricingService{}
	pricingMap, err := svc.parsePricingData(body)
	require.NoError(t, err)

	pricing := pricingMap["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 2.5e-6, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 30e-6, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.5e-6, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesServiceTierPriorityFields(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 0.0000025, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 0.000005, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.000015, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 0.00003, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.00000025, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.0000005, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

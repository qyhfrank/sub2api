package service

import "strings"

const defaultBedrockRegion = "us-east-1"

var bedrockCrossRegionPrefixes = []string{"us.", "eu.", "apac.", "jp.", "au.", "us-gov.", "global."}

var bedrockModelPrefixes = []string{
	"ai21.",
	"amazon.",
	"anthropic.",
	"cohere.",
	"deepseek.",
	"meta.",
	"mistral.",
	"moonshot.",
	"moonshotai.",
	"nova.",
	"openai.",
	"stability.",
	"writer.",
	"zai.",
}

// DefaultBedrockModelMapping 是 AWS Bedrock 平台的默认模型映射。
// 此处的区域前缀仅为默认值，最终调用模型会根据账号配置调整到匹配区域。
var DefaultBedrockModelMapping = map[string]string{
	"claude-opus-4-6-thinking":   "us.anthropic.claude-opus-4-6-v1",
	"claude-opus-4-6":            "us.anthropic.claude-opus-4-6-v1",
	"claude-opus-4-5-thinking":   "us.anthropic.claude-opus-4-5-20251101-v1:0",
	"claude-opus-4-5-20251101":   "us.anthropic.claude-opus-4-5-20251101-v1:0",
	"claude-opus-4-1":            "us.anthropic.claude-opus-4-1-20250805-v1:0",
	"claude-opus-4-20250514":     "us.anthropic.claude-opus-4-20250514-v1:0",
	"claude-sonnet-4-6-thinking": "us.anthropic.claude-sonnet-4-6",
	"claude-sonnet-4-6":          "us.anthropic.claude-sonnet-4-6",
	"claude-sonnet-4-5":          "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-5-thinking": "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-5-20250929": "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
	"claude-sonnet-4-20250514":   "us.anthropic.claude-sonnet-4-20250514-v1:0",
	"claude-haiku-4-5":           "us.anthropic.claude-haiku-4-5-20251001-v1:0",
	"claude-haiku-4-5-20251001":  "us.anthropic.claude-haiku-4-5-20251001-v1:0",
	"deepseek-r1":                "deepseek.r1-v1:0",
	"deepseek-v3.2":              "deepseek.v3.2",
	"gpt-oss-120b":               "openai.gpt-oss-120b-1:0",
	"gpt-oss-20b":                "openai.gpt-oss-20b-1:0",
	"glm-4.7":                    "zai.glm-4.7",
	"glm-4.7-flash":              "zai.glm-4.7-flash",
	"glm-5":                      "zai.glm-5",
	"kimi-k2.5":                  "moonshotai.kimi-k2.5",
	"kimi-k2-thinking":           "moonshot.kimi-k2-thinking",
}

type BedrockModelFamily string

const (
	BedrockModelFamilyUnknown    BedrockModelFamily = "unknown"
	BedrockModelFamilyAnthropic  BedrockModelFamily = "anthropic"
	BedrockModelFamilyThirdParty BedrockModelFamily = "third_party"
)

type BedrockModelSupport struct {
	// Keep the full support contract explicit here so later routing work can
	// attach after canonical model resolution without reshaping call sites again.
	RequestedModel       string
	MappedModel          string
	CanonicalModel       string
	InvocationModel      string
	RuntimeRegion        string
	Family               BedrockModelFamily
	NeedsRequestAdapter  bool
	NeedsResponseAdapter bool
	NeedsStreamAdapter   bool
}

// BedrockCrossRegionPrefix 根据 AWS Region 返回 Bedrock 跨区域推理的模型 ID 前缀。
func BedrockCrossRegionPrefix(region string) string {
	switch {
	case strings.HasPrefix(region, "us-gov"):
		return "us-gov"
	case strings.HasPrefix(region, "us-"):
		return "us"
	case strings.HasPrefix(region, "eu-"):
		return "eu"
	case region == "ap-northeast-1":
		return "jp"
	case region == "ap-southeast-2":
		return "au"
	case strings.HasPrefix(region, "ap-"):
		return "apac"
	case strings.HasPrefix(region, "ca-"):
		return "us"
	case strings.HasPrefix(region, "sa-"):
		return "us"
	default:
		return "us"
	}
}

// AdjustBedrockModelRegionPrefix 将模型 ID 的区域前缀替换为与当前 AWS Region 匹配的前缀。
func AdjustBedrockModelRegionPrefix(modelID, region string) string {
	var targetPrefix string
	if region == "global" {
		targetPrefix = "global"
	} else {
		targetPrefix = BedrockCrossRegionPrefix(region)
	}

	for _, p := range bedrockCrossRegionPrefixes {
		if strings.HasPrefix(modelID, p) {
			if p == targetPrefix+"." {
				return modelID
			}
			return targetPrefix + "." + modelID[len(p):]
		}
	}

	return modelID
}

func bedrockRuntimeRegion(account *Account) string {
	if account == nil {
		return defaultBedrockRegion
	}
	if region := account.GetCredential("aws_region"); region != "" {
		return region
	}
	return defaultBedrockRegion
}

func shouldForceBedrockGlobal(account *Account) bool {
	return account != nil && account.GetCredential("aws_force_global") == "true"
}

func isRegionalBedrockModelID(modelID string) bool {
	for _, prefix := range bedrockCrossRegionPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}
	return false
}

func isLikelyBedrockModelID(modelID string) bool {
	lower := strings.ToLower(strings.TrimSpace(modelID))
	if lower == "" {
		return false
	}
	if strings.HasPrefix(lower, "arn:") {
		return true
	}
	for _, prefix := range bedrockModelPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return isRegionalBedrockModelID(lower)
}

func normalizeBedrockModelID(modelID string) (normalized string, shouldAdjustRegion bool, ok bool) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", false, false
	}
	if mapped, exists := DefaultBedrockModelMapping[modelID]; exists {
		return mapped, true, true
	}
	if isRegionalBedrockModelID(modelID) {
		return modelID, true, true
	}
	if isLikelyBedrockModelID(modelID) {
		return modelID, false, true
	}
	return "", false, false
}

func CanonicalBedrockModelID(modelID string) string {
	for _, prefix := range bedrockCrossRegionPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return modelID[len(prefix):]
		}
	}
	return modelID
}

func detectBedrockModelFamily(modelID string) BedrockModelFamily {
	canonical := strings.ToLower(CanonicalBedrockModelID(modelID))
	if strings.HasPrefix(canonical, "anthropic.") {
		return BedrockModelFamilyAnthropic
	}
	if canonical != "" {
		return BedrockModelFamilyThirdParty
	}
	return BedrockModelFamilyUnknown
}

func ResolveBedrockModelSupport(account *Account, requestedModel string) (BedrockModelSupport, bool) {
	if account == nil {
		return BedrockModelSupport{}, false
	}

	mappedModel := account.GetMappedModel(requestedModel)
	modelID, shouldAdjustRegion, ok := normalizeBedrockModelID(mappedModel)
	if !ok {
		return BedrockModelSupport{}, false
	}

	runtimeRegion := bedrockRuntimeRegion(account)
	invocationModel := modelID
	if shouldAdjustRegion {
		targetRegion := runtimeRegion
		if shouldForceBedrockGlobal(account) {
			targetRegion = "global"
		}
		invocationModel = AdjustBedrockModelRegionPrefix(modelID, targetRegion)
	}

	family := detectBedrockModelFamily(invocationModel)
	needsAdapters := family == BedrockModelFamilyThirdParty

	return BedrockModelSupport{
		RequestedModel:       requestedModel,
		MappedModel:          mappedModel,
		CanonicalModel:       CanonicalBedrockModelID(modelID),
		InvocationModel:      invocationModel,
		RuntimeRegion:        runtimeRegion,
		Family:               family,
		NeedsRequestAdapter:  needsAdapters,
		NeedsResponseAdapter: needsAdapters,
		NeedsStreamAdapter:   needsAdapters,
	}, true
}

// ResolveBedrockModelID resolves a requested Claude model into a Bedrock model ID.
func ResolveBedrockModelID(account *Account, requestedModel string) (string, bool) {
	support, ok := ResolveBedrockModelSupport(account, requestedModel)
	if !ok {
		return "", false
	}
	return support.InvocationModel, true
}

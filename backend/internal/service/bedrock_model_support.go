package service

import "strings"

type BedrockModelFamily string

const (
	BedrockModelFamilyUnknown    BedrockModelFamily = "unknown"
	BedrockModelFamilyAnthropic  BedrockModelFamily = "anthropic"
	BedrockModelFamilyThirdParty BedrockModelFamily = "third_party"
)

type BedrockModelSupport struct {
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

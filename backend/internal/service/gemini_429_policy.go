package service

import (
	"encoding/json"
	"strings"
	"time"
)

type gemini429Class string

const (
	gemini429ClassUnknown        gemini429Class = "unknown"
	gemini429ClassServerOverload gemini429Class = "server_overload"
	gemini429ClassDailyQuota     gemini429Class = "daily_quota"
	gemini429ClassQuotaReset     gemini429Class = "quota_reset_delay"
	gemini429ClassRetryDelay     gemini429Class = "retry_delay"
)

type gemini429Info struct {
	class           gemini429Class
	isDailyQuota    bool
	hasResetDelay   bool
	quotaResetDelay time.Duration
	retryDelay      time.Duration
}

func normalizeGemini429Body(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	normalized, err := unwrapGeminiResponse(body)
	if err != nil || len(normalized) == 0 {
		return body
	}
	return normalized
}

func parseGemini429Info(body []byte) gemini429Info {
	body = normalizeGemini429Body(body)
	info := gemini429Info{class: gemini429ClassUnknown}
	if looksLikeGeminiServerOverloadBody(body) {
		info.class = gemini429ClassServerOverload
		return info
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok {
				info.isDailyQuota = looksLikeGeminiDailyQuota(msg)
			}
			if details, ok := errObj["details"].([]any); ok {
				for _, d := range details {
					dm, ok := d.(map[string]any)
					if !ok {
						continue
					}
					if meta, ok := dm["metadata"].(map[string]any); ok {
						if v, ok := meta["quotaResetDelay"].(string); ok {
							if dur, err := time.ParseDuration(v); err == nil {
								info.hasResetDelay = true
								info.quotaResetDelay = dur
							}
						}
					}
				}
			}
		}
	}

	matches := retryInRegex.FindStringSubmatch(string(body))
	if len(matches) == 2 {
		if dur, err := time.ParseDuration(matches[1] + "s"); err == nil {
			info.retryDelay = dur
		}
	}

	switch {
	case info.isDailyQuota:
		info.class = gemini429ClassDailyQuota
	case info.hasResetDelay:
		info.class = gemini429ClassQuotaReset
	case info.retryDelay > 0:
		info.class = gemini429ClassRetryDelay
	default:
		info.class = gemini429ClassUnknown
	}
	return info
}

func shouldFastFailoverGemini429(body []byte) (bool, string) {
	info := parseGemini429Info(body)
	switch info.class {
	case gemini429ClassServerOverload:
		return false, string(info.class)
	case gemini429ClassDailyQuota, gemini429ClassQuotaReset, gemini429ClassRetryDelay:
		return true, string(info.class)
	default:
		return false, string(info.class)
	}
}

func looksLikeGeminiServerOverloadBody(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	if looksLikeGeminiServerOverload(strings.TrimSpace(extractUpstreamErrorMessage(body))) {
		return true
	}
	return looksLikeGeminiServerOverload(string(body))
}

func looksLikeGeminiServerOverload(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "no capacity available")
}

func gemini429MarkPriority(class gemini429Class) int {
	switch class {
	case gemini429ClassDailyQuota:
		return 3
	case gemini429ClassQuotaReset:
		return 2
	case gemini429ClassRetryDelay:
		return 1
	case gemini429ClassUnknown:
		return 0
	default:
		return -1
	}
}

func shouldProcessGemini429Mark(marked bool, previous, current gemini429Class) bool {
	if !marked {
		return true
	}
	return gemini429MarkPriority(current) > gemini429MarkPriority(previous)
}

func gemini429InspectionBody(account *Account, body []byte) []byte {
	if account == nil || len(body) == 0 {
		return body
	}
	return unwrapIfNeeded(account.Type == AccountTypeOAuth, body)
}

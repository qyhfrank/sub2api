package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func convertBedrockThirdPartyMessages(body []byte, modelID string) []byte {
	if isBedrockAnthropicModel(modelID) {
		return body
	}

	systemText := stringifyBedrockThirdPartyContent(gjson.GetBytes(body, "system"))
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		if systemText != "" {
			body, _ = sjson.SetBytes(body, "messages", []map[string]any{{"role": "system", "content": systemText}})
			body, _ = sjson.DeleteBytes(body, "system")
		}
		return body
	}

	convertedMessages := make([]map[string]any, 0, len(messages.Array()))
	if systemText != "" {
		convertedMessages = append(convertedMessages, map[string]any{
			"role":    "system",
			"content": systemText,
		})
	}
	for _, msg := range messages.Array() {
		role := msg.Get("role").String()
		content := msg.Get("content")

		if !content.Exists() || !content.IsArray() {
			convertedMessages = append(convertedMessages, map[string]any{
				"role":    role,
				"content": content.Value(),
			})
			continue
		}

		textParts := make([]string, 0)
		assistantToolCalls := make([]map[string]any, 0)
		toolResults := make([]map[string]any, 0)

		for _, block := range content.Array() {
			switch block.Get("type").String() {
			case "text":
				text := block.Get("text").String()
				if text != "" {
					textParts = append(textParts, text)
				}
			case "tool_use":
				name := block.Get("name").String()
				if name == "" {
					continue
				}
				assistantToolCalls = append(assistantToolCalls, map[string]any{
					"id":   firstNonEmptyString(block.Get("id").String(), fmt.Sprintf("tool_%s", name)),
					"type": "function",
					"function": map[string]any{
						"name":      name,
						"arguments": block.Get("input").Raw,
					},
				})
			case "tool_result":
				toolResults = append(toolResults, map[string]any{
					"role":         "tool",
					"tool_call_id": block.Get("tool_use_id").String(),
					"content":      stringifyBedrockThirdPartyContent(block.Get("content")),
				})
			}
		}

		textContent := strings.Join(textParts, "\n")
		switch role {
		case "assistant":
			entry := map[string]any{"role": "assistant"}
			if textContent != "" {
				entry["content"] = textContent
			} else {
				entry["content"] = nil
			}
			if len(assistantToolCalls) > 0 {
				entry["tool_calls"] = assistantToolCalls
			}
			convertedMessages = append(convertedMessages, entry)
		case "user":
			if textContent != "" {
				convertedMessages = append(convertedMessages, map[string]any{
					"role":    "user",
					"content": textContent,
				})
			}
			if len(toolResults) > 0 {
				convertedMessages = append(convertedMessages, toolResults...)
			}
			if textContent == "" && len(toolResults) == 0 {
				convertedMessages = append(convertedMessages, map[string]any{
					"role":    role,
					"content": stringifyBedrockThirdPartyContent(content),
				})
			}
		default:
			convertedMessages = append(convertedMessages, map[string]any{
				"role":    role,
				"content": textContent,
			})
		}
	}

	converted, err := json.Marshal(convertedMessages)
	if err != nil {
		return body
	}
	body, _ = sjson.SetRawBytes(body, "messages", converted)
	body, _ = sjson.DeleteBytes(body, "system")
	return body
}

func stringifyBedrockThirdPartyContent(content gjson.Result) string {
	if !content.Exists() {
		return ""
	}
	if content.Type == gjson.String {
		return content.String()
	}
	if content.IsArray() {
		parts := make([]string, 0, len(content.Array()))
		for _, item := range content.Array() {
			if item.Get("type").String() == "text" {
				text := item.Get("text").String()
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return strings.TrimSpace(content.Raw)
}

func convertBedrockThirdPartyTools(body []byte, modelID string) []byte {
	if isBedrockAnthropicModel(modelID) {
		return body
	}

	tools := gjson.GetBytes(body, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return body
	}

	for i, tool := range tools.Array() {
		if tool.Get("type").String() == "function" && tool.Get("function").Exists() {
			continue
		}

		name := firstNonEmptyString(tool.Get("name").String(), tool.Get("function.name").String())
		if name == "" {
			continue
		}
		description := firstNonEmptyString(tool.Get("description").String(), tool.Get("function.description").String())
		parametersRaw := tool.Get("input_schema").Raw
		if parametersRaw == "" {
			parametersRaw = tool.Get("function.parameters").Raw
		}
		if parametersRaw == "" {
			parametersRaw = `{}`
		}

		function := map[string]any{
			"name":       name,
			"parameters": json.RawMessage(parametersRaw),
		}
		if description != "" {
			function["description"] = description
		}
		converted, err := json.Marshal(map[string]any{
			"type":     "function",
			"function": function,
		})
		if err != nil {
			continue
		}
		body, _ = sjson.SetRawBytes(body, fmt.Sprintf("tools.%d", i), converted)
	}

	return body
}

func convertBedrockThirdPartyToolChoice(body []byte, modelID string) []byte {
	if isBedrockAnthropicModel(modelID) {
		return body
	}

	toolChoice := gjson.GetBytes(body, "tool_choice")
	if !toolChoice.Exists() {
		return body
	}

	switch toolChoice.Get("type").String() {
	case "tool":
		name := firstNonEmptyString(toolChoice.Get("name").String(), toolChoice.Get("function.name").String())
		if name == "" {
			return body
		}
		converted, err := json.Marshal(map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": name,
			},
		})
		if err == nil {
			body, _ = sjson.SetRawBytes(body, "tool_choice", converted)
		}
	case "auto":
		body, _ = sjson.SetRawBytes(body, "tool_choice", []byte(`"auto"`))
	case "none":
		body, _ = sjson.SetRawBytes(body, "tool_choice", []byte(`"none"`))
	case "any":
		body, _ = sjson.SetRawBytes(body, "tool_choice", []byte(`"required"`))
	}

	return body
}

func isBedrockAnthropicModel(modelID string) bool {
	return detectBedrockModelFamily(modelID) == BedrockModelFamilyAnthropic
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

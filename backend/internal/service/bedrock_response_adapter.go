package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

func adaptBedrockNonStreamingBody(body []byte) []byte {
	body = transformBedrockInvocationMetrics(body)
	body = convertBedrockOpenAICompletionToClaudeMessage(body)
	return body
}

func convertBedrockOpenAICompletionToClaudeMessage(body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	parsed := gjson.ParseBytes(body)
	if parsed.Get("type").String() == "message" {
		return body
	}

	choices := parsed.Get("choices")
	if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
		return body
	}

	choice := choices.Array()[0]
	contentBlocks := convertOpenAIChoiceContentToClaudeBlocks(choice)
	if len(contentBlocks) == 0 {
		return body
	}

	response := map[string]any{
		"id":            firstNonEmptyString(parsed.Get("id").String(), "msg_bedrock"),
		"type":          "message",
		"role":          firstNonEmptyString(choice.Get("message.role").String(), "assistant"),
		"model":         parsed.Get("model").String(),
		"content":       contentBlocks,
		"stop_reason":   mapOpenAIFinishReasonToClaudeStopReason(choice.Get("finish_reason").String()),
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  firstPositiveInt(parsed.Get("usage.input_tokens").Int(), parsed.Get("usage.prompt_tokens").Int(), parsed.Get("usage.inputTokens").Int()),
			"output_tokens": firstPositiveInt(parsed.Get("usage.output_tokens").Int(), parsed.Get("usage.completion_tokens").Int(), parsed.Get("usage.outputTokens").Int()),
		},
	}

	converted, err := json.Marshal(response)
	if err != nil {
		return body
	}
	return converted
}

func convertOpenAIChoiceContentToClaudeBlocks(choice gjson.Result) []map[string]any {
	content := choice.Get("message.content")
	blocks := make([]map[string]any, 0)

	if content.Exists() {
		if content.Type == gjson.String {
			text := content.String()
			if text != "" {
				blocks = append(blocks, map[string]any{
					"type": "text",
					"text": text,
				})
			}
		} else if content.IsArray() {
			for _, item := range content.Array() {
				switch item.Get("type").String() {
				case "text", "output_text", "input_text":
					text := item.Get("text").String()
					if text == "" {
						continue
					}
					blocks = append(blocks, map[string]any{
						"type": "text",
						"text": text,
					})
				}
			}
		}
	}

	toolCalls := choice.Get("message.tool_calls")
	if toolCalls.Exists() && toolCalls.IsArray() {
		for _, toolCall := range toolCalls.Array() {
			name := toolCall.Get("function.name").String()
			if name == "" {
				continue
			}
			input := map[string]any{}
			args := strings.TrimSpace(toolCall.Get("function.arguments").String())
			if args != "" {
				_ = json.Unmarshal([]byte(args), &input)
			}
			blocks = append(blocks, map[string]any{
				"type":  "tool_use",
				"id":    firstNonEmptyString(toolCall.Get("id").String(), fmt.Sprintf("tool_%s", name)),
				"name":  name,
				"input": input,
			})
		}
	}

	return blocks
}

func mapOpenAIFinishReasonToClaudeStopReason(finishReason string) string {
	switch strings.ToLower(strings.TrimSpace(finishReason)) {
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "content_filtered"
	case "stop", "", "null":
		return "end_turn"
	default:
		return "end_turn"
	}
}

func firstPositiveInt(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

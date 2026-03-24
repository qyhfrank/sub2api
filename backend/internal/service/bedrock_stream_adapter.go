package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

type bedrockAnthropicSSEEvent struct {
	Event string
	Data  map[string]any
}

type bedrockOpenAIStreamState struct {
	started          bool
	openContentBlock bool
	contentIndex     int
	nextBlockIndex   int
	openToolIndex    int
	openToolID       string
	seenToolJSON     string
	sawToolUse       bool
}

func convertBedrockOpenAIChunkToAnthropicEvents(data []byte, state *bedrockOpenAIStreamState) ([]bedrockAnthropicSSEEvent, bool) {
	if state == nil || len(data) == 0 {
		return nil, false
	}

	parsed := gjson.ParseBytes(data)
	if parsed.Get("object").String() != "chat.completion.chunk" {
		return nil, false
	}

	choices := parsed.Get("choices")
	if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
		return nil, false
	}
	choice := choices.Array()[0]
	delta := choice.Get("delta")
	finishReason := choice.Get("finish_reason").String()
	inputTokens := firstPositiveInt(
		parsed.Get("usage.input_tokens").Int(),
		parsed.Get("usage.prompt_tokens").Int(),
		parsed.Get("usage.inputTokens").Int(),
		parsed.Get("amazon-bedrock-invocationMetrics.inputTokenCount").Int(),
	)
	outputTokens := firstPositiveInt(
		parsed.Get("usage.output_tokens").Int(),
		parsed.Get("usage.completion_tokens").Int(),
		parsed.Get("usage.outputTokens").Int(),
		parsed.Get("amazon-bedrock-invocationMetrics.outputTokenCount").Int(),
	)

	events := make([]bedrockAnthropicSSEEvent, 0, 6)
	if !state.started {
		state.started = true
		events = append(events, bedrockAnthropicSSEEvent{
			Event: "message_start",
			Data: map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":            firstNonEmptyString(parsed.Get("id").String(), "msg_bedrock"),
					"type":          "message",
					"role":          firstNonEmptyString(delta.Get("role").String(), "assistant"),
					"model":         parsed.Get("model").String(),
					"content":       []any{},
					"stop_reason":   nil,
					"stop_sequence": nil,
					"usage": map[string]any{
						"input_tokens":  inputTokens,
						"output_tokens": 0,
					},
				},
			},
		})
	}

	if delta.Get("content").Exists() && !delta.Get("tool_calls").Exists() && finishReason != "tool_calls" && !state.openContentBlock {
		events = append(events, bedrockAnthropicSSEEvent{
			Event: "content_block_start",
			Data: map[string]any{
				"type":          "content_block_start",
				"index":         state.nextBlockIndex,
				"content_block": map[string]any{"type": "text", "text": ""},
			},
		})
		state.contentIndex = state.nextBlockIndex
		state.nextBlockIndex++
		state.openContentBlock = true
	}

	if text := delta.Get("content").String(); text != "" {
		events = append(events, bedrockAnthropicSSEEvent{
			Event: "content_block_delta",
			Data: map[string]any{
				"type":  "content_block_delta",
				"index": state.contentIndex,
				"delta": map[string]any{"type": "text_delta", "text": text},
			},
		})
	}

	toolCalls := delta.Get("tool_calls")
	if toolCalls.Exists() && toolCalls.IsArray() {
		for _, toolCall := range toolCalls.Array() {
			name := toolCall.Get("function.name").String()
			incomingID := toolCall.Get("id").String()
			if state.openContentBlock {
				events = append(events, bedrockAnthropicSSEEvent{Event: "content_block_stop", Data: map[string]any{"type": "content_block_stop", "index": state.contentIndex}})
				state.openContentBlock = false
			}

			if name != "" {
				if state.openToolIndex >= 0 {
					events = append(events, bedrockAnthropicSSEEvent{Event: "content_block_stop", Data: map[string]any{"type": "content_block_stop", "index": state.openToolIndex}})
				}
				state.openToolIndex = state.nextBlockIndex
				state.nextBlockIndex++
				state.openToolID = firstNonEmptyString(incomingID, "toolu_"+randomHex(8))
				state.seenToolJSON = ""
				state.sawToolUse = true
				events = append(events, bedrockAnthropicSSEEvent{
					Event: "content_block_start",
					Data: map[string]any{
						"type":          "content_block_start",
						"index":         state.openToolIndex,
						"content_block": map[string]any{"type": "tool_use", "id": state.openToolID, "name": name, "input": map[string]any{}},
					},
				})
			}

			args := strings.TrimSpace(toolCall.Get("function.arguments").String())
			if args != "" && state.openToolIndex >= 0 {
				deltaJSON, newSeen := computeGeminiTextDelta(state.seenToolJSON, args)
				state.seenToolJSON = newSeen
				if deltaJSON != "" {
					events = append(events, bedrockAnthropicSSEEvent{
						Event: "content_block_delta",
						Data: map[string]any{
							"type":  "content_block_delta",
							"index": state.openToolIndex,
							"delta": map[string]any{"type": "input_json_delta", "partial_json": deltaJSON},
						},
					})
				}
			}
		}
	}

	if finishReason != "" {
		if state.openContentBlock {
			events = append(events, bedrockAnthropicSSEEvent{Event: "content_block_stop", Data: map[string]any{"type": "content_block_stop", "index": state.contentIndex}})
			state.openContentBlock = false
		}
		if state.openToolIndex >= 0 {
			events = append(events, bedrockAnthropicSSEEvent{Event: "content_block_stop", Data: map[string]any{"type": "content_block_stop", "index": state.openToolIndex}})
			state.openToolIndex = -1
			state.openToolID = ""
			state.seenToolJSON = ""
		}

		usageObj := map[string]any{"output_tokens": outputTokens}
		if inputTokens > 0 {
			usageObj["input_tokens"] = inputTokens
		}
		stopReason := mapOpenAIFinishReasonToClaudeStopReason(finishReason)
		if state.sawToolUse {
			stopReason = "tool_use"
		}

		events = append(events,
			bedrockAnthropicSSEEvent{Event: "message_delta", Data: map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil}, "usage": usageObj}},
			bedrockAnthropicSSEEvent{Event: "message_stop", Data: map[string]any{"type": "message_stop"}},
		)
	}

	return events, true
}

func applyBedrockConvertedUsage(usage *ClaudeUsage, event bedrockAnthropicSSEEvent) {
	if usage == nil {
		return
	}

	switch event.Event {
	case "message_start":
		message, ok := event.Data["message"].(map[string]any)
		if !ok {
			return
		}
		usageMap, ok := message["usage"].(map[string]any)
		if !ok {
			return
		}
		if v, ok := intFromAny(usageMap["input_tokens"]); ok {
			usage.InputTokens = v
		}
		if v, ok := intFromAny(usageMap["output_tokens"]); ok && v > 0 {
			usage.OutputTokens = v
		}
	case "message_delta":
		usageMap, ok := event.Data["usage"].(map[string]any)
		if !ok {
			return
		}
		if v, ok := intFromAny(usageMap["input_tokens"]); ok && v > 0 {
			usage.InputTokens = v
		}
		if v, ok := intFromAny(usageMap["output_tokens"]); ok && v > 0 {
			usage.OutputTokens = v
		}
	}
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

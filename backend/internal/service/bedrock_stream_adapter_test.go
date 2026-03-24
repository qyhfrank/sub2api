package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertBedrockOpenAIChunkToAnthropicEvents_TextChunk(t *testing.T) {
	state := &bedrockOpenAIStreamState{openToolIndex: -1}
	chunk := []byte(`{
		"id":"chatcmpl_123",
		"object":"chat.completion.chunk",
		"model":"moonshotai.kimi-k2.5",
		"choices":[{
			"delta":{"role":"assistant","content":"hello"}
		}],
		"usage":{"prompt_tokens":11}
	}`)

	events, converted := convertBedrockOpenAIChunkToAnthropicEvents(chunk, state)
	require.True(t, converted)
	require.Len(t, events, 3)
	require.Equal(t, "message_start", events[0].Event)
	require.Equal(t, "content_block_start", events[1].Event)
	require.Equal(t, "content_block_delta", events[2].Event)
	delta, ok := events[2].Data["delta"].(map[string]any)
	require.True(t, ok)
	text, ok := delta["text"].(string)
	require.True(t, ok)
	require.Equal(t, "hello", text)
}

func TestConvertBedrockOpenAIChunkToAnthropicEvents_ToolCallFinish(t *testing.T) {
	state := &bedrockOpenAIStreamState{openToolIndex: -1}
	chunk := []byte(`{
		"id":"chatcmpl_456",
		"object":"chat.completion.chunk",
		"model":"moonshotai.kimi-k2.5",
		"choices":[{
			"delta":{
				"role":"assistant",
				"tool_calls":[{
					"id":"call_123",
					"function":{"name":"bash","arguments":"{\"cmd\":\"echo hi\"}"}
				}]
			},
			"finish_reason":"tool_calls"
		}],
		"usage":{"prompt_tokens":12,"completion_tokens":7}
	}`)

	events, converted := convertBedrockOpenAIChunkToAnthropicEvents(chunk, state)
	require.True(t, converted)
	require.Len(t, events, 6)
	require.Equal(t, "message_start", events[0].Event)
	require.Equal(t, "content_block_start", events[1].Event)
	require.Equal(t, "content_block_delta", events[2].Event)
	require.Equal(t, "content_block_stop", events[3].Event)
	require.Equal(t, "message_delta", events[4].Event)
	require.Equal(t, "message_stop", events[5].Event)

	payload, err := json.Marshal(events[1].Data)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"tool_use"`)
	assert.Contains(t, string(payload), `"bash"`)
	assert.Contains(t, string(payload), `"call_123"`)
	delta, ok := events[4].Data["delta"].(map[string]any)
	require.True(t, ok)
	stopReason, ok := delta["stop_reason"].(string)
	require.True(t, ok)
	require.Equal(t, "tool_use", stopReason)
}

func TestApplyBedrockConvertedUsage(t *testing.T) {
	usage := &ClaudeUsage{}
	event := bedrockAnthropicSSEEvent{
		Event: "message_delta",
		Data: map[string]any{
			"usage": map[string]any{
				"input_tokens":  15,
				"output_tokens": 9,
			},
		},
	}

	applyBedrockConvertedUsage(usage, event)
	assert.Equal(t, 15, usage.InputTokens)
	assert.Equal(t, 9, usage.OutputTokens)
}

func TestConvertBedrockOpenAIChunkToAnthropicEvents_MultiToolSequence(t *testing.T) {
	state := &bedrockOpenAIStreamState{openToolIndex: -1}
	firstChunk := []byte(`{
		"id":"chatcmpl_789",
		"object":"chat.completion.chunk",
		"model":"moonshotai.kimi-k2.5",
		"choices":[{
			"delta":{
				"role":"assistant",
				"tool_calls":[{
					"id":"call_1",
					"function":{"name":"bash","arguments":"{\"cmd\":\"echo hi\"}"}
				}]
			}
		}]
	}`)
	secondChunk := []byte(`{
		"id":"chatcmpl_789",
		"object":"chat.completion.chunk",
		"model":"moonshotai.kimi-k2.5",
		"choices":[{
			"delta":{
				"tool_calls":[{
					"id":"call_2",
					"function":{"name":"read","arguments":"{\"path\":\"/tmp/x\"}"}
				}]
			},
			"finish_reason":"tool_calls"
		}],
		"usage":{"prompt_tokens":10,"completion_tokens":6}
	}`)

	firstEvents, converted := convertBedrockOpenAIChunkToAnthropicEvents(firstChunk, state)
	require.True(t, converted)
	require.Len(t, firstEvents, 3)
	assert.Equal(t, "content_block_start", firstEvents[1].Event)
	assert.Equal(t, "content_block_delta", firstEvents[2].Event)

	secondEvents, converted := convertBedrockOpenAIChunkToAnthropicEvents(secondChunk, state)
	require.True(t, converted)
	require.Len(t, secondEvents, 6)
	assert.Equal(t, "content_block_stop", secondEvents[0].Event)
	assert.Equal(t, "content_block_start", secondEvents[1].Event)
	payload, err := json.Marshal(secondEvents[1].Data)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"read"`)
	assert.Contains(t, string(payload), `"call_2"`)
	assert.Equal(t, "message_delta", secondEvents[4].Event)
	assert.Equal(t, "message_stop", secondEvents[5].Event)
}

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestConvertBedrockOpenAICompletionToClaudeMessage(t *testing.T) {
	input := []byte(`{
		"id":"chatcmpl_bedrock",
		"model":"moonshotai.kimi-k2.5",
		"choices":[{
			"finish_reason":"tool_calls",
			"message":{
				"role":"assistant",
				"content":"Need a tool",
				"tool_calls":[{
					"id":"call_123",
					"type":"function",
					"function":{"name":"bash","arguments":"{\"cmd\":\"echo hello\"}"}
				}]
			}
		}],
		"usage":{"prompt_tokens":12,"completion_tokens":34}
	}`)

	converted := convertBedrockOpenAICompletionToClaudeMessage(input)
	require.Equal(t, "message", gjson.GetBytes(converted, "type").String())
	assert.Equal(t, "assistant", gjson.GetBytes(converted, "role").String())
	assert.Equal(t, "moonshotai.kimi-k2.5", gjson.GetBytes(converted, "model").String())
	assert.Equal(t, "Need a tool", gjson.GetBytes(converted, "content.0.text").String())
	assert.Equal(t, "tool_use", gjson.GetBytes(converted, "content.1.type").String())
	assert.Equal(t, "bash", gjson.GetBytes(converted, "content.1.name").String())
	assert.Equal(t, "call_123", gjson.GetBytes(converted, "content.1.id").String())
	assert.Equal(t, "echo hello", gjson.GetBytes(converted, "content.1.input.cmd").String())
	assert.Equal(t, "tool_use", gjson.GetBytes(converted, "stop_reason").String())
	assert.Equal(t, int64(12), gjson.GetBytes(converted, "usage.input_tokens").Int())
	assert.Equal(t, int64(34), gjson.GetBytes(converted, "usage.output_tokens").Int())
}

func TestConvertBedrockOpenAICompletionToClaudeMessage_PassthroughClaudeMessage(t *testing.T) {
	input := []byte(`{"type":"message","role":"assistant","content":[{"type":"text","text":"hello"}]}`)
	converted := convertBedrockOpenAICompletionToClaudeMessage(input)
	assert.JSONEq(t, string(input), string(converted))
}

func TestParseClaudeUsageFromResponseBody_HandlesOpenAIUsageFallbacks(t *testing.T) {
	body := []byte(`{
		"usage":{
			"prompt_tokens":21,
			"completion_tokens":8,
			"cacheWriteInputTokens":3,
			"cacheReadInputTokens":5
		}
	}`)

	usage := parseClaudeUsageFromResponseBody(body)
	require.NotNil(t, usage)
	assert.Equal(t, 21, usage.InputTokens)
	assert.Equal(t, 8, usage.OutputTokens)
	assert.Equal(t, 3, usage.CacheCreationInputTokens)
	assert.Equal(t, 5, usage.CacheReadInputTokens)
}

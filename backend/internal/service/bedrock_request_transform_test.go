package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestPrepareBedrockRequestBody_NonClaudeConvertsToolsToOpenAIShape(t *testing.T) {
	input := `{
		"model":"kimi-k2.5",
		"tool_choice":{"type":"tool","name":"bash"},
		"tools":[
			{"name":"bash","description":"Run bash","input_schema":{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}},
			{"name":"read","description":"Read file","input_schema":{"type":"object"}}
		],
		"messages":[{"role":"user","content":"hi"}]
	}`

	result, err := PrepareBedrockRequestBody([]byte(input), "moonshotai.kimi-k2.5", "")
	require.NoError(t, err)

	tools := gjson.GetBytes(result, "tools").Array()
	require.Len(t, tools, 2)
	assert.Equal(t, "function", tools[0].Get("type").String())
	assert.Equal(t, "bash", tools[0].Get("function.name").String())
	assert.Equal(t, "Run bash", tools[0].Get("function.description").String())
	assert.Equal(t, "object", tools[0].Get("function.parameters.type").String())
	assert.Equal(t, "string", tools[0].Get("function.parameters.properties.cmd.type").String())

	assert.Equal(t, "function", gjson.GetBytes(result, "tool_choice.type").String())
	assert.Equal(t, "bash", gjson.GetBytes(result, "tool_choice.function.name").String())
}

func TestPrepareBedrockRequestBody_ClaudeKeepsAnthropicToolsShape(t *testing.T) {
	input := `{
		"tools":[{"name":"bash","description":"Run bash","input_schema":{"type":"object"}}],
		"tool_choice":{"type":"tool","name":"bash"},
		"messages":[{"role":"user","content":"hi"}]
	}`

	result, err := PrepareBedrockRequestBody([]byte(input), "us.anthropic.claude-opus-4-6-v1", "")
	require.NoError(t, err)

	assert.Equal(t, "bash", gjson.GetBytes(result, "tools.0.name").String())
	assert.False(t, gjson.GetBytes(result, "tools.0.function").Exists())
	assert.Equal(t, "tool", gjson.GetBytes(result, "tool_choice.type").String())
	assert.Equal(t, "bash", gjson.GetBytes(result, "tool_choice.name").String())
}

func TestPrepareBedrockRequestBody_NonClaudeConvertsToolUseAndToolResultMessages(t *testing.T) {
	input := `{
		"messages":[
			{"role":"user","content":"Use bash"},
			{"role":"assistant","content":[{"type":"tool_use","id":"functions.bash:0","name":"bash","input":{"cmd":"echo hello"}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"functions.bash:0","content":"hello","is_error":false}]}
		]
	}`

	result, err := PrepareBedrockRequestBody([]byte(input), "moonshotai.kimi-k2.5", "")
	require.NoError(t, err)

	msgs := gjson.GetBytes(result, "messages").Array()
	require.Len(t, msgs, 3)
	assert.Equal(t, "user", msgs[0].Get("role").String())
	assert.Equal(t, "Use bash", msgs[0].Get("content").String())
	assert.Equal(t, "assistant", msgs[1].Get("role").String())
	assert.Equal(t, "bash", msgs[1].Get("tool_calls.0.function.name").String())
	assert.Equal(t, `{"cmd":"echo hello"}`, msgs[1].Get("tool_calls.0.function.arguments").String())
	assert.Equal(t, "tool", msgs[2].Get("role").String())
	assert.Equal(t, "functions.bash:0", msgs[2].Get("tool_call_id").String())
	assert.Equal(t, "hello", msgs[2].Get("content").String())
}

func TestPrepareBedrockRequestBody_NonClaudeMovesSystemPromptAndStripsAnthropicFields(t *testing.T) {
	input := `{
		"messages":[{"role":"user","content":"hello"}],
		"system":[{"type":"text","text":"system rule"}],
		"anthropic_version":"should-be-removed",
		"anthropic_beta":["interleaved-thinking-2025-05-14"]
	}`

	result, err := PrepareBedrockRequestBodyWithTokens([]byte(input), "moonshotai.kimi-k2.5", nil)
	require.NoError(t, err)

	assert.False(t, gjson.GetBytes(result, "anthropic_version").Exists())
	assert.False(t, gjson.GetBytes(result, "anthropic_beta").Exists())
	assert.False(t, gjson.GetBytes(result, "system").Exists())
	assert.Equal(t, "system", gjson.GetBytes(result, "messages.0.role").String())
	assert.Equal(t, "system rule", gjson.GetBytes(result, "messages.0.content").String())
	assert.Equal(t, "user", gjson.GetBytes(result, "messages.1.role").String())
	assert.Equal(t, "hello", gjson.GetBytes(result, "messages.1.content").String())
}

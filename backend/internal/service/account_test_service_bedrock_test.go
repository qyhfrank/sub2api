package service

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBedrockAccountTestUsesRouteTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("route enabled uses routed url and signer region", func(t *testing.T) {
		ctx, recorder := newSoraTestContext()
		upstream := &queuedHTTPUpstream{responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"content":[{"text":"bedrock ok"}]}`),
		}}
		account := &Account{
			ID:          101,
			Name:        "bedrock-route",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeBedrock,
			Concurrency: 1,
			Credentials: map[string]any{
				"aws_access_key_id":          "AKIAEXAMPLE",
				"aws_secret_access_key":      "secret",
				"aws_region":                 "us-east-1",
				"aws_route_mode":             "single_route",
				"aws_route_scope":            "eu",
				"aws_route_preferred_region": "eu-central-1",
			},
		}

		svc := &AccountTestService{httpUpstream: upstream}

		err := svc.testClaudeAccountConnection(ctx, account, "claude-opus-4-6")
		require.NoError(t, err)
		require.Len(t, upstream.requests, 1)
		require.Equal(t, BuildBedrockURL("eu-central-1", "eu.anthropic.claude-opus-4-6-v1", false), upstream.requests[0].URL.String())
		require.Contains(t, upstream.requests[0].Header.Get("Authorization"), "/eu-central-1/bedrock/aws4_request")
		require.Contains(t, recorder.Body.String(), `"type":"test_complete","success":true`)
		require.Contains(t, recorder.Body.String(), `"text":"bedrock ok"`)
	})

	t.Run("route disabled keeps legacy target", func(t *testing.T) {
		ctx, _ := newSoraTestContext()
		upstream := &queuedHTTPUpstream{responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"content":[{"text":"legacy ok"}]}`),
		}}
		account := &Account{
			ID:          102,
			Name:        "bedrock-legacy",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeBedrock,
			Concurrency: 1,
			Credentials: map[string]any{
				"aws_access_key_id":     "AKIAEXAMPLE",
				"aws_secret_access_key": "secret",
				"aws_region":            "us-west-2",
			},
		}

		svc := &AccountTestService{httpUpstream: upstream}

		err := svc.testClaudeAccountConnection(ctx, account, "claude-opus-4-6")
		require.NoError(t, err)
		require.Len(t, upstream.requests, 1)
		require.Equal(t, BuildBedrockURL("us-west-2", "us.anthropic.claude-opus-4-6-v1", false), upstream.requests[0].URL.String())
		require.Contains(t, upstream.requests[0].Header.Get("Authorization"), "/us-west-2/bedrock/aws4_request")
	})

	t.Run("invalid route policy returns clear error before upstream call", func(t *testing.T) {
		ctx, recorder := newSoraTestContext()
		upstream := &queuedHTTPUpstream{}
		account := &Account{
			ID:          103,
			Name:        "bedrock-invalid-route",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeBedrock,
			Concurrency: 1,
			Credentials: map[string]any{
				"aws_access_key_id":     "AKIAEXAMPLE",
				"aws_secret_access_key": "secret",
				"aws_region":            "us-east-1",
				"aws_route_mode":        "single_route",
				"aws_route_scope":       "mars",
			},
		}

		svc := &AccountTestService{httpUpstream: upstream}

		err := svc.testClaudeAccountConnection(ctx, account, "claude-opus-4-6")
		require.Error(t, err)
		require.Contains(t, err.Error(), `invalid aws_route_scope "mars"`)
		require.Empty(t, upstream.requests)
		require.Contains(t, recorder.Body.String(), `"type":"error"`)
		require.Contains(t, recorder.Body.String(), `invalid aws_route_scope \"mars\"`)
	})

	t.Run("account test does not advance all routes runtime selection", func(t *testing.T) {
		runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
		ctx, _ := newSoraTestContext()
		upstream := &queuedHTTPUpstream{responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"content":[{"text":"pool ok"}]}`),
		}}
		account := &Account{
			ID:          104,
			Name:        "bedrock-route-pool-test",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeBedrock,
			Concurrency: 1,
			Credentials: map[string]any{
				"aws_access_key_id":     "AKIAEXAMPLE",
				"aws_secret_access_key": "secret",
				"aws_region":            "us-east-1",
				"aws_route_mode":        "all_routes",
				"aws_route_scope":       "us",
			},
		}

		svc := &AccountTestService{httpUpstream: upstream}

		err := svc.testClaudeAccountConnection(ctx, account, "claude-opus-4-6")
		require.NoError(t, err)
		require.Len(t, upstream.requests, 1)
		require.Equal(t, BuildBedrockURL("us-east-1", "us.anthropic.claude-opus-4-6-v1", false), upstream.requests[0].URL.String())

		target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
		require.NoError(t, err)
		require.NotNil(t, target.RouteKey)
		require.Equal(t, "us-east-1", target.RouteKey.RuntimeRegion)
	})
}

package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestForwardBedrockUsesRoutedInvocationTarget(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, _ := newTestContext()
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
		},
	}
	svc := &GatewayService{httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          41,
		Name:        "bedrock-route-single",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"aws_region":                 "us-east-1",
			"aws_route_mode":             "single_route",
			"aws_route_scope":            "eu",
			"aws_route_preferred_region": "eu-central-1",
			"auth_mode":                  "apikey",
			"api_key":                    "bedrock-test-key",
		},
	}
	parsed := mustParseBedrockRequest(t, "claude-opus-4-6")

	result, err := svc.forwardBedrock(context.Background(), ctx, account, parsed, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 1)
	require.Contains(t, upstream.requests[0].URL.Host, "bedrock-runtime.eu-central-1.amazonaws.com")
	require.Contains(t, upstream.requests[0].URL.Path, "/model/eu.anthropic.claude-opus-4-6-v1/invoke")
	require.Equal(t, "Bearer bedrock-test-key", upstream.requests[0].Header.Get("Authorization"))
	require.Equal(t, "eu.anthropic.claude-opus-4-6-v1", result.UpstreamModel)
}

func TestForwardBedrockLegacyTargetStaysUnchangedWhenRouteModeDisabled(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, _ := newTestContext()
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
		},
	}
	svc := &GatewayService{httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          42,
		Name:        "bedrock-legacy",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"aws_region": "us-east-1",
			"auth_mode":  "apikey",
			"api_key":    "bedrock-test-key",
		},
	}
	parsed := mustParseBedrockRequest(t, "claude-opus-4-6")

	result, err := svc.forwardBedrock(context.Background(), ctx, account, parsed, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 1)
	require.Contains(t, upstream.requests[0].URL.Host, "bedrock-runtime.us-east-1.amazonaws.com")
	require.Contains(t, upstream.requests[0].URL.Path, "/model/us.anthropic.claude-opus-4-6-v1/invoke")
	require.Equal(t, "us.anthropic.claude-opus-4-6-v1", result.UpstreamModel)
}

func TestForwardBedrockAllRoutesQuota429RetriesNextRoute(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, _ := newTestContext()
	quotaResp := newJSONResponse(http.StatusTooManyRequests, `{"message":"Invocation denied because the daily quota was exceeded for this model in this region."}`)
	quotaResp.Header.Set("x-amzn-errortype", "ServiceQuotaExceededException")
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			quotaResp,
			newJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
		},
	}
	svc := &GatewayService{httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          43,
		Name:        "bedrock-route-pool",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"aws_region":      "us-east-1",
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "us",
			"auth_mode":       "apikey",
			"api_key":         "bedrock-test-key",
		},
	}
	parsed := mustParseBedrockRequest(t, "claude-opus-4-6")

	result, err := svc.forwardBedrock(context.Background(), ctx, account, parsed, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Contains(t, upstream.requests[0].URL.Host, "bedrock-runtime.us-east-1.amazonaws.com")
	require.Contains(t, upstream.requests[1].URL.Host, "bedrock-runtime.us-east-2.amazonaws.com")

	policy, policyErr := ResolveBedrockRoutePolicy(account, "anthropic.claude-opus-4-6-v1")
	require.NoError(t, policyErr)
	poolKey := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy)
	pool := runtimeBedrockRoutePools.pools[poolKey]
	require.NotNil(t, pool)
	require.Greater(t, pool.cooldowns[BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-1"}], int64(0))
}

func TestForwardBedrockSingleRouteQuota429DoesNotFailover(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, recorder := newTestContext()
	quotaResp := newJSONResponse(http.StatusTooManyRequests, `{"message":"Invocation denied because the daily quota was exceeded for this model in this region."}`)
	quotaResp.Header.Set("x-amzn-errortype", "ServiceQuotaExceededException")
	upstream := &queuedHTTPUpstream{responses: []*http.Response{quotaResp}}
	svc := &GatewayService{httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          44,
		Name:        "bedrock-route-single-quota",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"aws_region":                 "us-east-1",
			"aws_route_mode":             "single_route",
			"aws_route_scope":            "us",
			"aws_route_preferred_region": "us-east-1",
			"auth_mode":                  "apikey",
			"api_key":                    "bedrock-test-key",
		},
	}
	parsed := mustParseBedrockRequest(t, "claude-opus-4-6")

	result, err := svc.forwardBedrock(context.Background(), ctx, account, parsed, time.Now())
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	require.Len(t, upstream.requests, 1)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, recorder.Body.String(), "rate_limit_error")
}

func mustParseBedrockRequest(t *testing.T, model string) *ParsedRequest {
	t.Helper()
	body := `{"model":"` + model + `","max_tokens":16,"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`
	parsed, err := ParseGatewayRequest([]byte(body), PlatformAnthropic)
	require.NoError(t, err)
	require.Equal(t, model, parsed.Model)
	require.False(t, parsed.Stream)
	require.True(t, strings.Contains(string(parsed.Body), model))
	return parsed
}

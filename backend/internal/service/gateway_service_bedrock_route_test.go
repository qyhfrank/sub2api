package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type bedrockRouteQueuedHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
}

func (u *bedrockRouteQueuedHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *bedrockRouteQueuedHTTPUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.requests = append(u.requests, req)
	if len(u.responses) == 0 {
		return nil, fmt.Errorf("no mocked response")
	}
	resp := u.responses[0]
	u.responses = u.responses[1:]
	return resp, nil
}

func newBedrockRouteJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newBedrockRouteTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, rec
}

func TestForwardBedrockUsesRoutedInvocationTarget(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, _ := newBedrockRouteTestContext()
	upstream := &bedrockRouteQueuedHTTPUpstream{
		responses: []*http.Response{
			newBedrockRouteJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
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

	ctx, _ := newBedrockRouteTestContext()
	upstream := &bedrockRouteQueuedHTTPUpstream{
		responses: []*http.Response{
			newBedrockRouteJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
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

	ctx, _ := newBedrockRouteTestContext()
	quotaResp := newBedrockRouteJSONResponse(http.StatusTooManyRequests, `{"message":"Invocation denied because the daily quota was exceeded for this model in this region."}`)
	quotaResp.Header.Set("x-amzn-errortype", "ServiceQuotaExceededException")
	upstream := &bedrockRouteQueuedHTTPUpstream{
		responses: []*http.Response{
			quotaResp,
			newBedrockRouteJSONResponse(http.StatusOK, `{"type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
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
	poolKey := routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy, "us-east-1", "us.anthropic.claude-opus-4-6-v1")
	pool := runtimeBedrockRoutePools.pools[poolKey]
	require.NotNil(t, pool)
	require.Greater(t, pool.cooldowns[BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-1"}], int64(0))
}

func TestForwardBedrockAllRoutesQuota429FallsBackWhenRoutePoolExhausted(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, recorder := newBedrockRouteTestContext()
	quotaResp := newBedrockRouteJSONResponse(http.StatusTooManyRequests, `{"message":"Invocation denied because the daily quota was exceeded for this model in this region."}`)
	quotaResp.Header.Set("x-amzn-errortype", "ServiceQuotaExceededException")
	upstream := &bedrockRouteQueuedHTTPUpstream{responses: []*http.Response{quotaResp}}
	svc := &GatewayService{httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          45,
		Name:        "bedrock-route-single-option",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"aws_region":      "eu-central-1",
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "eu",
			"auth_mode":       "apikey",
			"api_key":         "bedrock-test-key",
		},
	}
	parsed := mustParseBedrockRequest(t, "claude-opus-4-6")

	result, err := svc.forwardBedrock(context.Background(), ctx, account, parsed, time.Now())
	require.Nil(t, result)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Len(t, upstream.requests, 1)
	require.Equal(t, 200, recorder.Code)
	require.Empty(t, recorder.Body.String())
}

func TestForwardBedrockSingleRouteQuota429FallsBackToOuterFailover(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	gin.SetMode(gin.TestMode)

	ctx, recorder := newBedrockRouteTestContext()
	quotaResp := newBedrockRouteJSONResponse(http.StatusTooManyRequests, `{"message":"Invocation denied because the daily quota was exceeded for this model in this region."}`)
	quotaResp.Header.Set("x-amzn-errortype", "ServiceQuotaExceededException")
	upstream := &bedrockRouteQueuedHTTPUpstream{responses: []*http.Response{quotaResp}}
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
	require.True(t, errors.As(err, &failoverErr))
	require.Len(t, upstream.requests, 1)
	require.Equal(t, 200, recorder.Code)
	require.Empty(t, recorder.Body.String())
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

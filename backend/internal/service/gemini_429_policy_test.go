package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type accountRepoRateLimitSpy struct {
	AccountRepository
	calls               int
	account             int64
	resetAt             time.Time
	modelRateLimitCalls int
	modelRateLimitScope string
}

func (s *accountRepoRateLimitSpy) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	s.calls++
	s.account = id
	s.resetAt = resetAt
	return nil
}

func (s *accountRepoRateLimitSpy) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	s.modelRateLimitCalls++
	s.account = id
	s.modelRateLimitScope = scope
	s.resetAt = resetAt
	return nil
}

func TestShouldFastFailoverGemini429(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		want       bool
		wantReason string
	}{
		{
			name:       "server overload does not fast failover",
			body:       `{"error":{"message":"no capacity available"}}`,
			want:       false,
			wantReason: "server_overload",
		},
		{
			name:       "daily quota fast failover",
			body:       `{"error":{"message":"requests per day exceeded"}}`,
			want:       true,
			wantReason: "daily_quota",
		},
		{
			name:       "quota reset delay fast failover",
			body:       `{"error":{"details":[{"metadata":{"quotaResetDelay":"12s"}}]}}`,
			want:       true,
			wantReason: "quota_reset_delay",
		},
		{
			name:       "retry delay fast failover",
			body:       `Please retry in 30s`,
			want:       true,
			wantReason: "retry_delay",
		},
		{
			name:       "wrapped oauth quota reset delay fast failover",
			body:       `{"response":{"error":{"details":[{"metadata":{"quotaResetDelay":"12s"}}]}}}`,
			want:       true,
			wantReason: "quota_reset_delay",
		},
		{
			name:       "reset after fast failover",
			body:       `Your quota will reset after 3s`,
			want:       true,
			wantReason: "retry_delay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := shouldFastFailoverGemini429([]byte(tt.body))
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestHandleGeminiUpstreamError_SkipsRateLimitOnServerOverload(t *testing.T) {
	repo := &accountRepoRateLimitSpy{}
	svc := &GeminiMessagesCompatService{accountRepo: repo}
	account := &Account{
		ID:       42,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"tier_id":    "free",
		},
		Schedulable: true,
	}

	svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"no capacity available"}}`),
		"gemini-2.5-pro",
	)

	require.Equal(t, 0, repo.calls)
	require.Equal(t, 1, repo.modelRateLimitCalls)
	require.Equal(t, "gemini-2.5-pro", repo.modelRateLimitScope)
}

func TestHandleGeminiUpstreamError_UsesMappedModelScopeForAPIKeyOverload(t *testing.T) {
	repo := &accountRepoRateLimitSpy{}
	svc := &GeminiMessagesCompatService{accountRepo: repo}
	account := &Account{
		ID:       43,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"api_key": "test-key",
			"model_mapping": map[string]any{
				"gemini-2.5-pro": "gemini-2.5-pro-002",
			},
		},
		Schedulable: true,
	}

	svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"no capacity available"}}`),
		"gemini-2.5-pro",
	)

	require.Equal(t, 0, repo.calls)
	require.Equal(t, 1, repo.modelRateLimitCalls)
	require.Equal(t, "gemini-2.5-pro-002", repo.modelRateLimitScope)
}

func TestHandleGeminiUpstreamError_RateLimitsOnDailyQuota(t *testing.T) {
	repo := &accountRepoRateLimitSpy{}
	svc := &GeminiMessagesCompatService{accountRepo: repo}
	account := &Account{
		ID:       7,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"tier_id":    "free",
		},
		Schedulable: true,
	}

	marked := svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"quota per day exceeded"}}`),
		"gemini-2.5-pro",
	)

	require.True(t, marked)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, int64(7), repo.account)
	require.True(t, repo.resetAt.After(time.Now().Add(-time.Second)))
}

func TestHandleGeminiUpstreamError_ServerOverloadThenDailyQuota(t *testing.T) {
	repo := &accountRepoRateLimitSpy{}
	svc := &GeminiMessagesCompatService{accountRepo: repo}
	account := &Account{
		ID:       9,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"tier_id":    "free",
		},
		Schedulable: true,
	}

	rateLimitMarked := false
	rateLimitMarked = svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"error":{"message":"no capacity available"}}`),
		"gemini-2.5-pro",
	)
	require.False(t, rateLimitMarked)
	require.Equal(t, 0, repo.calls)

	if !rateLimitMarked {
		rateLimitMarked = svc.handleGeminiUpstreamError(
			context.Background(),
			account,
			http.StatusTooManyRequests,
			http.Header{},
			[]byte(`{"error":{"message":"quota per day exceeded"}}`),
			"gemini-2.5-pro",
		)
	}

	require.True(t, rateLimitMarked)
	require.Equal(t, 1, repo.calls)
}

func TestHandleGeminiUpstreamError_RateLimitsOnWrappedOAuthDailyQuota(t *testing.T) {
	repo := &accountRepoRateLimitSpy{}
	svc := &GeminiMessagesCompatService{accountRepo: repo}
	account := &Account{
		ID:       10,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"oauth_type": "code_assist",
			"tier_id":    "free",
			"project_id": "demo-project",
		},
		Schedulable: true,
	}

	marked := svc.handleGeminiUpstreamError(
		context.Background(),
		account,
		http.StatusTooManyRequests,
		http.Header{},
		[]byte(`{"response":{"error":{"message":"requests per day exceeded"}}}`),
		"gemini-2.5-pro",
	)

	require.True(t, marked)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, 0, repo.modelRateLimitCalls)
}

func TestShouldFastFailoverGemini429_UsesInspectionBodyForWrappedOAuthResponse(t *testing.T) {
	account := &Account{Type: AccountTypeOAuth}
	body := gemini429InspectionBody(account, []byte(`{"response":{"error":{"details":[{"metadata":{"quotaResetDelay":"12s"}}]}}}`))

	fastFailover, reason := shouldFastFailoverGemini429(body)
	require.True(t, fastFailover)
	require.Equal(t, "quota_reset_delay", reason)
}

func TestParseGeminiRateLimitResetTime_WrappedOAuthResponse(t *testing.T) {
	now := time.Now().Unix()
	got := ParseGeminiRateLimitResetTime([]byte(`{"response":{"error":{"details":[{"metadata":{"quotaResetDelay":"12.345s"}}]}}}`))

	require.NotNil(t, got)
	delta := *got - now
	require.GreaterOrEqual(t, delta, int64(11))
	require.LessOrEqual(t, delta, int64(15))
}

func TestShouldProcessGemini429Mark(t *testing.T) {
	require.True(t, shouldProcessGemini429Mark(false, gemini429ClassUnknown, gemini429ClassUnknown))
	require.True(t, shouldProcessGemini429Mark(true, gemini429ClassUnknown, gemini429ClassDailyQuota))
	require.True(t, shouldProcessGemini429Mark(true, gemini429ClassRetryDelay, gemini429ClassDailyQuota))
	require.False(t, shouldProcessGemini429Mark(true, gemini429ClassDailyQuota, gemini429ClassRetryDelay))
	require.False(t, shouldProcessGemini429Mark(true, gemini429ClassDailyQuota, gemini429ClassUnknown))
}

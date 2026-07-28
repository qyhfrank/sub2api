package service

// 本文件由 gateway_service.go 纯移动拆分而来：Bedrock 上游转发（CC 兼容转换、
// 请求构建、错误处理与非流式响应）。仅做代码搬迁，无任何行为变更。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ApplyBedrockCCCompat 应用 Bedrock CC 兼容转换（渠道级模型映射后调用）
// 清理 body 中 Anthropic API 专有字段、修复 thinking/tool_use ID、过滤 beta token，
// 同时过滤 HTTP header 中的 anthropic-beta（防止 Passthrough 路径透传不支持的 token）。
func (s *GatewayService) ApplyBedrockCCCompat(c *gin.Context, body []byte, model string, account *Account, groupID *int64) []byte {
	if !s.isBedrockCCCompatEnabled(c.Request.Context(), account, groupID) {
		return body
	}
	body = sanitizeBedrockCCFields(body)
	body = sanitizeBedrockThinking(body, model)
	body = sanitizeBedrockToolUseIDs(body)
	body = sanitizeBedrockCCBetaTokens(body, model)
	// 过滤 HTTP header 中的 anthropic-beta，只保留 Bedrock 支持的 token
	if betaHeader := c.GetHeader("anthropic-beta"); betaHeader != "" {
		if filtered := ResolveBedrockBetaTokens(betaHeader, body, model); len(filtered) > 0 {
			c.Request.Header.Set("anthropic-beta", strings.Join(filtered, ", "))
		} else {
			c.Request.Header.Del("anthropic-beta")
		}
	}
	return body
}

// isBedrockCCCompatEnabled 检查渠道是否启用了 Bedrock CC 兼容模式
func (s *GatewayService) isBedrockCCCompatEnabled(ctx context.Context, account *Account, groupID *int64) bool {
	if groupID == nil || s.channelService == nil {
		return false
	}
	ch, err := s.channelService.GetChannelForGroup(ctx, *groupID)
	if err != nil || ch == nil {
		return false
	}
	return ch.IsBedrockCCCompatEnabled(account.Platform)
}

// forwardBedrock 转发请求到 AWS Bedrock
func (s *GatewayService) forwardBedrock(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *ParsedRequest,
	startTime time.Time,
) (*ForwardResult, error) {
	reqModel := parsed.Model
	reqStream := parsed.Stream
	body := parsed.Body.Bytes()

	target, err := ResolveBedrockInvocationTarget(account, reqModel)
	if err != nil {
		return nil, err
	}
	if target.InvocationModel != reqModel {
		logger.LegacyPrintf("service.gateway", "[Bedrock] Model mapping: %s -> %s (account: %s)", reqModel, target.InvocationModel, account.Name)
	}

	betaHeader := ""
	if c != nil && c.Request != nil {
		betaHeader = c.GetHeader("anthropic-beta")
	}

	// 准备请求体（注入 anthropic_version/anthropic_beta，移除 Bedrock 不支持的字段，清理 cache_control）
	betaTokens, err := s.resolveBedrockBetaTokensForRequest(ctx, account, betaHeader, body, target.InvocationModel)
	if err != nil {
		return nil, err
	}

	bedrockBody, err := PrepareBedrockRequestBodyWithTokens(body, target.InvocationModel, betaTokens, false)
	if err != nil {
		return nil, fmt.Errorf("prepare bedrock request body: %w", err)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	logger.LegacyPrintf("service.gateway", "[Bedrock] 命中 Bedrock 分支: account=%d name=%s model=%s->%s stream=%v",
		account.ID, account.Name, reqModel, target.InvocationModel, reqStream)

	attemptedRoutes := make(map[BedrockRouteKey]struct{})
	for {
		// 根据账号类型选择认证方式
		var signer *BedrockSigner
		var bedrockAPIKey string
		if account.IsBedrockAPIKey() {
			bedrockAPIKey = account.GetCredential("api_key")
			if bedrockAPIKey == "" {
				return nil, fmt.Errorf("api_key not found in bedrock credentials")
			}
		} else {
			signer, err = NewBedrockSignerFromAccountForRegion(account, target.RuntimeRegion)
			if err != nil {
				return nil, fmt.Errorf("create bedrock signer: %w", err)
			}
		}

		// 执行上游请求（含重试）
		resp, err := s.executeBedrockUpstream(ctx, c, account, bedrockBody, target.InvocationModel, target.RuntimeRegion, reqStream, signer, bedrockAPIKey, proxyURL)
		if err != nil {
			return nil, err
		}

		// 将 Bedrock 的 x-amzn-requestid 映射到 x-request-id，
		// 使通用错误处理函数（handleErrorResponse、handleRetryExhaustedError）能正确提取 AWS request ID。
		if awsReqID := resp.Header.Get("x-amzn-requestid"); awsReqID != "" && resp.Header.Get("x-request-id") == "" {
			resp.Header.Set("x-request-id", awsReqID)
		}

		// 错误/failover 处理
		if resp.StatusCode >= 400 {
			_, routeLocalQuota := inspectBedrockRouteLocalQuota429(resp)
			if routeLocalQuota && target.Policy.Mode == "all_routes" && target.RouteKey != nil {
				attemptedRoutes[*target.RouteKey] = struct{}{}
				markBedrockRouteCooldown(account, target, bedrockRouteLocalQuotaCooldownUntil(time.Now()))

				nextTarget, nextErr := ResolveBedrockInvocationTarget(account, reqModel)
				if nextErr == nil && nextTarget.RouteKey != nil {
					if _, seen := attemptedRoutes[*nextTarget.RouteKey]; !seen {
						_ = resp.Body.Close()
						target = nextTarget
						continue
					}
				}
			}

			defer func() { _ = resp.Body.Close() }()
			return s.handleBedrockUpstreamErrors(ctx, resp, c, account)
		}

		// Bedrock bypasses the generic Forward success path, so preserve the
		// upstream-accepted callback used to release serial scheduling locks.
		if parsed.OnUpstreamAccepted != nil {
			parsed.OnUpstreamAccepted()
		}

		defer func() { _ = resp.Body.Close() }()

		// 响应处理
		var usage *ClaudeUsage
		var firstTokenMs *int
		var clientDisconnect bool
		if reqStream {
			streamResult, err := s.handleBedrockStreamingResponse(ctx, resp, c, account, startTime, reqModel)
			if err != nil {
				return nil, err
			}
			usage = streamResult.usage
			firstTokenMs = streamResult.firstTokenMs
			clientDisconnect = streamResult.clientDisconnect
		} else {
			usage, err = s.handleBedrockNonStreamingResponse(ctx, resp, c, account)
			if err != nil {
				return nil, err
			}
		}
		if usage == nil {
			usage = &ClaudeUsage{}
		}

		return &ForwardResult{
			RequestID:        resp.Header.Get("x-amzn-requestid"),
			Usage:            *usage,
			Model:            reqModel,
			UpstreamModel:    target.InvocationModel,
			Stream:           reqStream,
			Duration:         time.Since(startTime),
			FirstTokenMs:     firstTokenMs,
			ClientDisconnect: clientDisconnect,
		}, nil
	}
}

func inspectBedrockRouteLocalQuota429(resp *http.Response) ([]byte, bool) {
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests || resp.Body == nil {
		return nil, false
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	if isKnownBedrockRouteLocalQuota429(resp, body) {
		return body, true
	}
	return body, false
}

func isKnownBedrockRouteLocalQuota429(resp *http.Response, body []byte) bool {
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		return false
	}

	errType := strings.ToLower(strings.TrimSpace(resp.Header.Get("x-amzn-errortype")))
	if strings.Contains(errType, "servicequotaexceededexception") {
		return true
	}

	msg := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	raw := strings.ToLower(strings.TrimSpace(string(body)))
	if strings.Contains(raw, "servicequotaexceededexception") {
		return true
	}
	if strings.Contains(msg, "daily quota") || strings.Contains(raw, "daily quota") {
		return true
	}
	if strings.Contains(msg, "service quota") || strings.Contains(raw, "service quota") {
		return true
	}
	if (strings.Contains(msg, "quota exceeded") || strings.Contains(raw, "quota exceeded")) &&
		(strings.Contains(msg, "region") || strings.Contains(raw, "region") || strings.Contains(msg, "daily") || strings.Contains(raw, "daily")) {
		return true
	}
	return false
}

func bedrockRouteLocalQuotaCooldownUntil(now time.Time) int64 {
	return now.Add(24 * time.Hour).Unix()
}

func markBedrockRouteCooldown(account *Account, target BedrockInvocationTarget, blockedUntil int64) {
	if target.RouteKey == nil || target.Policy.Mode != "all_routes" {
		return
	}
	routes := filterBedrockRoutesByScope(LookupBedrockRoutes(target.Support.CanonicalModel), target.Policy.Scope)
	if len(routes) == 0 {
		return
	}
	pool := runtimeBedrockRoutePools.getOrCreate(
		routePoolRegistryKey(account, target.Support.CanonicalModel, target.Policy, target.Support.RuntimeRegion, target.Support.InvocationModel),
		prioritizeBedrockRoutes(routes, target.Policy, target.Support.RuntimeRegion, target.Support.InvocationModel),
	)
	pool.MarkCooldown(*target.RouteKey, blockedUntil)
}

// executeBedrockUpstream 执行 Bedrock 上游请求（含重试逻辑）
func (s *GatewayService) executeBedrockUpstream(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	modelID string,
	region string,
	stream bool,
	signer *BedrockSigner,
	apiKey string,
	proxyURL string,
) (*http.Response, error) {
	var resp *http.Response
	var err error
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		var upstreamReq *http.Request
		if account.IsBedrockAPIKey() {
			upstreamReq, err = s.buildUpstreamRequestBedrockAPIKey(ctx, body, modelID, region, stream, apiKey)
		} else {
			upstreamReq, err = s.buildUpstreamRequestBedrock(ctx, body, modelID, region, stream, signer)
		}
		if err != nil {
			return nil, err
		}

		resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, nil)
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
				Kind:               "request_error",
				Message:            safeErr,
			})
			c.JSON(http.StatusBadGateway, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "upstream_error",
					"message": "Upstream request failed",
				},
			})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}

		if resp.StatusCode >= 400 && resp.StatusCode != 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
			if attempt < maxRetryAttempts {
				elapsed := time.Since(retryStart)
				if elapsed >= maxRetryElapsed {
					break
				}

				delay := retryBackoffDelay(attempt)
				remaining := maxRetryElapsed - elapsed
				if delay > remaining {
					delay = remaining
				}
				if delay <= 0 {
					break
				}

				respBody, _ := s.readUpstreamErrorBody(resp)
				_ = resp.Body.Close()
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
					Platform:           account.Platform,
					AccountID:          account.ID,
					AccountName:        account.Name,
					UpstreamStatusCode: resp.StatusCode,
					UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
					Kind:               "retry",
					Message:            extractUpstreamErrorMessage(respBody),
					Detail: func() string {
						if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
							return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
						}
						return ""
					}(),
				})
				logger.LegacyPrintf("service.gateway", "[Bedrock] account %d: upstream error %d, retry %d/%d after %v",
					account.ID, resp.StatusCode, attempt, maxRetryAttempts, delay)
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}
			break
		}

		break
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream request failed: empty response")
	}
	return resp, nil
}

// handleBedrockUpstreamErrors 处理 Bedrock 上游 4xx/5xx 错误（failover + 错误响应）
func (s *GatewayService) handleBedrockUpstreamErrors(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
) (*ForwardResult, error) {
	// retry exhausted + failover
	if s.shouldRetryUpstreamError(account, resp.StatusCode) {
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			respBody, _ := s.readUpstreamErrorBody(resp)
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))

			logger.LegacyPrintf("service.gateway", "[Bedrock] Upstream error (retry exhausted, failover): Account=%d(%s) Status=%d Body=%s",
				account.ID, account.Name, resp.StatusCode, truncateString(string(respBody), 1000))

			s.handleRetryExhaustedSideEffects(ctx, resp, account)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				Kind:               "retry_exhausted_failover",
				Message:            extractUpstreamErrorMessage(respBody),
			})
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return s.handleRetryExhaustedError(ctx, resp, c, account)
	}

	// non-retryable failover
	if s.shouldFailoverUpstreamError(resp.StatusCode) {
		respBody, _ := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		s.handleFailoverSideEffects(ctx, resp, account)
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			Kind:               "failover",
			Message:            extractUpstreamErrorMessage(respBody),
		})
		return nil, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           respBody,
			RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
		}
	}

	// other errors
	return s.handleErrorResponse(ctx, resp, c, account)
}

// buildUpstreamRequestBedrock 构建 Bedrock 上游请求
func (s *GatewayService) buildUpstreamRequestBedrock(
	ctx context.Context,
	body []byte,
	modelID string,
	region string,
	stream bool,
	signer *BedrockSigner,
) (*http.Request, error) {
	targetURL := BuildBedrockURL(region, modelID, stream)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// SigV4 签名
	if err := signer.SignRequest(ctx, req, body); err != nil {
		return nil, fmt.Errorf("sign bedrock request: %w", err)
	}

	return req, nil
}

// buildUpstreamRequestBedrockAPIKey 构建 Bedrock API Key (Bearer Token) 上游请求
func (s *GatewayService) buildUpstreamRequestBedrockAPIKey(
	ctx context.Context,
	body []byte,
	modelID string,
	region string,
	stream bool,
	apiKey string,
) (*http.Request, error) {
	targetURL := BuildBedrockURL(region, modelID, stream)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return req, nil
}

// handleBedrockNonStreamingResponse 处理 Bedrock 非流式响应
// Bedrock InvokeModel 非流式响应的 body 格式与 Claude API 兼容
func (s *GatewayService) handleBedrockNonStreamingResponse(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
) (*ClaudeUsage, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, anthropicTooLargeError)
	if err != nil {
		return nil, err
	}

	// 转换 Bedrock 特有的 amazon-bedrock-invocationMetrics 为标准 Anthropic usage 格式
	// 并移除该字段避免透传给客户端
	body = transformBedrockInvocationMetrics(body)

	usage := parseClaudeUsageFromResponseBody(body)

	c.Header("Content-Type", "application/json")
	if v := resp.Header.Get("x-amzn-requestid"); v != "" {
		c.Header("x-request-id", v)
	}
	c.Data(resp.StatusCode, "application/json", body)
	return usage, nil
}

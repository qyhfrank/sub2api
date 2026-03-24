package service

import (
	"testing"
	"time"
)

func TestGatewayServiceIsModelSupportedByAccount_BedrockDefaultMappingRestrictsModels(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region": "us-east-1",
		},
	}

	if !svc.isModelSupportedByAccount(account, "claude-sonnet-4-5") {
		t.Fatalf("expected default Bedrock alias to be supported")
	}

	if svc.isModelSupportedByAccount(account, "claude-3-5-sonnet-20241022") {
		t.Fatalf("expected unsupported alias to be rejected for Bedrock account")
	}
}

func TestGatewayServiceIsModelSupportedByAccount_BedrockCustomMappingStillActsAsAllowlist(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region": "eu-west-1",
			"model_mapping": map[string]any{
				"claude-sonnet-*": "claude-sonnet-4-6",
			},
		},
	}

	if !svc.isModelSupportedByAccount(account, "claude-sonnet-4-6") {
		t.Fatalf("expected matched custom mapping to be supported")
	}

	if !svc.isModelSupportedByAccount(account, "claude-opus-4-6") {
		t.Fatalf("expected default Bedrock alias fallback to remain supported")
	}

	if svc.isModelSupportedByAccount(account, "claude-3-5-sonnet-20241022") {
		t.Fatalf("expected unsupported model to still be rejected")
	}
}

func TestGatewayBedrockRoutePrecheckRejectsInvalidPolicy(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region":      "us-east-1",
			"aws_route_mode":  "single_route",
			"aws_route_scope": "mars",
		},
	}

	if svc.isModelSupportedByAccount(account, "claude-opus-4-6") {
		t.Fatalf("expected invalid routed Bedrock policy to fail precheck")
	}
}

func TestGatewayBedrockRoutePrecheckDoesNotAdvanceAllRoutesPool(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	svc := &GatewayService{}
	account := &Account{
		ID:       201,
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region":      "us-east-1",
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "us",
		},
	}

	if !svc.isModelSupportedByAccount(account, "claude-opus-4-6") {
		t.Fatalf("expected valid routed Bedrock policy to pass precheck")
	}

	target, err := ResolveBedrockInvocationTarget(account, "claude-opus-4-6")
	if err != nil {
		t.Fatalf("expected runtime invocation target after precheck: %v", err)
	}
	if target.RouteKey == nil || target.RouteKey.RuntimeRegion != "us-east-1" {
		t.Fatalf("expected precheck to leave first all_routes target intact, got %#v", target.RouteKey)
	}
}

func TestGatewayBedrockRoutePrecheckIgnoresAllRoutesCooldownState(t *testing.T) {
	runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}
	svc := &GatewayService{}
	account := &Account{
		ID:       202,
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_region":      "us-east-1",
			"aws_route_mode":  "all_routes",
			"aws_route_scope": "us",
		},
	}

	policy, err := ResolveBedrockRoutePolicy(account, "anthropic.claude-opus-4-6-v1")
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	routes := filterBedrockRoutesByScope(LookupBedrockRoutes("anthropic.claude-opus-4-6-v1"), policy.Scope)
	pool := runtimeBedrockRoutePools.getOrCreate(routePoolRegistryKey(account, "anthropic.claude-opus-4-6-v1", policy), routes)
	for _, route := range routes {
		pool.MarkCooldown(route.Key, time.Now().Add(time.Hour).Unix())
	}

	if !svc.isModelSupportedByAccount(account, "claude-opus-4-6") {
		t.Fatalf("expected precheck to validate config even when all routes are cooling down")
	}
}

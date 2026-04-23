package service

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type BedrockInvocationTarget struct {
	Support         BedrockModelSupport
	RuntimeRegion   string
	InvocationModel string
	RouteKey        *BedrockRouteKey
	Policy          BedrockRoutePolicy
	Legacy          bool
}

type BedrockRoutePool struct {
	routes    []BedrockRoute
	nextIndex int
	cooldowns map[BedrockRouteKey]int64
	mu        sync.Mutex
}

type bedrockRoutePoolRegistry struct {
	mu    sync.Mutex
	pools map[string]*BedrockRoutePool
}

var runtimeBedrockRoutePools = &bedrockRoutePoolRegistry{pools: make(map[string]*BedrockRoutePool)}

func NewBedrockRoutePool(routes []BedrockRoute) *BedrockRoutePool {
	copyRoutes := make([]BedrockRoute, len(routes))
	copy(copyRoutes, routes)
	return &BedrockRoutePool{
		routes:    copyRoutes,
		cooldowns: make(map[BedrockRouteKey]int64),
	}
}

func (p *BedrockRoutePool) SelectNextRoute(now int64) (BedrockRoute, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.nextRoute(now, true)
}

func (p *BedrockRoutePool) PeekNextRoute(now int64) (BedrockRoute, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.nextRoute(now, false)
}

func (p *BedrockRoutePool) nextRoute(now int64, advance bool) (BedrockRoute, bool) {
	if len(p.routes) == 0 {
		return BedrockRoute{}, false
	}
	start := p.nextIndex
	for i := 0; i < len(p.routes); i++ {
		idx := (start + i) % len(p.routes)
		route := p.routes[idx]
		if blockedUntil, ok := p.cooldowns[route.Key]; ok && now < blockedUntil {
			continue
		}
		if advance {
			p.nextIndex = (idx + 1) % len(p.routes)
		}
		return route, true
	}
	return BedrockRoute{}, false
}

func (p *BedrockRoutePool) MarkCooldown(key BedrockRouteKey, blockedUntil int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cooldowns[key] = blockedUntil
}

func ResolveBedrockInvocationTarget(account *Account, requestedModel string) (BedrockInvocationTarget, error) {
	return resolveBedrockInvocationTarget(account, requestedModel, true)
}

func ValidateBedrockInvocationTarget(account *Account, requestedModel string) error {
	_, err := PreviewBedrockInvocationTarget(account, requestedModel)
	return err
}

func PreviewBedrockInvocationTarget(account *Account, requestedModel string) (BedrockInvocationTarget, error) {
	return resolveBedrockInvocationTarget(account, requestedModel, false)
}

func resolveBedrockInvocationTarget(account *Account, requestedModel string, advancePool bool) (BedrockInvocationTarget, error) {
	support, ok := ResolveBedrockModelSupport(account, requestedModel)
	if !ok {
		return BedrockInvocationTarget{}, fmt.Errorf("unsupported bedrock model: %s", requestedModel)
	}

	policy, err := ResolveBedrockRoutePolicy(account, support.CanonicalModel)
	if err != nil {
		return BedrockInvocationTarget{}, err
	}
	if policy.Mode == "" {
		runtimeRegion := support.RuntimeRegion
		invocationModel := support.InvocationModel
		if !isRegionalBedrockModelID(invocationModel) {
			for _, route := range LookupBedrockRoutes(support.CanonicalModel) {
				if route.InvocationModel == invocationModel {
					runtimeRegion = route.Key.RuntimeRegion
					invocationModel = route.InvocationModel
					break
				}
			}
		}
		return BedrockInvocationTarget{
			Support:         support,
			RuntimeRegion:   runtimeRegion,
			InvocationModel: invocationModel,
			Policy:          policy,
			Legacy:          true,
		}, nil
	}

	routes := filterBedrockRoutesByScope(LookupBedrockRoutes(support.CanonicalModel), policy.Scope)
	if len(routes) == 0 {
		return BedrockInvocationTarget{}, fmt.Errorf("route catalog does not include %q for scope %q", support.CanonicalModel, policy.Scope)
	}

	var selected BedrockRoute
	switch policy.Mode {
	case "single_route":
		selected, err = selectSingleBedrockRoute(routes, policy, support.RuntimeRegion)
		if err != nil {
			return BedrockInvocationTarget{}, err
		}
	case "all_routes":
		selected, err = selectAllRoutesBedrockTarget(account, support.CanonicalModel, policy, routes, support.RuntimeRegion, support.InvocationModel, advancePool)
		if err != nil {
			return BedrockInvocationTarget{}, err
		}
	default:
		return BedrockInvocationTarget{}, fmt.Errorf("invalid aws_route_mode %q", policy.Mode)
	}

	routeKey := selected.Key
	return BedrockInvocationTarget{
		Support:         support,
		RuntimeRegion:   selected.Key.RuntimeRegion,
		InvocationModel: selected.InvocationModel,
		RouteKey:        &routeKey,
		Policy:          policy,
		Legacy:          false,
	}, nil
}

func selectSingleBedrockRoute(routes []BedrockRoute, policy BedrockRoutePolicy, preferredRuntimeRegion string) (BedrockRoute, error) {
	if policy.PreferredRegion != "" {
		for _, route := range routes {
			if route.Key.RuntimeRegion == policy.PreferredRegion {
				return route, nil
			}
		}
		return BedrockRoute{}, fmt.Errorf("no route matches aws_route_preferred_region %q", policy.PreferredRegion)
	}
	for _, route := range routes {
		if route.Key.RuntimeRegion == preferredRuntimeRegion {
			return route, nil
		}
	}
	return routes[0], nil
}

func selectAllRoutesBedrockTarget(account *Account, canonicalModel string, policy BedrockRoutePolicy, routes []BedrockRoute, baselineRuntimeRegion, baselineInvocationModel string, advancePool bool) (BedrockRoute, error) {
	pool := runtimeBedrockRoutePools.getOrCreate(routePoolRegistryKey(account, canonicalModel, policy, baselineRuntimeRegion, baselineInvocationModel), prioritizeBedrockRoutes(routes, policy, baselineRuntimeRegion, baselineInvocationModel))
	var (
		route BedrockRoute
		ok    bool
	)
	if advancePool {
		route, ok = pool.SelectNextRoute(time.Now().Unix())
	} else {
		route, ok = pool.PeekNextRoute(time.Now().Unix())
	}
	if !ok {
		return BedrockRoute{}, fmt.Errorf("no healthy route available for %q", canonicalModel)
	}
	return route, nil
}

func prioritizeBedrockRoutes(routes []BedrockRoute, policy BedrockRoutePolicy, baselineRuntimeRegion, baselineInvocationModel string) []BedrockRoute {
	ordered := make([]BedrockRoute, len(routes))
	copy(ordered, routes)
	targetRegion := policy.PreferredRegion
	if targetRegion == "" {
		targetRegion = baselineRuntimeRegion
	}
	targetInvocationModel := ""
	if policy.PreferredRegion == "" || baselineRuntimeRegion == targetRegion {
		targetInvocationModel = baselineInvocationModel
	}
	if !isRegionalBedrockModelID(targetInvocationModel) {
		targetInvocationModel = ""
	}
	if targetRegion == "" && targetInvocationModel == "" {
		return ordered
	}
	primaryExact := make([]BedrockRoute, 0, len(routes))
	primaryFamily := make([]BedrockRoute, 0, len(routes))
	preferredFamily := make([]BedrockRoute, 0, len(routes))
	preferred := make([]BedrockRoute, 0, len(routes))
	apacFallback := make([]BedrockRoute, 0, len(routes))
	remaining := make([]BedrockRoute, 0, len(routes))
	useAPACFallback := policy.PreferredRegion == "" && shouldUseBedrockAPACFallback(targetRegion)
	if useAPACFallback && !isRegionalBedrockModelID(targetInvocationModel) {
		targetInvocationModel = ""
	}
	preferredInvocationPrefix := ""
	if targetRegion != "" {
		preferredInvocationPrefix = BedrockCrossRegionPrefix(targetRegion) + "."
	}
	for _, route := range ordered {
		if targetInvocationModel != "" && route.InvocationModel == targetInvocationModel && route.Key.RuntimeRegion == targetRegion {
			primaryExact = append(primaryExact, route)
			continue
		}
		if targetInvocationModel != "" && route.InvocationModel == targetInvocationModel {
			primaryFamily = append(primaryFamily, route)
			continue
		}
		if preferredInvocationPrefix != "" && route.Key.RuntimeRegion == targetRegion && strings.HasPrefix(route.InvocationModel, preferredInvocationPrefix) {
			preferredFamily = append(preferredFamily, route)
			continue
		}
		if route.Key.RuntimeRegion == targetRegion {
			preferred = append(preferred, route)
			continue
		}
		if useAPACFallback && (route.Key.Scope == "au" || route.Key.Scope == "jp") {
			apacFallback = append(apacFallback, route)
			continue
		}
		remaining = append(remaining, route)
	}
	return append(append(append(append(append(primaryExact, primaryFamily...), preferredFamily...), preferred...), apacFallback...), remaining...)
}

func shouldUseBedrockAPACFallback(region string) bool {
	if !strings.HasPrefix(region, "ap-") {
		return false
	}
	switch region {
	case "ap-northeast-1", "ap-southeast-2":
		return false
	default:
		return true
	}
}

func routePoolRegistryKey(account *Account, canonicalModel string, policy BedrockRoutePolicy, baselineRuntimeRegion, baselineInvocationModel string) string {
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	routes := filterBedrockRoutesByScope(LookupBedrockRoutes(canonicalModel), policy.Scope)
	ordered := prioritizeBedrockRoutes(routes, policy, baselineRuntimeRegion, baselineInvocationModel)
	orderSignature := ""
	for _, route := range ordered {
		orderSignature += "|" + route.Key.Scope + "|" + route.Key.RuntimeRegion + "|" + route.InvocationModel
	}
	return fmt.Sprintf("%d/%s/%s/%s", accountID, canonicalModel, policy.Mode, orderSignature)
}

func (r *bedrockRoutePoolRegistry) getOrCreate(key string, routes []BedrockRoute) *BedrockRoutePool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pool, ok := r.pools[key]; ok {
		return pool
	}
	pool := NewBedrockRoutePool(routes)
	r.pools[key] = pool
	return pool
}

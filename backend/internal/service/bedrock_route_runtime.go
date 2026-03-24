package service

import (
	"fmt"
	"sync"
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
	routes     []BedrockRoute
	nextIndex  int
	cooldowns  map[BedrockRouteKey]int64
	mu         sync.Mutex
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
		p.nextIndex = (idx + 1) % len(p.routes)
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
	support, ok := ResolveBedrockModelSupport(account, requestedModel)
	if !ok {
		return BedrockInvocationTarget{}, fmt.Errorf("unsupported bedrock model: %s", requestedModel)
	}

	policy, err := ResolveBedrockRoutePolicy(account, support.CanonicalModel)
	if err != nil {
		return BedrockInvocationTarget{}, err
	}
	if policy.Mode == "" {
		return BedrockInvocationTarget{
			Support:         support,
			RuntimeRegion:   support.RuntimeRegion,
			InvocationModel: support.InvocationModel,
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
		selected, err = selectAllRoutesBedrockTarget(account, support.CanonicalModel, policy, routes)
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

func selectAllRoutesBedrockTarget(account *Account, canonicalModel string, policy BedrockRoutePolicy, routes []BedrockRoute) (BedrockRoute, error) {
	pool := runtimeBedrockRoutePools.getOrCreate(routePoolRegistryKey(account, canonicalModel, policy), routes)
	route, ok := pool.SelectNextRoute(0)
	if !ok {
		return BedrockRoute{}, fmt.Errorf("no healthy route available for %q", canonicalModel)
	}
	if policy.PreferredRegion == "" {
		return route, nil
	}
	for _, candidate := range routes {
		if candidate.Key.RuntimeRegion == policy.PreferredRegion {
			return candidate, nil
		}
	}
	return route, nil
}

func routePoolRegistryKey(account *Account, canonicalModel string, policy BedrockRoutePolicy) string {
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	return fmt.Sprintf("%d/%s/%s/%s", accountID, canonicalModel, policy.Mode, policy.Scope)
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

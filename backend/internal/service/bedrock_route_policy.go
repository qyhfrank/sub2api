package service

import (
	"fmt"
	"strings"
)

type BedrockRoutePolicy struct {
	Mode            string
	Scope           string
	PreferredRegion string
}

const bedrockRouteScopeOnDemand = "on_demand"

func ResolveBedrockRoutePolicy(account *Account, canonicalModel string) (BedrockRoutePolicy, error) {
	if account == nil {
		return BedrockRoutePolicy{}, nil
	}

	mode := strings.TrimSpace(account.GetCredential("aws_route_mode"))
	if mode == "" {
		return BedrockRoutePolicy{}, nil
	}
	if mode != "single_route" && mode != "all_routes" {
		return BedrockRoutePolicy{}, fmt.Errorf("invalid aws_route_mode %q", mode)
	}
	if account.GetCredential("aws_force_global") == "true" {
		return BedrockRoutePolicy{}, fmt.Errorf("aws_force_global conflicts with aws_route_mode")
	}

	routes := LookupBedrockRoutes(canonicalModel)
	if len(routes) == 0 {
		return BedrockRoutePolicy{}, fmt.Errorf("route catalog does not include %q", canonicalModel)
	}

	rawScope := strings.TrimSpace(account.GetCredential("aws_route_scope"))
	scope, err := normalizeBedrockRoutePolicyScope(rawScope, mode)
	if err != nil {
		return BedrockRoutePolicy{}, err
	}

	filtered := filterBedrockRoutesByScope(routes, scope)
	if len(filtered) == 0 {
		return BedrockRoutePolicy{}, fmt.Errorf("invalid aws_route_scope %q for %q", rawScope, canonicalModel)
	}

	preferredRegion := strings.TrimSpace(account.GetCredential("aws_route_preferred_region"))
	if preferredRegion != "" && !bedrockRoutesContainRegion(filtered, preferredRegion) {
		return BedrockRoutePolicy{}, fmt.Errorf("invalid aws_route_preferred_region %q for %q", preferredRegion, canonicalModel)
	}

	return BedrockRoutePolicy{
		Mode:            mode,
		Scope:           scope,
		PreferredRegion: preferredRegion,
	}, nil
}

func normalizeBedrockRoutePolicyScope(rawScope, mode string) (string, error) {
	scope := strings.TrimSpace(rawScope)
	if scope == "" {
		if mode == "single_route" {
			return "", fmt.Errorf("aws_route_scope is required for single_route")
		}
		return "", nil
	}
	if scope == bedrockRouteScopeOnDemand {
		return bedrockRouteScopeOnDemand, nil
	}
	return scope, nil
}

func filterBedrockRoutesByScope(routes []BedrockRoute, scope string) []BedrockRoute {
	if scope == bedrockRouteScopeOnDemand {
		out := make([]BedrockRoute, 0, len(routes))
		for _, route := range routes {
			if route.Key.Scope == "" {
				out = append(out, route)
			}
		}
		return out
	}
	if scope == "" {
		out := make([]BedrockRoute, len(routes))
		copy(out, routes)
		return out
	}
	out := make([]BedrockRoute, 0, len(routes))
	for _, route := range routes {
		if route.Key.Scope == scope {
			out = append(out, route)
		}
	}
	return out
}

func bedrockRoutesContainRegion(routes []BedrockRoute, region string) bool {
	for _, route := range routes {
		if route.Key.RuntimeRegion == region {
			return true
		}
	}
	return false
}

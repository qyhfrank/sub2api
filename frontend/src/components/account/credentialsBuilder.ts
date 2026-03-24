export function applyInterceptWarmup(
  credentials: Record<string, unknown>,
  enabled: boolean,
  mode: 'create' | 'edit'
): void {
  if (enabled) {
    credentials.intercept_warmup_requests = true
  } else if (mode === 'edit') {
    delete credentials.intercept_warmup_requests
  }
}

export type BedrockRouteMode = 'off' | 'all_routes'

interface BedrockRouteConfigInput {
  mode: BedrockRouteMode
  region: string
  forceGlobal: boolean
  routeScope: string
  preferredRegion: string
}

export function applyBedrockRouteCredentials(
  credentials: Record<string, unknown>,
  config: BedrockRouteConfigInput
): void {
  const region = config.region.trim() || 'us-east-1'
  const routeScope = config.routeScope.trim()
  const preferredRegion = config.preferredRegion.trim()

  credentials.aws_region = region

  if (config.mode === 'all_routes') {
    credentials.aws_route_mode = 'all_routes'
    if (routeScope) {
      credentials.aws_route_scope = routeScope
    } else {
      delete credentials.aws_route_scope
    }
    if (preferredRegion) {
      credentials.aws_route_preferred_region = preferredRegion
    } else {
      delete credentials.aws_route_preferred_region
    }
    delete credentials.aws_force_global
    return
  }

  if (config.forceGlobal) {
    credentials.aws_force_global = 'true'
  } else {
    delete credentials.aws_force_global
  }
  delete credentials.aws_route_mode
  delete credentials.aws_route_scope
  delete credentials.aws_route_preferred_region
}

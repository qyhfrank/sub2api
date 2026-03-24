import { describe, it, expect } from 'vitest'
import { applyBedrockRouteCredentials, applyInterceptWarmup } from '../credentialsBuilder'

describe('applyInterceptWarmup', () => {
  it('create + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, true, 'create')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('create + enabled=false: should not add the field', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, false, 'create')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, true, 'edit')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('edit + enabled=false + field exists: should delete the field', () => {
    const creds: Record<string, unknown> = { api_key: 'sk', intercept_warmup_requests: true }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=false + field absent: should not throw', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('should not affect other fields', () => {
    const creds: Record<string, unknown> = {
      api_key: 'sk',
      base_url: 'url',
      intercept_warmup_requests: true
    }
    applyInterceptWarmup(creds, false, 'edit')
    expect(creds.api_key).toBe('sk')
    expect(creds.base_url).toBe('url')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })
})

describe('applyBedrockRouteCredentials', () => {
  it('off mode keeps legacy region and clears route keys', () => {
    const creds: Record<string, unknown> = {
      aws_region: 'eu-west-1',
      aws_route_mode: 'all_routes',
      aws_route_scope: 'us',
      aws_route_preferred_region: 'us-east-1'
    }

    applyBedrockRouteCredentials(creds, {
      mode: 'off',
      region: 'eu-west-1',
      forceGlobal: true,
      routeScope: '',
      preferredRegion: ''
    })

    expect(creds.aws_region).toBe('eu-west-1')
    expect(creds.aws_force_global).toBe('true')
    expect('aws_route_mode' in creds).toBe(false)
    expect('aws_route_scope' in creds).toBe(false)
    expect('aws_route_preferred_region' in creds).toBe(false)
  })

  it('all_routes mode preserves region, clears force_global, and writes route keys', () => {
    const creds: Record<string, unknown> = {
      aws_region: 'us-east-2',
      aws_force_global: 'true'
    }

    applyBedrockRouteCredentials(creds, {
      mode: 'all_routes',
      region: 'us-east-2',
      forceGlobal: true,
      routeScope: 'eu',
      preferredRegion: 'eu-central-1'
    })

    expect(creds.aws_region).toBe('us-east-2')
    expect(creds.aws_route_mode).toBe('all_routes')
    expect(creds.aws_route_scope).toBe('eu')
    expect(creds.aws_route_preferred_region).toBe('eu-central-1')
    expect('aws_force_global' in creds).toBe(false)
  })
})

package service

import "strings"

type BedrockRouteKey struct {
	CanonicalModel string
	Scope          string
	RuntimeRegion  string
}

type BedrockRoute struct {
	Key             BedrockRouteKey
	InvocationModel string
}

var bedrockRouteCatalog = map[string][]BedrockRoute{
	"anthropic.claude-haiku-4-5-20251001-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-haiku-4-5-20251001-v1:0", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-haiku-4-5-20251001-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-haiku-4-5-20251001-v1:0", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-haiku-4-5-20251001-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-haiku-4-5-20251001-v1:0", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-haiku-4-5-20251001-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-haiku-4-5-20251001-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-haiku-4-5-20251001-v1:0"},
	},
	"anthropic.claude-opus-4-1-20250805-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-1-20250805-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-1-20250805-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-1-20250805-v1:0", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-opus-4-1-20250805-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-1-20250805-v1:0", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-opus-4-1-20250805-v1:0"},
	},
	"anthropic.claude-opus-4-20250514-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-20250514-v1:0", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-opus-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-20250514-v1:0", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-opus-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-20250514-v1:0", Scope: "jp", RuntimeRegion: "ap-northeast-1"}, InvocationModel: "jp.anthropic.claude-opus-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-20250514-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-20250514-v1:0"},
	},
	"anthropic.claude-opus-4-5-20251101-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "", RuntimeRegion: "eu-west-2"}, InvocationModel: "anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-opus-4-5-20251101-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-5-20251101-v1:0", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-opus-4-5-20251101-v1:0"},
	},
	"anthropic.claude-opus-4-6-v1": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "", RuntimeRegion: "eu-west-2"}, InvocationModel: "anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-opus-4-6-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-6-v1", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-opus-4-6-v1"},
	},
	"anthropic.claude-opus-4-7-v1": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "", RuntimeRegion: "eu-west-2"}, InvocationModel: "anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-opus-4-7-v1"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-opus-4-7-v1", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-opus-4-7-v1"},
	},
	"anthropic.claude-sonnet-4-20250514-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-20250514-v1:0", Scope: "jp", RuntimeRegion: "ap-northeast-1"}, InvocationModel: "jp.anthropic.claude-sonnet-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-20250514-v1:0", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-sonnet-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-20250514-v1:0", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-sonnet-4-20250514-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-20250514-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-sonnet-4-20250514-v1:0"},
	},
	"anthropic.claude-sonnet-4-5-20250929-v1:0": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "jp", RuntimeRegion: "ap-northeast-1"}, InvocationModel: "jp.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-sonnet-4-5-20250929-v1:0"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-5-20250929-v1:0", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-sonnet-4-5-20250929-v1:0"},
	},
	"anthropic.claude-sonnet-4-6": {
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "", RuntimeRegion: "eu-west-2"}, InvocationModel: "anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "au", RuntimeRegion: "ap-southeast-2"}, InvocationModel: "au.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "eu", RuntimeRegion: "eu-central-1"}, InvocationModel: "eu.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "global", RuntimeRegion: "us-east-1"}, InvocationModel: "global.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "jp", RuntimeRegion: "ap-northeast-1"}, InvocationModel: "jp.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "us", RuntimeRegion: "us-east-1"}, InvocationModel: "us.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "us", RuntimeRegion: "us-east-2"}, InvocationModel: "us.anthropic.claude-sonnet-4-6"},
		{Key: BedrockRouteKey{CanonicalModel: "anthropic.claude-sonnet-4-6", Scope: "us", RuntimeRegion: "us-west-2"}, InvocationModel: "us.anthropic.claude-sonnet-4-6"},
	},
}

func LookupBedrockRoutes(modelID string) []BedrockRoute {
	canonical := CanonicalBedrockModelID(strings.TrimSpace(modelID))
	routes, ok := bedrockRouteCatalog[canonical]
	if !ok {
		return nil
	}
	out := make([]BedrockRoute, len(routes))
	copy(out, routes)
	return out
}

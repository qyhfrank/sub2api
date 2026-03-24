"""Async HTTP client for Sub2API admin endpoints."""

from typing import Any

import httpx

from sub2api_mcp.config import configure, get_config, reset_config


def _auth_headers(api_token: str) -> dict[str, str]:
    """Build auth headers based on token format.

    Tokens starting with 'admin-' use the x-api-key header;
    all others use Authorization: Bearer.
    """
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if not api_token:
        return headers
    if api_token.startswith("admin-"):
        headers["x-api-key"] = api_token
    else:
        headers["Authorization"] = f"Bearer {api_token}"
    return headers


def _client() -> httpx.AsyncClient:
    config = get_config()
    return httpx.AsyncClient(
        base_url=config.base_url,
        headers=_auth_headers(config.api_token),
        timeout=config.timeout,
    )


async def api_get(path: str, params: dict[str, Any] | None = None) -> dict:
    async with _client() as client:
        resp = await client.get(path, params=params)
        resp.raise_for_status()
        return resp.json()


async def api_post(path: str, json: dict[str, Any] | None = None) -> dict:
    async with _client() as client:
        resp = await client.post(path, json=json)
        resp.raise_for_status()
        return resp.json()


async def api_put(
    path: str,
    json: dict[str, Any] | None = None,
    params: dict[str, Any] | None = None,
) -> dict:
    async with _client() as client:
        resp = await client.put(path, json=json, params=params)
        resp.raise_for_status()
        return resp.json()


async def api_patch(path: str, json: dict[str, Any] | None = None) -> dict:
    async with _client() as client:
        resp = await client.patch(path, json=json)
        resp.raise_for_status()
        return resp.json()


async def api_delete(path: str, params: dict[str, Any] | None = None) -> dict:
    async with _client() as client:
        resp = await client.delete(path, params=params)
        resp.raise_for_status()
        return resp.json()

"""Async HTTP client for Sub2API admin endpoints."""

import os
from typing import Any

import httpx

BASE_URL = os.environ.get("SUB2API_BASE_URL", "http://localhost:20165")
API_TOKEN = os.environ.get("SUB2API_TOKEN", "")


def _auth_headers() -> dict[str, str]:
    """Build auth headers based on token format.

    Tokens starting with 'admin-' use the x-api-key header;
    all others use Authorization: Bearer.
    """
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if API_TOKEN.startswith("admin-"):
        headers["x-api-key"] = API_TOKEN
    else:
        headers["Authorization"] = f"Bearer {API_TOKEN}"
    return headers


def _client() -> httpx.AsyncClient:
    return httpx.AsyncClient(
        base_url=BASE_URL,
        headers=_auth_headers(),
        timeout=30.0,
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

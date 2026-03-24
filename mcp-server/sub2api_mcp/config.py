"""Runtime configuration shared by CLI, MCP transport, and HTTP calls."""

from __future__ import annotations

from contextlib import contextmanager
import os
from dataclasses import dataclass

DEFAULT_BASE_URL = "http://localhost:20165"
DEFAULT_TIMEOUT = 30.0

_runtime_base_url: str | None = None
_runtime_api_token: str | None = None
_runtime_timeout: float | None = None


@dataclass(frozen=True)
class RuntimeConfig:
    base_url: str
    api_token: str
    timeout: float


def _normalize_base_url(base_url: str) -> str:
    value = base_url.strip()
    if not value:
        return DEFAULT_BASE_URL
    return value.rstrip("/")


def _normalize_timeout(timeout: float | str | None) -> float:
    if timeout is None:
        return DEFAULT_TIMEOUT
    value = float(timeout)
    if value <= 0:
        raise ValueError("timeout must be greater than zero")
    return value


def configure(
    *,
    base_url: str | None = None,
    api_token: str | None = None,
    timeout: float | None = None,
) -> None:
    global _runtime_base_url, _runtime_api_token, _runtime_timeout

    if base_url is not None:
        _runtime_base_url = _normalize_base_url(base_url)
    if api_token is not None:
        _runtime_api_token = api_token.strip()
    if timeout is not None:
        _runtime_timeout = _normalize_timeout(timeout)


def reset_config() -> None:
    global _runtime_base_url, _runtime_api_token, _runtime_timeout
    _runtime_base_url = None
    _runtime_api_token = None
    _runtime_timeout = None


@contextmanager
def override_config(
    *,
    base_url: str | None = None,
    api_token: str | None = None,
    timeout: float | None = None,
):
    global _runtime_base_url, _runtime_api_token, _runtime_timeout

    snapshot = (_runtime_base_url, _runtime_api_token, _runtime_timeout)
    configure(base_url=base_url, api_token=api_token, timeout=timeout)
    try:
        yield
    finally:
        _runtime_base_url, _runtime_api_token, _runtime_timeout = snapshot


def get_config() -> RuntimeConfig:
    base_url = _runtime_base_url
    if base_url is None:
        base_url = _normalize_base_url(os.environ.get("SUB2API_BASE_URL", DEFAULT_BASE_URL))

    api_token = _runtime_api_token
    if api_token is None:
        api_token = os.environ.get("SUB2API_TOKEN", "").strip()

    timeout = _runtime_timeout
    if timeout is None:
        timeout = _normalize_timeout(os.environ.get("SUB2API_TIMEOUT", DEFAULT_TIMEOUT))

    return RuntimeConfig(base_url=base_url, api_token=api_token, timeout=timeout)

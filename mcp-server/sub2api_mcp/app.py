"""Application factory for Sub2API MCP surfaces."""

from __future__ import annotations

from fastmcp import FastMCP

from sub2api_mcp.tools import register_all_tools

_INSTRUCTIONS = (
    "MCP server for managing a Sub2API LLM gateway platform. "
    "Provides tools for account, user, API key, group, proxy, dashboard, "
    "operations, system, and settings management. "
    "Set SUB2API_BASE_URL and SUB2API_TOKEN environment variables "
    "before connecting."
)

_mcp_instance: FastMCP | None = None


def create_mcp() -> FastMCP:
    """Create a FastMCP app with the full Sub2API tool registry."""
    mcp = FastMCP("Sub2API Manager", instructions=_INSTRUCTIONS)
    register_all_tools(mcp)
    return mcp


def get_mcp() -> FastMCP:
    """Return a cached FastMCP app instance."""
    global _mcp_instance
    if _mcp_instance is None:
        _mcp_instance = create_mcp()
    return _mcp_instance

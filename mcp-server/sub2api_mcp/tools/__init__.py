"""Tool registry for the Sub2API MCP package."""

from typing import TYPE_CHECKING

from sub2api_mcp.tool_catalog import call_tool, get_tool_spec, list_tool_specs

if TYPE_CHECKING:
    from fastmcp import FastMCP

from .accounts import register_tools as register_account_tools
from .users import register_tools as register_user_tools
from .groups import register_tools as register_group_tools
from .api_keys import register_tools as register_api_key_tools
from .dashboard import register_tools as register_dashboard_tools
from .ops import register_tools as register_ops_tools
from .system import register_tools as register_system_tools
from .settings import register_tools as register_settings_tools
from .misc import register_tools as register_misc_tools
from .proxies import register_tools as register_proxy_tools

TOOL_REGISTRARS = [
    register_account_tools,
    register_user_tools,
    register_group_tools,
    register_api_key_tools,
    register_dashboard_tools,
    register_ops_tools,
    register_system_tools,
    register_settings_tools,
    register_misc_tools,
    register_proxy_tools,
]


def register_all_tools(mcp: "FastMCP") -> "FastMCP":
    """Register every enabled tool module on the provided MCP server."""
    for register in TOOL_REGISTRARS:
        register(mcp)
    return mcp


__all__ = [
    "TOOL_REGISTRARS",
    "call_tool",
    "get_tool_spec",
    "list_tool_specs",
    "register_all_tools",
]

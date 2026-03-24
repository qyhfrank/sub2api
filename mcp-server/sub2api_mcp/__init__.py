"""Sub2API MCP package exports."""

from sub2api_mcp.app import create_mcp, get_mcp
from sub2api_mcp.server import main

mcp = get_mcp()

__all__ = ["create_mcp", "get_mcp", "main", "mcp"]

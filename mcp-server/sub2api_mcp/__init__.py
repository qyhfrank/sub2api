"""Sub2API MCP Server -- manage accounts, users, groups, and operations."""

from fastmcp import FastMCP

mcp = FastMCP(
    "Sub2API Manager",
    instructions=(
        "MCP server for managing a Sub2API LLM gateway platform. "
        "Provides tools for account, user, API key, group, dashboard, "
        "operations, system, and settings management. "
        "Set SUB2API_BASE_URL and SUB2API_TOKEN environment variables "
        "before connecting."
    ),
)

# Tool modules are imported and registered below.
# To enable/disable modules, comment or uncomment the corresponding lines.
# Keep total registered tools under ~60 to stay within Claude API payload limits.

from sub2api_mcp.tools.accounts import register_tools as register_account_tools
from sub2api_mcp.tools.users import register_tools as register_user_tools
from sub2api_mcp.tools.groups import register_tools as register_group_tools
from sub2api_mcp.tools.api_keys import register_tools as register_api_key_tools
from sub2api_mcp.tools.dashboard import register_tools as register_dashboard_tools
from sub2api_mcp.tools.ops import register_tools as register_ops_tools
from sub2api_mcp.tools.system import register_tools as register_system_tools
from sub2api_mcp.tools.settings import register_tools as register_settings_tools
from sub2api_mcp.tools.misc import register_tools as register_misc_tools

register_account_tools(mcp)   # 16 tools
register_user_tools(mcp)      # 8 tools
register_group_tools(mcp)     # 7 tools
register_api_key_tools(mcp)   # 4 tools
register_dashboard_tools(mcp) # 6 tools
register_ops_tools(mcp)       # 12 tools
register_system_tools(mcp)    # 5 tools
register_settings_tools(mcp)  # 2 tools
register_misc_tools(mcp)      # 2 tools
# Total: 62 tools


def main():
    mcp.run()


if __name__ == "__main__":
    main()

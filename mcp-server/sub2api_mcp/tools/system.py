"""System management tools -- version, updates, restart."""

from sub2api_mcp.client import api_get, api_post


def register_tools(mcp):
    @mcp.tool()
    async def get_version() -> dict:
        """Get the current Sub2API version information."""
        return await api_get("/api/v1/admin/system/version")

    @mcp.tool()
    async def check_updates() -> dict:
        """Check if a newer version of Sub2API is available."""
        return await api_get("/api/v1/admin/system/check-updates")

    @mcp.tool()
    async def perform_update() -> dict:
        """Perform a system update to the latest available version."""
        return await api_post("/api/v1/admin/system/update")

    @mcp.tool()
    async def rollback_update() -> dict:
        """Roll back the last system update."""
        return await api_post("/api/v1/admin/system/rollback")

    @mcp.tool()
    async def restart_service() -> dict:
        """Restart the Sub2API service."""
        return await api_post("/api/v1/admin/system/restart")

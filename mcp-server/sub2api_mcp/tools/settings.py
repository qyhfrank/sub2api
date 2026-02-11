"""Settings tools (read-only -- write endpoints are full-replacement and not exposed)."""

from sub2api_mcp.client import api_get


def register_tools(mcp):
    @mcp.tool()
    async def get_settings() -> dict:
        """Get all current system settings."""
        return await api_get("/api/v1/admin/settings")

    @mcp.tool()
    async def get_stream_timeout_settings() -> dict:
        """Get stream timeout configuration."""
        return await api_get("/api/v1/admin/settings/stream-timeout")

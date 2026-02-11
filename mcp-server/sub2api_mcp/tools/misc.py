"""Miscellaneous tools -- health check, error passthrough rules."""

from sub2api_mcp.client import api_get


def register_tools(mcp):
    @mcp.tool()
    async def health_check() -> dict:
        """Check the health status of the Sub2API service."""
        return await api_get("/health")

    @mcp.tool()
    async def list_error_passthrough_rules(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List error passthrough rules.

        Args:
            page: Page number.
            page_size: Number of entries per page.
        """
        return await api_get(
            "/api/v1/admin/error-passthrough-rules",
            params={"page": page, "page_size": page_size},
        )

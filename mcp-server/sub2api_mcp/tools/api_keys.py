"""API key management tools for Sub2API."""

from sub2api_mcp.client import api_get, api_post


def register_tools(mcp):
    @mcp.tool()
    async def list_user_api_keys(
        user_id: int,
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List API keys for a specific user."""
        params = {"page": page, "page_size": page_size}
        return await api_get(f"/api/v1/admin/users/{user_id}/api-keys", params=params)

    @mcp.tool()
    async def list_group_api_keys(
        group_id: int,
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List API keys associated with a specific group."""
        params = {"page": page, "page_size": page_size}
        return await api_get(f"/api/v1/admin/groups/{group_id}/api-keys", params=params)

    @mcp.tool()
    async def get_api_key_usage_trend(
        top_n: int = 10,
        start_date: str = "",
        end_date: str = "",
        granularity: str = "day",
    ) -> dict:
        """Get usage trend data for top API keys."""
        params: dict = {"top_n": top_n, "granularity": granularity}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        return await api_get("/api/v1/admin/dashboard/api-keys-trend", params=params)

    @mcp.tool()
    async def get_batch_api_keys_usage(api_key_ids: list[int]) -> dict:
        """Get usage data for a batch of API keys by their IDs."""
        return await api_post(
            "/api/v1/admin/dashboard/api-keys-usage",
            json={"api_key_ids": api_key_ids},
        )

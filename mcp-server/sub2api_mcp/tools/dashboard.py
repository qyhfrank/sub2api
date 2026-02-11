"""Dashboard analytics and usage trend tools."""

from sub2api_mcp.client import api_get, api_post


def register_tools(mcp):
    @mcp.tool()
    async def get_dashboard_stats() -> dict:
        """Get overall dashboard statistics (totals, summaries)."""
        return await api_get("/api/v1/admin/dashboard/stats")

    @mcp.tool()
    async def get_realtime_metrics() -> dict:
        """Get real-time dashboard metrics (active requests, recent throughput)."""
        return await api_get("/api/v1/admin/dashboard/realtime")

    @mcp.tool()
    async def get_usage_trend(
        start_date: str = "",
        end_date: str = "",
        granularity: str = "day",
        user_id: int | None = None,
        api_key_id: int | None = None,
        model: str = "",
        account_id: int | None = None,
        group_id: int | None = None,
    ) -> dict:
        """Get usage trend data over a time range with optional filters.

        Args:
            start_date: Start date (YYYY-MM-DD). Defaults to 7 days ago.
            end_date: End date (YYYY-MM-DD). Defaults to today.
            granularity: Time granularity - "hour", "day", or "month".
            user_id: Filter by user ID.
            api_key_id: Filter by API key ID.
            model: Filter by model name.
            account_id: Filter by account ID.
            group_id: Filter by group ID.
        """
        params = {"granularity": granularity}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        if user_id is not None:
            params["user_id"] = user_id
        if api_key_id is not None:
            params["api_key_id"] = api_key_id
        if model:
            params["model"] = model
        if account_id is not None:
            params["account_id"] = account_id
        if group_id is not None:
            params["group_id"] = group_id
        return await api_get("/api/v1/admin/dashboard/trend", params=params)

    @mcp.tool()
    async def get_model_stats(
        start_date: str = "",
        end_date: str = "",
    ) -> dict:
        """Get per-model usage statistics over a time range.

        Args:
            start_date: Start date (YYYY-MM-DD). Defaults to 7 days ago.
            end_date: End date (YYYY-MM-DD). Defaults to today.
        """
        params = {}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        return await api_get("/api/v1/admin/dashboard/models", params=params)

    @mcp.tool()
    async def get_user_usage_trend(
        top_n: int = 10,
        start_date: str = "",
        end_date: str = "",
        granularity: str = "day",
    ) -> dict:
        """Get usage trends for top N users.

        Args:
            top_n: Number of top users to include.
            start_date: Start date (YYYY-MM-DD).
            end_date: End date (YYYY-MM-DD).
            granularity: Time granularity - "hour", "day", or "month".
        """
        params = {"top_n": top_n, "granularity": granularity}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        return await api_get("/api/v1/admin/dashboard/users-trend", params=params)

    @mcp.tool()
    async def get_batch_users_usage(user_ids: list[int]) -> dict:
        """Get usage data for a batch of users by their IDs.

        Args:
            user_ids: List of user IDs to query.
        """
        return await api_post(
            "/api/v1/admin/dashboard/users-usage",
            json={"user_ids": user_ids},
        )

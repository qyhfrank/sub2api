"""Operations monitoring tools -- concurrency, errors, alerts, traffic."""

from sub2api_mcp.client import api_get


def register_tools(mcp):
    @mcp.tool()
    async def get_concurrency_stats() -> dict:
        """Get current concurrency statistics across all accounts."""
        return await api_get("/api/v1/admin/ops/concurrency")

    @mcp.tool()
    async def get_account_availability() -> dict:
        """Get account availability overview (healthy, degraded, down)."""
        return await api_get("/api/v1/admin/ops/account-availability")

    @mcp.tool()
    async def get_realtime_traffic_summary() -> dict:
        """Get real-time traffic summary (requests per second, latency)."""
        return await api_get("/api/v1/admin/ops/realtime-traffic")

    @mcp.tool()
    async def list_alert_rules() -> dict:
        """List all configured alert rules."""
        return await api_get("/api/v1/admin/ops/alert-rules")

    @mcp.tool()
    async def list_alert_events(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List alert events (triggered alerts).

        Args:
            page: Page number.
            page_size: Number of events per page.
        """
        return await api_get(
            "/api/v1/admin/ops/alert-events",
            params={"page": page, "page_size": page_size},
        )

    @mcp.tool()
    async def get_error_logs(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """Get error logs.

        Args:
            page: Page number.
            page_size: Number of entries per page.
        """
        return await api_get(
            "/api/v1/admin/ops/errors",
            params={"page": page, "page_size": page_size},
        )

    @mcp.tool()
    async def list_request_errors(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List request-level errors.

        Args:
            page: Page number.
            page_size: Number of entries per page.
        """
        return await api_get(
            "/api/v1/admin/ops/request-errors",
            params={"page": page, "page_size": page_size},
        )

    @mcp.tool()
    async def get_request_error(error_id: int) -> dict:
        """Get details of a specific request error.

        Args:
            error_id: The request error ID.
        """
        return await api_get(f"/api/v1/admin/ops/request-errors/{error_id}")

    @mcp.tool()
    async def list_upstream_errors(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List upstream (provider-side) errors.

        Args:
            page: Page number.
            page_size: Number of entries per page.
        """
        return await api_get(
            "/api/v1/admin/ops/upstream-errors",
            params={"page": page, "page_size": page_size},
        )

    @mcp.tool()
    async def get_upstream_error(error_id: int) -> dict:
        """Get details of a specific upstream error.

        Args:
            error_id: The upstream error ID.
        """
        return await api_get(f"/api/v1/admin/ops/upstream-errors/{error_id}")

    @mcp.tool()
    async def get_ops_dashboard_overview() -> dict:
        """Get the ops dashboard overview (aggregate health and metrics)."""
        return await api_get("/api/v1/admin/ops/dashboard/overview")

    @mcp.tool()
    async def list_request_details(
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List detailed request records.

        Args:
            page: Page number.
            page_size: Number of entries per page.
        """
        return await api_get(
            "/api/v1/admin/ops/requests",
            params={"page": page, "page_size": page_size},
        )

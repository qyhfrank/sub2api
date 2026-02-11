"""Group management tools for Sub2API."""

from sub2api_mcp.client import api_get, api_post, api_put, api_delete


def register_tools(mcp):
    @mcp.tool()
    async def list_groups(
        page: int = 1,
        page_size: int = 20,
        platform: str = "",
        status: str = "",
        search: str = "",
        is_exclusive: bool | None = None,
    ) -> dict:
        """List groups with pagination and optional filters."""
        params: dict = {"page": page, "page_size": page_size}
        if platform:
            params["platform"] = platform
        if status:
            params["status"] = status
        if search:
            params["search"] = search
        if is_exclusive is not None:
            params["is_exclusive"] = is_exclusive
        return await api_get("/api/v1/admin/groups", params=params)

    @mcp.tool()
    async def get_all_groups(platform: str = "") -> dict:
        """Get all active groups without pagination."""
        params = {}
        if platform:
            params["platform"] = platform
        return await api_get("/api/v1/admin/groups/all", params=params or None)

    @mcp.tool()
    async def get_group(group_id: int) -> dict:
        """Get details for a specific group by ID."""
        return await api_get(f"/api/v1/admin/groups/{group_id}")

    @mcp.tool()
    async def create_group(
        name: str,
        platform: str = "",
        description: str = "",
        rate_multiplier: float | None = None,
        is_exclusive: bool = False,
        subscription_type: str = "",
        model_routing_enabled: bool = False,
    ) -> dict:
        """Create a new group."""
        data: dict = {
            "name": name,
            "is_exclusive": is_exclusive,
            "model_routing_enabled": model_routing_enabled,
        }
        if platform:
            data["platform"] = platform
        if description:
            data["description"] = description
        if rate_multiplier is not None:
            data["rate_multiplier"] = rate_multiplier
        if subscription_type:
            data["subscription_type"] = subscription_type
        return await api_post("/api/v1/admin/groups", json=data)

    @mcp.tool()
    async def update_group(
        group_id: int,
        name: str = "",
        description: str = "",
        platform: str = "",
        rate_multiplier: float | None = None,
        status: str = "",
        is_exclusive: bool | None = None,
        subscription_type: str = "",
        model_routing_enabled: bool | None = None,
    ) -> dict:
        """Update a group. Only provided (non-empty/non-None) fields are sent."""
        data: dict = {}
        if name:
            data["name"] = name
        if description:
            data["description"] = description
        if platform:
            data["platform"] = platform
        if rate_multiplier is not None:
            data["rate_multiplier"] = rate_multiplier
        if status:
            data["status"] = status
        if is_exclusive is not None:
            data["is_exclusive"] = is_exclusive
        if subscription_type:
            data["subscription_type"] = subscription_type
        if model_routing_enabled is not None:
            data["model_routing_enabled"] = model_routing_enabled
        return await api_put(f"/api/v1/admin/groups/{group_id}", json=data)

    @mcp.tool()
    async def delete_group(group_id: int) -> dict:
        """Delete a group by ID."""
        return await api_delete(f"/api/v1/admin/groups/{group_id}")

    @mcp.tool()
    async def get_group_stats(group_id: int) -> dict:
        """Get statistics for a specific group."""
        return await api_get(f"/api/v1/admin/groups/{group_id}/stats")

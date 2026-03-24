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
        daily_limit_usd: float | None = None,
        weekly_limit_usd: float | None = None,
        monthly_limit_usd: float | None = None,
        image_price_1k: float | None = None,
        image_price_2k: float | None = None,
        image_price_4k: float | None = None,
        claude_code_only: bool = False,
        fallback_group_id: int | None = None,
        fallback_group_id_on_invalid_request: int | None = None,
        model_routing: dict | None = None,
        mcp_xml_inject: bool | None = None,
        supported_model_scopes: list[str] | None = None,
        allow_messages_dispatch: bool = False,
        default_mapped_model: str = "",
        copy_accounts_from_group_ids: list[int] | None = None,
    ) -> dict:
        """Create a new group.

        Platform values: anthropic, openai, gemini, antigravity, sora.
        model_routing: map of model pattern to account ID list, e.g. {"claude-*": [1,2]}.
        copy_accounts_from_group_ids: bind accounts from these groups after creation.
        """
        data: dict = {
            "name": name,
            "is_exclusive": is_exclusive,
            "model_routing_enabled": model_routing_enabled,
            "claude_code_only": claude_code_only,
            "allow_messages_dispatch": allow_messages_dispatch,
        }
        if platform:
            data["platform"] = platform
        if description:
            data["description"] = description
        if rate_multiplier is not None:
            data["rate_multiplier"] = rate_multiplier
        if subscription_type:
            data["subscription_type"] = subscription_type
        if daily_limit_usd is not None:
            data["daily_limit_usd"] = daily_limit_usd
        if weekly_limit_usd is not None:
            data["weekly_limit_usd"] = weekly_limit_usd
        if monthly_limit_usd is not None:
            data["monthly_limit_usd"] = monthly_limit_usd
        if image_price_1k is not None:
            data["image_price_1k"] = image_price_1k
        if image_price_2k is not None:
            data["image_price_2k"] = image_price_2k
        if image_price_4k is not None:
            data["image_price_4k"] = image_price_4k
        if fallback_group_id is not None:
            data["fallback_group_id"] = fallback_group_id
        if fallback_group_id_on_invalid_request is not None:
            data["fallback_group_id_on_invalid_request"] = fallback_group_id_on_invalid_request
        if model_routing is not None:
            data["model_routing"] = model_routing
        if mcp_xml_inject is not None:
            data["mcp_xml_inject"] = mcp_xml_inject
        if supported_model_scopes is not None:
            data["supported_model_scopes"] = supported_model_scopes
        if default_mapped_model:
            data["default_mapped_model"] = default_mapped_model
        if copy_accounts_from_group_ids is not None:
            data["copy_accounts_from_group_ids"] = copy_accounts_from_group_ids
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
        daily_limit_usd: float | None = None,
        weekly_limit_usd: float | None = None,
        monthly_limit_usd: float | None = None,
        image_price_1k: float | None = None,
        image_price_2k: float | None = None,
        image_price_4k: float | None = None,
        claude_code_only: bool | None = None,
        fallback_group_id: int | None = None,
        fallback_group_id_on_invalid_request: int | None = None,
        model_routing: dict | None = None,
        mcp_xml_inject: bool | None = None,
        supported_model_scopes: list[str] | None = None,
        allow_messages_dispatch: bool | None = None,
        default_mapped_model: str | None = None,
        copy_accounts_from_group_ids: list[int] | None = None,
    ) -> dict:
        """Update a group. Only provided (non-empty/non-None) fields are sent.

        model_routing: map of model pattern to account ID list.
        copy_accounts_from_group_ids: clears current bindings, then binds accounts from source groups.
        Use negative price values to clear pricing overrides.
        """
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
        if daily_limit_usd is not None:
            data["daily_limit_usd"] = daily_limit_usd
        if weekly_limit_usd is not None:
            data["weekly_limit_usd"] = weekly_limit_usd
        if monthly_limit_usd is not None:
            data["monthly_limit_usd"] = monthly_limit_usd
        if image_price_1k is not None:
            data["image_price_1k"] = image_price_1k
        if image_price_2k is not None:
            data["image_price_2k"] = image_price_2k
        if image_price_4k is not None:
            data["image_price_4k"] = image_price_4k
        if claude_code_only is not None:
            data["claude_code_only"] = claude_code_only
        if fallback_group_id is not None:
            data["fallback_group_id"] = fallback_group_id
        if fallback_group_id_on_invalid_request is not None:
            data["fallback_group_id_on_invalid_request"] = fallback_group_id_on_invalid_request
        if model_routing is not None:
            data["model_routing"] = model_routing
        if mcp_xml_inject is not None:
            data["mcp_xml_inject"] = mcp_xml_inject
        if supported_model_scopes is not None:
            data["supported_model_scopes"] = supported_model_scopes
        if allow_messages_dispatch is not None:
            data["allow_messages_dispatch"] = allow_messages_dispatch
        if default_mapped_model is not None:
            data["default_mapped_model"] = default_mapped_model
        if copy_accounts_from_group_ids is not None:
            data["copy_accounts_from_group_ids"] = copy_accounts_from_group_ids
        return await api_put(f"/api/v1/admin/groups/{group_id}", json=data)

    @mcp.tool()
    async def delete_group(group_id: int) -> dict:
        """Delete a group by ID."""
        return await api_delete(f"/api/v1/admin/groups/{group_id}")

    @mcp.tool()
    async def get_group_stats(group_id: int) -> dict:
        """Get statistics for a specific group."""
        return await api_get(f"/api/v1/admin/groups/{group_id}/stats")

"""Account management tools for Sub2API MCP server."""

from sub2api_mcp.client import api_delete, api_get, api_post, api_put


def register_tools(mcp):
    @mcp.tool()
    async def list_accounts(
        page: int = 1,
        page_size: int = 20,
        platform: str = "",
        type: str = "",
        status: str = "",
        search: str = "",
        group: int | None = None,
        lite: bool = False,
    ) -> dict:
        """List accounts with optional filters.

        Platform values: anthropic (not "claude"), openai, gemini, antigravity.
        Status values: active, inactive, error, rate_limited.
        Set group to filter by group ID. Set lite=True for lightweight response.
        """
        params = {"page": page, "page_size": page_size}
        if platform:
            params["platform"] = platform
        if type:
            params["type"] = type
        if status:
            params["status"] = status
        if search:
            params["search"] = search
        if group is not None:
            params["group"] = group
        if lite:
            params["lite"] = "true"
        return await api_get("/api/v1/admin/accounts", params=params)

    @mcp.tool()
    async def get_account(account_id: int) -> dict:
        """Get a single account by ID."""
        return await api_get(f"/api/v1/admin/accounts/{account_id}")

    @mcp.tool()
    async def create_account(
        name: str,
        platform: str,
        type: str,
        credentials: dict,
        notes: str = "",
        proxy_id: int = 0,
        concurrency: int = 0,
        priority: int = 0,
        rate_multiplier: float | None = None,
        load_factor: int | None = None,
        group_ids: list[int] | None = None,
        extra: dict | None = None,
        expires_at: int | None = None,
        auto_pause_on_expired: bool | None = None,
    ) -> dict:
        """Create a new account.

        Platform values: anthropic (not "claude"), openai, gemini, antigravity.
        The credentials dict structure varies by platform/type.
        expires_at is a unix timestamp. rate_multiplier controls billing rate.
        """
        data = {
            "name": name,
            "platform": platform,
            "type": type,
            "credentials": credentials,
            "proxy_id": proxy_id,
            "concurrency": concurrency,
            "priority": priority,
        }
        if notes:
            data["notes"] = notes
        if rate_multiplier is not None:
            data["rate_multiplier"] = rate_multiplier
        if load_factor is not None:
            data["load_factor"] = load_factor
        if group_ids is not None:
            data["group_ids"] = group_ids
        if extra is not None:
            data["extra"] = extra
        if expires_at is not None:
            data["expires_at"] = expires_at
        if auto_pause_on_expired is not None:
            data["auto_pause_on_expired"] = auto_pause_on_expired
        return await api_post("/api/v1/admin/accounts", json=data)

    @mcp.tool()
    async def update_account(
        account_id: int,
        name: str = "",
        notes: str = "",
        type: str = "",
        credentials: dict | None = None,
        extra: dict | None = None,
        proxy_id: int | None = None,
        concurrency: int | None = None,
        priority: int | None = None,
        rate_multiplier: float | None = None,
        load_factor: int | None = None,
        status: str = "",
        group_ids: list[int] | None = None,
        expires_at: int | None = None,
        auto_pause_on_expired: bool | None = None,
    ) -> dict:
        """Partial update of an account. Only provided fields are sent.

        WARNING: credentials and extra replace the ENTIRE map when provided,
        not individual keys. Fetch the current value first if you need to
        preserve existing keys.

        Platform values: anthropic (not "claude"), openai, gemini, antigravity.
        expires_at is a unix timestamp. rate_multiplier controls billing rate.
        """
        data: dict = {}
        if name:
            data["name"] = name
        if notes:
            data["notes"] = notes
        if type:
            data["type"] = type
        if credentials is not None:
            data["credentials"] = credentials
        if extra is not None:
            data["extra"] = extra
        if proxy_id is not None:
            data["proxy_id"] = proxy_id
        if concurrency is not None:
            data["concurrency"] = concurrency
        if priority is not None:
            data["priority"] = priority
        if rate_multiplier is not None:
            data["rate_multiplier"] = rate_multiplier
        if load_factor is not None:
            data["load_factor"] = load_factor
        if status:
            data["status"] = status
        if group_ids is not None:
            data["group_ids"] = group_ids
        if expires_at is not None:
            data["expires_at"] = expires_at
        if auto_pause_on_expired is not None:
            data["auto_pause_on_expired"] = auto_pause_on_expired
        return await api_put(f"/api/v1/admin/accounts/{account_id}", json=data)

    @mcp.tool()
    async def delete_account(account_id: int) -> dict:
        """Delete an account by ID."""
        return await api_delete(f"/api/v1/admin/accounts/{account_id}")

    @mcp.tool()
    async def test_account(account_id: int, model_id: str = "") -> dict:
        """Test an account by sending a probe request.

        Optionally specify a model_id to test with a specific model.
        """
        data = {}
        if model_id:
            data["model_id"] = model_id
        return await api_post(
            f"/api/v1/admin/accounts/{account_id}/test", json=data or None
        )

    @mcp.tool()
    async def refresh_account(account_id: int) -> dict:
        """Refresh an account's token/session.

        For OAuth accounts this refreshes the access token.
        """
        return await api_post(f"/api/v1/admin/accounts/{account_id}/refresh")

    @mcp.tool()
    async def get_account_stats(account_id: int, days: int = 30) -> dict:
        """Get usage statistics for an account over a number of days."""
        return await api_get(
            f"/api/v1/admin/accounts/{account_id}/stats",
            params={"days": days},
        )

    @mcp.tool()
    async def get_account_today_stats(account_id: int) -> dict:
        """Get today's usage statistics for an account."""
        return await api_get(f"/api/v1/admin/accounts/{account_id}/today-stats")

    @mcp.tool()
    async def clear_account_error(account_id: int) -> dict:
        """Clear the error state of an account, resetting it to active."""
        return await api_post(f"/api/v1/admin/accounts/{account_id}/clear-error")

    @mcp.tool()
    async def clear_account_rate_limit(account_id: int) -> dict:
        """Clear the rate-limit state of an account."""
        return await api_post(
            f"/api/v1/admin/accounts/{account_id}/clear-rate-limit"
        )

    @mcp.tool()
    async def get_account_temp_unschedulable(account_id: int) -> dict:
        """Get the temporary unschedulable state of an account."""
        return await api_get(
            f"/api/v1/admin/accounts/{account_id}/temp-unschedulable"
        )

    @mcp.tool()
    async def clear_account_temp_unschedulable(account_id: int) -> dict:
        """Clear the temporary unschedulable state of an account."""
        return await api_delete(
            f"/api/v1/admin/accounts/{account_id}/temp-unschedulable"
        )

    @mcp.tool()
    async def set_account_schedulable(account_id: int, schedulable: bool) -> dict:
        """Set whether an account is schedulable (enabled for request routing)."""
        return await api_post(
            f"/api/v1/admin/accounts/{account_id}/schedulable",
            json={"schedulable": schedulable},
        )

    @mcp.tool()
    async def refresh_account_tier(account_id: int) -> dict:
        """Refresh an account's subscription tier (e.g. Gemini Google One tier)."""
        return await api_post(f"/api/v1/admin/accounts/{account_id}/refresh-tier")

    @mcp.tool()
    async def reset_account_quota(account_id: int) -> dict:
        """Reset an API key account's usage quota."""
        return await api_post(f"/api/v1/admin/accounts/{account_id}/reset-quota")

    @mcp.tool()
    async def get_account_available_models(account_id: int) -> dict:
        """Get the list of models available for an account."""
        return await api_get(f"/api/v1/admin/accounts/{account_id}/models")

    @mcp.tool()
    async def bulk_update_accounts(
        account_ids: list[int], updates: dict
    ) -> dict:
        """Bulk update multiple accounts with the same changes.

        The updates dict is spread into the request body alongside account_ids.
        """
        data = {"account_ids": account_ids, **updates}
        return await api_post("/api/v1/admin/accounts/bulk-update", json=data)

    @mcp.tool()
    async def get_account_usage(account_id: int) -> dict:
        """Get detailed usage data for an account."""
        return await api_get(f"/api/v1/admin/accounts/{account_id}/usage")

    @mcp.tool()
    async def copy_account(
        account_id: int,
        new_name: str = "",
    ) -> dict:
        """Copy an existing account to create a new one with the same settings.

        Copies: name, notes, platform, type, credentials, extra, proxy_id,
        concurrency, priority, rate_multiplier, load_factor, group_ids,
        expires_at, auto_pause_on_expired.

        If new_name is not provided, the copy is named "<original_name> (copy)".
        """
        src = await api_get(f"/api/v1/admin/accounts/{account_id}")
        acct = src.get("data", src)

        data: dict = {
            "name": new_name or f"{acct['name']} (copy)",
            "platform": acct["platform"],
            "type": acct["type"],
            "credentials": acct.get("credentials") or {},
        }
        if acct.get("notes"):
            data["notes"] = acct["notes"]
        if acct.get("extra"):
            data["extra"] = acct["extra"]
        if acct.get("proxy_id"):
            data["proxy_id"] = acct["proxy_id"]
        if acct.get("concurrency"):
            data["concurrency"] = acct["concurrency"]
        if acct.get("priority"):
            data["priority"] = acct["priority"]
        if acct.get("rate_multiplier") is not None:
            data["rate_multiplier"] = acct["rate_multiplier"]
        if acct.get("load_factor") is not None:
            data["load_factor"] = acct["load_factor"]
        if acct.get("group_ids"):
            data["group_ids"] = acct["group_ids"]
        if acct.get("expires_at"):
            data["expires_at"] = acct["expires_at"]
        if acct.get("auto_pause_on_expired") is not None:
            data["auto_pause_on_expired"] = acct["auto_pause_on_expired"]
        return await api_post("/api/v1/admin/accounts", json=data)

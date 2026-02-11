"""User management tools for Sub2API."""

from sub2api_mcp.client import api_get, api_post, api_put, api_delete


def register_tools(mcp):
    @mcp.tool()
    async def list_users(
        page: int = 1,
        page_size: int = 20,
        status: str = "",
        role: str = "",
        search: str = "",
    ) -> dict:
        """List users with pagination and optional filters."""
        params = {"page": page, "page_size": page_size}
        if status:
            params["status"] = status
        if role:
            params["role"] = role
        if search:
            params["search"] = search
        return await api_get("/api/v1/admin/users", params=params)

    @mcp.tool()
    async def get_user(user_id: int) -> dict:
        """Get details for a specific user by ID."""
        return await api_get(f"/api/v1/admin/users/{user_id}")

    @mcp.tool()
    async def create_user(
        email: str,
        password: str,
        username: str = "",
        notes: str = "",
        balance: float = 0,
    ) -> dict:
        """Create a new user account."""
        data: dict = {"email": email, "password": password, "balance": balance}
        if username:
            data["username"] = username
        if notes:
            data["notes"] = notes
        return await api_post("/api/v1/admin/users", json=data)

    @mcp.tool()
    async def update_user(
        user_id: int,
        email: str = "",
        password: str = "",
        username: str = "",
        notes: str = "",
        balance: float | None = None,
        concurrency: int | None = None,
        status: str = "",
        allowed_groups: list[int] | None = None,
    ) -> dict:
        """Update a user. Only provided (non-empty/non-None) fields are sent."""
        data: dict = {}
        if email:
            data["email"] = email
        if password:
            data["password"] = password
        if username:
            data["username"] = username
        if notes:
            data["notes"] = notes
        if balance is not None:
            data["balance"] = balance
        if concurrency is not None:
            data["concurrency"] = concurrency
        if status:
            data["status"] = status
        if allowed_groups is not None:
            data["allowed_groups"] = allowed_groups
        return await api_put(f"/api/v1/admin/users/{user_id}", json=data)

    @mcp.tool()
    async def delete_user(user_id: int) -> dict:
        """Delete a user by ID."""
        return await api_delete(f"/api/v1/admin/users/{user_id}")

    @mcp.tool()
    async def update_user_balance(
        user_id: int,
        balance: float,
        operation: str = "set",
        notes: str = "",
    ) -> dict:
        """Update a user's balance. operation: 'set', 'add', or 'subtract'."""
        data: dict = {"balance": balance, "operation": operation}
        if notes:
            data["notes"] = notes
        return await api_post(f"/api/v1/admin/users/{user_id}/balance", json=data)

    @mcp.tool()
    async def get_user_usage(user_id: int, period: str = "") -> dict:
        """Get usage statistics for a specific user."""
        params = {}
        if period:
            params["period"] = period
        return await api_get(f"/api/v1/admin/users/{user_id}/usage", params=params or None)

    @mcp.tool()
    async def get_user_subscriptions(user_id: int) -> dict:
        """Get subscriptions associated with a specific user."""
        return await api_get(f"/api/v1/admin/users/{user_id}/subscriptions")

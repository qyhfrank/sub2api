"""Proxy management tools for Sub2API."""

from sub2api_mcp.client import api_delete, api_get, api_post, api_put


def register_tools(mcp):
    @mcp.tool()
    async def list_proxies(
        page: int = 1,
        page_size: int = 20,
        protocol: str = "",
        status: str = "",
        search: str = "",
    ) -> dict:
        """List proxies with pagination and optional filters.

        Protocol values: http, https, socks5, socks5h.
        Status values: active, inactive.
        Response includes account_count per proxy.
        """
        params: dict = {"page": page, "page_size": page_size}
        if protocol:
            params["protocol"] = protocol
        if status:
            params["status"] = status
        if search:
            params["search"] = search
        return await api_get("/api/v1/admin/proxies", params=params)

    @mcp.tool()
    async def get_proxy(proxy_id: int) -> dict:
        """Get details for a specific proxy by ID."""
        return await api_get(f"/api/v1/admin/proxies/{proxy_id}")

    @mcp.tool()
    async def create_proxy(
        name: str,
        protocol: str,
        host: str,
        port: int,
        username: str = "",
        password: str = "",
    ) -> dict:
        """Create a new proxy.

        Protocol values: http, https, socks5, socks5h.
        Port range: 1-65535.
        """
        data: dict = {
            "name": name,
            "protocol": protocol,
            "host": host,
            "port": port,
        }
        if username:
            data["username"] = username
        if password:
            data["password"] = password
        return await api_post("/api/v1/admin/proxies", json=data)

    @mcp.tool()
    async def update_proxy(
        proxy_id: int,
        name: str = "",
        protocol: str = "",
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
        status: str = "",
    ) -> dict:
        """Update a proxy. Only provided (non-empty/non-zero) fields are sent.

        Protocol values: http, https, socks5, socks5h.
        Status values: active, inactive.
        """
        data: dict = {}
        if name:
            data["name"] = name
        if protocol:
            data["protocol"] = protocol
        if host:
            data["host"] = host
        if port:
            data["port"] = port
        if username:
            data["username"] = username
        if password:
            data["password"] = password
        if status:
            data["status"] = status
        return await api_put(f"/api/v1/admin/proxies/{proxy_id}", json=data)

    @mcp.tool()
    async def delete_proxy(proxy_id: int) -> dict:
        """Delete a proxy by ID."""
        return await api_delete(f"/api/v1/admin/proxies/{proxy_id}")

    @mcp.tool()
    async def test_proxy(proxy_id: int) -> dict:
        """Test proxy connectivity.

        Returns latency, resolved IP, and geolocation info.
        """
        return await api_post(f"/api/v1/admin/proxies/{proxy_id}/test")

    @mcp.tool()
    async def get_proxy_accounts(proxy_id: int) -> dict:
        """List accounts that are using a specific proxy.

        Useful for checking dependencies before deleting a proxy.
        """
        return await api_get(f"/api/v1/admin/proxies/{proxy_id}/accounts")

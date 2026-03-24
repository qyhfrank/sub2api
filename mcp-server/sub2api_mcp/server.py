"""Stdio server entrypoint for the Sub2API FastMCP app."""

from sub2api_mcp.app import get_mcp


def main() -> None:
    get_mcp().run()


if __name__ == "__main__":
    main()

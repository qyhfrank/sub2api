"""CLI wrapper around the Sub2API FastMCP tool registry."""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from pathlib import Path
from typing import Any

from sub2api_mcp.config import get_config, override_config
from sub2api_mcp.tools import call_tool, get_tool_spec, list_tool_specs

_BUILTIN_COMMANDS = {"tools", "describe", "call", "doctor", "mcp-server", "server"}
_SENSITIVE_KEYS = {
    "access_token",
    "refresh_token",
    "id_token",
    "api_key",
    "password",
    "session_token",
    "authorization",
    "cookie",
}


def _parse_value(raw: str) -> Any:
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return raw


def _merge_args(payload: dict[str, Any], token: str) -> None:
    if token.startswith("@"):
        file_data = json.loads(Path(token[1:]).read_text())
        if not isinstance(file_data, dict):
            raise ValueError(f"JSON file must contain an object: {token[1:]}")
        payload.update(file_data)
        return
    if "=" not in token:
        raise ValueError(f"Expected KEY=VALUE or @file.json, got: {token}")
    key, raw_value = token.split("=", 1)
    key = key.strip()
    if not key:
        raise ValueError(f"Invalid empty argument key in: {token}")
    payload[key] = _parse_value(raw_value)


def _load_tool_args(args: argparse.Namespace) -> dict[str, Any]:
    payload: dict[str, Any] = {}
    if getattr(args, "json_args", None):
        json_payload = json.loads(args.json_args)
        if not isinstance(json_payload, dict):
            raise ValueError("--json-args must decode to a JSON object")
        payload.update(json_payload)
    if getattr(args, "json_file", None):
        file_payload = json.loads(Path(args.json_file).read_text())
        if not isinstance(file_payload, dict):
            raise ValueError("--json-file must contain a JSON object")
        payload.update(file_payload)
    for item in getattr(args, "kv_args", []) or []:
        _merge_args(payload, item)
    return payload


def _mask_secret(value: Any) -> Any:
    if not isinstance(value, str):
        return "<redacted>"
    if len(value) <= 8:
        return "<redacted>"
    return f"{value[:4]}...{value[-4:]}"


def _jsonable(value: Any, *, show_secrets: bool = False, parent_key: str | None = None) -> Any:
    if value is None or isinstance(value, (str, int, float, bool)):
        if not show_secrets and parent_key in _SENSITIVE_KEYS:
            return _mask_secret(value)
        return value
    if isinstance(value, Path):
        return str(value)
    if isinstance(value, dict):
        out: dict[str, Any] = {}
        for key, inner in value.items():
            key_str = str(key)
            out[key_str] = _jsonable(inner, show_secrets=show_secrets, parent_key=key_str)
        return out
    if isinstance(value, (list, tuple, set)):
        return [_jsonable(v, show_secrets=show_secrets, parent_key=parent_key) for v in value]
    if hasattr(value, "model_dump"):
        return _jsonable(value.model_dump(), show_secrets=show_secrets, parent_key=parent_key)
    if hasattr(value, "data"):
        return _jsonable(value.data, show_secrets=show_secrets, parent_key=parent_key)
    return str(value)


def _print_json(value: Any, *, show_secrets: bool = False) -> None:
    print(json.dumps(_jsonable(value, show_secrets=show_secrets), ensure_ascii=False, indent=2))


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="sub2api-mcp-cli",
        description=(
            "Use Sub2API admin tools from a terminal command or start the stdio MCP server. "
            "The CLI reuses the same FastMCP registry as the MCP transport."
        ),
    )
    parser.add_argument("--base-url", help="Override SUB2API_BASE_URL for this invocation")
    parser.add_argument("--token", help="Override SUB2API_TOKEN for this invocation")
    parser.add_argument(
        "--timeout",
        type=float,
        help="Override backend request timeout in seconds for this invocation",
    )
    parser.add_argument(
        "--show-secrets",
        action="store_true",
        help="Do not redact sensitive fields in JSON output",
    )

    subparsers = parser.add_subparsers(dest="command", required=True)

    tools_parser = subparsers.add_parser("tools", help="List available tool names")
    tools_parser.add_argument("--json", action="store_true", help="Print full JSON tool metadata")

    describe_parser = subparsers.add_parser("describe", help="Show one tool's description and schema")
    describe_parser.add_argument("tool", help="Tool name, for example list_accounts")
    describe_parser.add_argument("--json", action="store_true", help="Print raw JSON schema")

    call_parser = subparsers.add_parser("call", help="Invoke one tool")
    call_parser.add_argument("tool", help="Tool name, for example list_accounts")
    call_parser.add_argument("kv_args", nargs="*", help="KEY=VALUE pairs or @payload.json")
    call_parser.add_argument("--json-args", help="Inline JSON object with tool arguments")
    call_parser.add_argument("--json-file", help="Path to a JSON file containing tool arguments")

    doctor_parser = subparsers.add_parser("doctor", help="Show resolved config and run health_check")
    doctor_parser.add_argument("--skip-check", action="store_true", help="Do not call health_check")

    server_parser = subparsers.add_parser("mcp-server", help="Run the stdio MCP server")
    server_parser.add_argument("--transport", default="stdio", choices=["stdio"], help="Reserved for future transports")
    subparsers.add_parser("server", help="Alias for mcp-server")

    return parser


def _tool_index() -> dict[str, Any]:
    return {tool.name: tool for tool in list_tool_specs()}


async def _cmd_tools(args: argparse.Namespace) -> int:
    tools = list_tool_specs()
    if args.json:
        _print_json(
            [tool.mcp_tool().model_dump(mode="json", by_alias=True) for tool in tools],
            show_secrets=args.show_secrets,
        )
        return 0
    width = max(len(tool.name) for tool in tools) if tools else 0
    for tool in tools:
        first_line = tool.description.splitlines()[0] if tool.description else ""
        print(f"{tool.name.ljust(width)}  {first_line}")
    return 0


async def _cmd_describe(args: argparse.Namespace) -> int:
    tool_map = _tool_index()
    tool = tool_map.get(args.tool)
    if tool is None:
        print(f"Unknown tool: {args.tool}", file=sys.stderr)
        return 2
    payload = tool.mcp_tool().model_dump(mode="json", by_alias=True)
    if args.json:
        _print_json(payload, show_secrets=args.show_secrets)
        return 0
    print(tool.name)
    if tool.description:
        print(f"\n{tool.description.strip()}\n")
    input_schema = payload.get("inputSchema") or {}
    properties = input_schema.get("properties") or {}
    required = set(input_schema.get("required") or [])
    if properties:
        print("Parameters:")
        for key, schema in properties.items():
            type_name = schema.get("type")
            if not type_name and "anyOf" in schema:
                type_name = " | ".join(sorted(filter(None, {item.get("type") for item in schema["anyOf"]})))
            default = schema.get("default", "<required>" if key in required else "<none>")
            print(f"- {key}: type={type_name or 'unknown'} default={default}")
    else:
        print("Parameters: none")
    print("\nInput schema JSON:")
    _print_json(input_schema, show_secrets=args.show_secrets)
    return 0


async def _cmd_call(args: argparse.Namespace) -> int:
    if get_tool_spec(args.tool) is None:
        print(f"Unknown tool: {args.tool}", file=sys.stderr)
        return 2
    payload = _load_tool_args(args)
    result = await call_tool(args.tool, payload)
    _print_json(result, show_secrets=args.show_secrets)
    return 0


async def _cmd_doctor(args: argparse.Namespace) -> int:
    config = get_config()
    token_mode = "missing"
    if config.api_token:
        token_mode = (
            "x-api-key"
            if config.api_token.startswith("admin-")
            else "authorization-bearer"
        )
    summary: dict[str, Any] = {
        "base_url": config.base_url,
        "token_present": bool(config.api_token),
        "auth_mode": token_mode,
        "timeout": config.timeout,
        "tool_count": len(list_tool_specs()),
    }
    if not args.skip_check:
        try:
            summary["health_check"] = await call_tool("health_check", {})
        except Exception as exc:
            summary["health_check_error"] = str(exc)
            _print_json(summary, show_secrets=args.show_secrets)
            return 1
    _print_json(summary, show_secrets=args.show_secrets)
    return 0


async def _run_async(args: argparse.Namespace) -> int:
    if args.command == "tools":
        return await _cmd_tools(args)
    if args.command == "describe":
        return await _cmd_describe(args)
    if args.command == "call":
        return await _cmd_call(args)
    if args.command == "doctor":
        return await _cmd_doctor(args)
    return 1


def _run_server() -> int:
    from sub2api_mcp.app import get_mcp

    get_mcp().run()
    return 0


def main(argv: list[str] | None = None) -> int:
    raw_args = list(sys.argv[1:] if argv is None else argv)
    if raw_args and not raw_args[0].startswith("-") and raw_args[0] not in _BUILTIN_COMMANDS:
        raw_args = ["call", *raw_args]

    parser = _build_parser()
    args = parser.parse_args(raw_args)
    try:
        with override_config(
            base_url=args.base_url,
            api_token=args.token,
            timeout=args.timeout,
        ):
            if args.command in {"mcp-server", "server"}:
                return _run_server()
            return asyncio.run(_run_async(args))
    except KeyboardInterrupt:
        print("Interrupted", file=sys.stderr)
        return 130
    except Exception as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

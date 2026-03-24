"""Transport-agnostic tool catalog shared by CLI and MCP registration."""

from __future__ import annotations

from dataclasses import dataclass
from functools import lru_cache
from typing import Any

from fastmcp.server.apps import ui_to_meta_dict
from fastmcp.tools.function_tool import FunctionTool


_ALLOWED_TOOL_KWARGS = {
    "name",
    "version",
    "title",
    "description",
    "icons",
    "tags",
    "output_schema",
    "annotations",
    "exclude_args",
    "meta",
    "ui",
    "task",
    "timeout",
    "auth",
}


@dataclass(frozen=True)
class ToolSpec:
    tool: FunctionTool

    @property
    def name(self) -> str:
        return self.tool.name

    @property
    def description(self) -> str:
        return self.tool.description or ""

    @property
    def input_schema(self) -> dict[str, Any]:
        return self.mcp_tool().inputSchema

    def mcp_tool(self) -> Any:
        return self.tool.to_mcp_tool()

    async def invoke(self, payload: dict[str, Any]) -> Any:
        result = await self.tool.run(payload)
        if result.structured_content is not None:
            if self.tool.output_schema and self.tool.output_schema.get(
                "x-fastmcp-wrap-result"
            ):
                return result.structured_content.get("result")
            return result.structured_content
        return [block.model_dump() for block in result.content]


def _normalize_tool_kwargs(kwargs: dict[str, Any]) -> dict[str, Any]:
    unsupported = sorted(set(kwargs) - _ALLOWED_TOOL_KWARGS)
    if unsupported:
        raise TypeError(
            "Unsupported tool kwargs for CLI catalog: " + ", ".join(unsupported)
        )

    normalized = dict(kwargs)
    ui = normalized.pop("ui", None)
    if ui is not None:
        meta = dict(normalized.get("meta") or {})
        meta["ui"] = ui_to_meta_dict(ui)
        normalized["meta"] = meta
    return normalized


class CollectingRegistry:
    def __init__(self) -> None:
        self.tools: dict[str, ToolSpec] = {}

    def tool(self, *args: Any, **kwargs: Any):
        normalized_kwargs = _normalize_tool_kwargs(kwargs)

        if args and callable(args[0]) and len(args) == 1:
            return self._register(args[0], normalized_kwargs)

        def decorator(handler: Any) -> Any:
            return self._register(handler, normalized_kwargs)

        return decorator

    def _register(self, handler: Any, kwargs: dict[str, Any]) -> Any:
        tool = FunctionTool.from_function(handler, **kwargs)
        self.tools[tool.name] = ToolSpec(tool=tool)
        return handler


@lru_cache(maxsize=1)
def build_tool_catalog() -> dict[str, ToolSpec]:
    from sub2api_mcp.tools import TOOL_REGISTRARS

    collector = CollectingRegistry()
    for register in TOOL_REGISTRARS:
        register(collector)
    return collector.tools


def list_tool_specs() -> list[ToolSpec]:
    return list(build_tool_catalog().values())


def get_tool_spec(name: str) -> ToolSpec | None:
    return build_tool_catalog().get(name)


async def call_tool(name: str, payload: dict[str, Any]) -> Any:
    spec = get_tool_spec(name)
    if spec is None:
        raise KeyError(name)
    return await spec.invoke(payload)

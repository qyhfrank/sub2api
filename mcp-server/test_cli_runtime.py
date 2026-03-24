import contextlib
import io
import asyncio
import json
import unittest
from unittest import mock

from sub2api_mcp import cli
from sub2api_mcp import client
from sub2api_mcp.config import get_config
from sub2api_mcp.tool_catalog import CollectingRegistry, get_tool_spec


class CLIRuntimeTests(unittest.TestCase):
    def tearDown(self) -> None:
        client.reset_config()

    def test_cli_tools_does_not_require_fastmcp_client(self) -> None:
        stdout = io.StringIO()
        stderr = io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            exit_code = cli.main(["tools"])

        self.assertEqual(exit_code, 0)
        self.assertFalse(hasattr(cli, "Client"))
        self.assertIn("health_check", stdout.getvalue())

    def test_cli_timeout_updates_backend_client_timeout(self) -> None:
        with mock.patch("sub2api_mcp.client.httpx.AsyncClient") as async_client:
            response = mock.Mock()
            response.raise_for_status.return_value = None
            response.json.return_value = {"ok": True}

            transport = mock.AsyncMock()
            transport.get.return_value = response
            async_client.return_value.__aenter__.return_value = transport

            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                exit_code = cli.main(["--timeout", "12.5", "doctor"])

        self.assertEqual(exit_code, 0)
        self.assertEqual(async_client.call_args.kwargs["timeout"], 12.5)

    def test_cli_main_restores_runtime_config_after_invocation(self) -> None:
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            cli.main(["--timeout", "12.5", "tools"])

        self.assertEqual(get_config().timeout, 30.0)

    def test_optional_parameter_schema_keeps_union_types(self) -> None:
        spec = get_tool_spec("list_accounts")

        assert spec is not None
        group_schema = spec.input_schema["properties"]["group"]
        self.assertEqual(group_schema["anyOf"][0]["type"], "integer")
        self.assertEqual(group_schema["anyOf"][1]["type"], "null")

    def test_collecting_registry_runs_sync_tools(self) -> None:
        registry = CollectingRegistry()

        @registry.tool(description="sync tool")
        def sync_tool(value: int) -> dict:
            return {"value": value + 1}

        spec = registry.tools["sync_tool"]
        result = asyncio.run(spec.invoke({"value": 2}))

        self.assertEqual(result, {"value": 3})

    def test_collecting_registry_unwraps_scalar_results(self) -> None:
        registry = CollectingRegistry()

        @registry.tool(description="scalar tool")
        def scalar_tool() -> str:
            return "ok"

        spec = registry.tools["scalar_tool"]
        result = asyncio.run(spec.invoke({}))

        self.assertEqual(result, "ok")

    def test_collecting_registry_rejects_unknown_tool_kwargs(self) -> None:
        registry = CollectingRegistry()

        with self.assertRaises(TypeError):
            @registry.tool(unsupported_flag=True)
            async def bad_tool() -> dict:
                return {"ok": True}

    def test_describe_json_keeps_output_schema_metadata(self) -> None:
        stdout = io.StringIO()
        stderr = io.StringIO()

        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            exit_code = cli.main(["describe", "health_check", "--json"])

        self.assertEqual(exit_code, 0)
        payload = json.loads(stdout.getvalue())
        self.assertIn("outputSchema", payload)


if __name__ == "__main__":
    unittest.main()

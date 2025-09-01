#!/usr/bin/env python3
"""
MCP Client Example - Demonstrates how external applications can use Mimir AIP plugins as MCP tools
"""

import requests
import json
import time

class MimirAIPMCPClient:
    """Example MCP client for Mimir AIP"""

    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.tools = []
        self.discover_tools()

    def discover_tools(self):
        """Discover available MCP tools"""
        try:
            response = requests.get(f"{self.base_url}/mcp/tools")
            if response.status_code == 200:
                self.tools = response.json().get("tools", [])
                print(f"Discovered {len(self.tools)} MCP tools:")
                for tool in self.tools:
                    print(f"  - {tool['name']}: {tool['description']}")
            else:
                print(f"Failed to discover tools: {response.status_code}")
        except Exception as e:
            print(f"Error discovering tools: {e}")

    def execute_tool(self, tool_name, step_config, context=None):
        """Execute an MCP tool"""
        if context is None:
            context = {}

        arguments = {
            "step_config": step_config,
            "context": context
        }

        try:
            response = requests.post(
                f"{self.base_url}/mcp/tools/execute",
                json={
                    "tool_name": tool_name,
                    "arguments": arguments
                },
                timeout=30
            )

            if response.status_code == 200:
                return response.json()
            else:
                return {
                    "success": False,
                    "error": f"HTTP {response.status_code}: {response.text}"
                }
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }

    def demonstrate_tools(self):
        """Demonstrate using different MCP tools"""
        print("\n=== MCP Tool Demonstration ===\n")

        # Example 1: Use API input plugin
        print("1. Testing Input.api plugin:")
        result = self.execute_tool("Input.api", {
            "name": "fetch_test_data",
            "config": {
                "url": "https://httpbin.org/get",
                "method": "GET"
            },
            "output": "test_data"
        })

        print(f"   Result: {json.dumps(result, indent=2)}")

        # Example 2: Use HTML output plugin
        print("\n2. Testing Output.html plugin:")
        result = self.execute_tool("Output.html", {
            "name": "generate_report",
            "config": {
                "title": "Test Report",
                "template": "simple"
            },
            "output": "report_html"
        }, context={"previous_data": "some data"})

        print(f"   Result: {json.dumps(result, indent=2)}")

        # Example 3: Chain tools together
        print("\n3. Chaining tools together:")
        # First, get some data
        api_result = self.execute_tool("Input.api", {
            "name": "get_data",
            "config": {"url": "https://httpbin.org/get"},
            "output": "raw_data"
        })

        if api_result.get("success"):
            # Then generate a report with that data
            context = api_result.get("result", {})
            report_result = self.execute_tool("Output.html", {
                "name": "create_report",
                "config": {"title": "Generated Report"},
                "output": "final_report"
            }, context)

            print(f"   Chained result: {json.dumps(report_result, indent=2)}")
        else:
            print("   Failed to get initial data for chaining")

def main():
    """Main demonstration function"""
    print("Mimir AIP MCP Client Example")
    print("=" * 40)

    # Wait a moment for server to start if needed
    print("Connecting to Mimir AIP server...")
    time.sleep(1)

    # Create MCP client
    client = MimirAIPMCPClient()

    if not client.tools:
        print("No tools discovered. Make sure Mimir AIP server is running on http://localhost:8080")
        return

    # Demonstrate tool usage
    client.demonstrate_tools()

    print("\n=== MCP Integration Complete ===")
    print("External applications can now use Mimir AIP plugins as MCP tools!")
    print("This enables seamless integration with LLM agents and other MCP-compatible systems.")

if __name__ == "__main__":
    main()
#!/usr/bin/env python3
"""
Comprehensive test suite for TinyMem MCP functionality

This test suite covers MCP tool handlers and their functionality.
"""

import os
import sys
import tempfile
import shutil
import subprocess
import json
import unittest
import threading
import time
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
if str(ROOT_DIR) not in sys.path:
    sys.path.insert(0, str(ROOT_DIR))

from test.http_stub_server import StubLLMServer


class TinyMemMCPTestCase(unittest.TestCase):
    """Test case for TinyMem MCP functionality"""
    
    def setUp(self):
        """Set up test environment with temporary directory"""
        self.original_cwd = os.getcwd()
        self.temp_dir = tempfile.mkdtemp(prefix="tinymem_mcp_test_")
        os.chdir(self.temp_dir)
        
        # Initialize a git repo to ensure TinyMem can detect project root
        subprocess.run(["git", "init"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.email", "test@example.com"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test User"], check=True, capture_output=True)
        
        # Path to tinymem binary (next to this test file)
        test_dir = Path(__file__).resolve().parent
        repo_root = test_dir.parent
        self.tinymem_path = str(test_dir / "tinymem")
        
        # Verify tinymem binary exists
        if not os.path.exists(self.tinymem_path):
            # Try to build it
            build_result = subprocess.run(
                [
                    "go",
                    "build",
                    "-tags",
                    "fts5",
                    "-o",
                    self.tinymem_path,
                    "./cmd/tinymem",
                ],
                cwd=repo_root,
                capture_output=True,
                text=True,
            )
            if build_result.returncode != 0:
                stderr = build_result.stderr.strip() if build_result.stderr else "unknown build error"
                raise RuntimeError(f"Could not build tinymem binary: {stderr}")
    
    def tearDown(self):
        """Clean up test environment"""
        os.chdir(self.original_cwd)
        shutil.rmtree(self.temp_dir, ignore_errors=True)
    
    def send_mcp_request(self, method, params=None, tool_name=None, tool_args=None, env=None):
        """Send an MCP request to tinymem mcp server"""
        if method == "tools/call":
            # Format for tool call
            request = {
                "jsonrpc": "2.0",
                "method": "tools/call",
                "params": {
                    "name": tool_name,
                    "arguments": tool_args or {}
                },
                "id": 1
            }
        else:
            # Format for other methods
            request_obj = {
                "jsonrpc": "2.0",
                "method": method,
                "id": 1
            }
            if params:
                request_obj["params"] = params
            
            request = request_obj
        
        # Start MCP server in a subprocess
        full_env = os.environ.copy()
        if env:
            full_env.update(env)

        proc = subprocess.Popen([self.tinymem_path, "mcp"],
                               stdin=subprocess.PIPE,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               text=True,
                               env=full_env)
        
        # Send the request
        request_json = json.dumps(request) + "\n"
        stdout, stderr = proc.communicate(input=request_json, timeout=10)

        # Clean stdout to remove diagnostic prefixes (e.g., logging from the Ralph loop)
        clean_stdout = stdout
        first_json = clean_stdout.find("{")
        last_json = clean_stdout.rfind("}")
        if first_json != -1 and last_json != -1 and last_json >= first_json:
            clean_stdout = clean_stdout[first_json:last_json + 1]

        # Parse the response
        try:
            response = json.loads(clean_stdout.strip())
            return response, stderr
        except json.JSONDecodeError:
            return None, f"Could not parse response: {stdout}\nSTDERR: {stderr}"

    def _ralph_chat_response(self, path, body, patch_name):
        return {
            "choices": [
                {
                    "message": {
                        "content": f"@@@ FILE: {patch_name} @@@\nRalph repair output\n@@@ END_FILE @@@"
                    }
                }
            ]
        }

    def _cove_chat_response(self, path, body):
        if "MEMORIES:" in body:
            payload = [
                {"id": "0", "include": False},
                {"id": "1", "include": True},
            ]
            return {"choices": [{"message": {"content": json.dumps(payload)}}]}
        if "CANDIDATES:" in body:
            payload = [
                {"id": "0", "confidence": 0.88, "reason": "keep"},
            ]
            return {"choices": [{"message": {"content": json.dumps(payload)}}]}
        return StubLLMServer._default_chat_response(path, body)
    
    def test_mcp_initialize(self):
        """Test MCP initialize method"""
        response, stderr = self.send_mcp_request("initialize")
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("serverInfo", response["result"])
        self.assertEqual(response["result"]["serverInfo"]["name"], "tinyMem")
    
    def test_mcp_ping(self):
        """Test MCP ping method"""
        response, stderr = self.send_mcp_request("ping")
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertEqual(response["result"], {})
    
    def test_mcp_tools_list(self):
        """Test MCP tools/list method"""
        response, stderr = self.send_mcp_request("tools/list")
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("tools", response["result"])
        
        # Check that expected tools are present
        tool_names = [tool["name"] for tool in response["result"]["tools"]]
        expected_tools = [
            "memory_query", "memory_recent", "memory_write", 
            "memory_stats", "memory_health", "memory_doctor"
        ]
        
        for expected_tool in expected_tools:
            self.assertIn(expected_tool, tool_names)
    
    def test_mcp_memory_write(self):
        """Test MCP memory_write tool"""
        # Write a memory using MCP
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_write",
            tool_args={
                "type": "note",
                "summary": "MCP test note",
                "detail": "This note was created via MCP"
            }
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        self.assertTrue(any("Memory created successfully" in str(item.get("text", "")) 
                           for item in response["result"]["content"]))
    
    def test_mcp_memory_query(self):
        """Test MCP memory_query tool"""
        # First write a memory
        self.send_mcp_request(
            "tools/call", 
            tool_name="memory_write",
            tool_args={
                "type": "note",
                "summary": "Query test note",
                "detail": "This is for query testing"
            }
        )
        
        # Then query it
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_query",
            tool_args={"query": "query test"}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
        self.assertIn("Query test note", content_text)
    
    def test_mcp_memory_recent(self):
        """Test MCP memory_recent tool"""
        # Write a memory first
        self.send_mcp_request(
            "tools/call", 
            tool_name="memory_write",
            tool_args={
                "type": "note",
                "summary": "Recent test note"
            }
        )
        
        # Get recent memories
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_recent",
            tool_args={}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
        self.assertIn("Recent test note", content_text)
    
    def test_mcp_memory_stats(self):
        """Test MCP memory_stats tool"""
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_stats",
            tool_args={}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
        self.assertIn("Memory Statistics", content_text)
        self.assertIn("Total memories:", content_text)
    
    def test_mcp_memory_health(self):
        """Test MCP memory_health tool"""
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_health",
            tool_args={}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
        self.assertIn("HEALTHY", content_text)
        self.assertIn("Database connectivity: OK", content_text)
    
    def test_mcp_memory_doctor(self):
        """Test MCP memory_doctor tool"""
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="memory_doctor",
            tool_args={}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("result", response)
        self.assertIn("content", response["result"])
        content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
        self.assertIn("tinyMem Diagnostics Report", content_text)
    
    def test_mcp_invalid_tool(self):
        """Test MCP with invalid tool name"""
        response, stderr = self.send_mcp_request(
            "tools/call", 
            tool_name="invalid_tool_name",
            tool_args={}
        )
        
        self.assertIsNotNone(response, f"Failed to get valid response: {stderr}")
        self.assertIn("error", response)
        self.assertIn("Tool not found", response["error"]["message"])

    def test_mcp_memory_ralph_repair(self):
        """memory_ralph should run the repair loop and write a patch file."""
        patch_file = "ralph_patch.txt"
        try:
            stub = StubLLMServer(chat_response=lambda path, body: self._ralph_chat_response(path, body, patch_file))
        except PermissionError as exc:
            self.skipTest(f"Cannot bind stub server: {exc}")
        stub.start()
        env = {
            "TINYMEM_LLM_BASE_URL": stub.base_url,
            "TINYMEM_COVE_ENABLED": "false",
            "TINYMEM_SEMANTIC_ENABLED": "false",
            "TINYMEM_COVE_RECALL_FILTER_ENABLED": "false",
        }

        try:
            args = {
                "task": "Fix patch file",
                "command": "sh -c 'exit 1'",
                "evidence": [f"file_exists::{patch_file}"],
                "max_iterations": 2,
                "recall": {
                    "query_terms": ["patch"],
                    "limit": 2,
                },
                "safety": {
                    "allow_shell": True,
                    "forbid_paths": [],
                    "forbid_commands": [],
                },
                "human_gate": {
                    "on_ambiguity": False,
                    "after_iterations": 0,
                },
            }

            response, stderr = self.send_mcp_request(
                "tools/call",
                tool_name="memory_ralph",
                tool_args=args,
                env=env,
            )

            self.assertIsNotNone(response, f"Failed to get memory_ralph response: {stderr}")
            text = response["result"]["content"][0]["text"]
            result_obj = json.loads(text)
            self.assertEqual(result_obj["status"], "success")
            self.assertTrue(os.path.exists(patch_file))
            with open(patch_file, "r") as f:
                self.assertIn("Ralph repair output", f.read())
        finally:
            stub.stop()

    def test_mcp_memory_query_with_cove_filter(self):
        """memory_query should filter out low-confidence items when CoVe recall filtering runs."""
        try:
            stub = StubLLMServer(chat_response=self._cove_chat_response)
        except PermissionError as exc:
            self.skipTest(f"Cannot bind stub server: {exc}")
        stub.start()

        env = {
            "TINYMEM_LLM_BASE_URL": stub.base_url,
            "TINYMEM_COVE_ENABLED": "true",
            "TINYMEM_COVE_RECALL_FILTER_ENABLED": "true",
            "TINYMEM_SEMANTIC_ENABLED": "false",
        }

        try:
            summaries = ["First note", "Second note"]
            for summary in summaries:
                self.send_mcp_request(
                    "tools/call",
                    tool_name="memory_write",
                    tool_args={
                        "type": "note",
                        "summary": summary,
                        "detail": f"Detail for {summary}",
                    },
                    env=env,
                )

            response, stderr = self.send_mcp_request(
                "tools/call",
                tool_name="memory_query",
                tool_args={"query": "note", "limit": 5},
                env=env,
            )

            self.assertIsNotNone(response, f"Memory query failed: {stderr}")
            content_text = " ".join([item.get("text", "") for item in response["result"]["content"]])
            self.assertIn("First note", content_text)
            self.assertIn("Second note", content_text)

        finally:
            stub.stop()


def run_mcp_tests():
    """Run the MCP test suite"""
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromTestCase(TinyMemMCPTestCase)
    
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    
    return result.wasSuccessful()


if __name__ == "__main__":
    success = run_mcp_tests()
    sys.exit(0 if success else 1)

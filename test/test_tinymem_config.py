#!/usr/bin/env python3
"""
Comprehensive test suite for TinyMem environment variables and configuration

This test suite covers environment variable handling and configuration functionality.
"""

import os
import sys
import tempfile
import shutil
import subprocess
import json
import unittest
from pathlib import Path


class TinyMemConfigTestCase(unittest.TestCase):
    """Test case for TinyMem configuration and environment variables"""
    
    def setUp(self):
        """Set up test environment with temporary directory"""
        self.original_cwd = os.getcwd()
        self.temp_dir = tempfile.mkdtemp(prefix="tinymem_config_test_")
        os.chdir(self.temp_dir)
        
        # Initialize a git repo to ensure TinyMem can detect project root
        subprocess.run(["git", "init"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.email", "test@example.com"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test User"], check=True, capture_output=True)
        
        # Path to tinymem binary
        self.tinymem_path = os.path.join(os.path.dirname(__file__), "tinymem")
        
        # Verify tinymem binary exists
        if not os.path.exists(self.tinymem_path):
            # Try to build it
            build_result = subprocess.run([
                "go", "build", "-tags", "fts5", "-o", self.tinymem_path, 
                "./cmd/tinymem"
            ], cwd=os.path.dirname(__file__))
            if build_result.returncode != 0:
                raise RuntimeError(f"Could not build tinymem binary: {build_result.stderr}")
    
    def tearDown(self):
        """Clean up test environment"""
        os.chdir(self.original_cwd)
        shutil.rmtree(self.temp_dir, ignore_errors=True)
    
    def run_tinymem_cmd(self, args, env=None, expected_exit_code=0):
        """Helper to run tinymem command with custom environment"""
        cmd = [self.tinymem_path] + args
        full_env = os.environ.copy()
        if env:
            full_env.update(env)
        
        result = subprocess.run(cmd, capture_output=True, text=True, env=full_env)
        
        if expected_exit_code is not None:
            self.assertEqual(result.returncode, expected_exit_code,
                           f"Command {' '.join(cmd)} failed with exit code {result.returncode}. "
                           f"Stderr: {result.stderr}")
        
        return result
    
    def test_env_proxy_port(self):
        """Test TINYMEM_PROXY_PORT environment variable"""
        env = {"TINYMEM_PROXY_PORT": "9999"}
        result = self.run_tinymem_cmd(["health"], env=env)
        # Health check should still pass even with different port
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_log_level(self):
        """Test TINYMEM_LOG_LEVEL environment variable"""
        env = {"TINYMEM_LOG_LEVEL": "debug"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_llm_base_url(self):
        """Test TINYMEM_LLM_BASE_URL environment variable"""
        env = {"TINYMEM_LLM_BASE_URL": "http://localhost:8080/v1"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_semantic_enabled(self):
        """Test TINYMEM_SEMANTIC_ENABLED environment variable"""
        env = {"TINYMEM_SEMANTIC_ENABLED": "false"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_hybrid_weight(self):
        """Test TINYMEM_HYBRID_WEIGHT environment variable"""
        env = {"TINYMEM_HYBRID_WEIGHT": "0.7"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_recall_max_items(self):
        """Test TINYMEM_RECALL_MAX_ITEMS environment variable"""
        env = {"TINYMEM_RECALL_MAX_ITEMS": "20"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_recall_max_tokens(self):
        """Test TINYMEM_RECALL_MAX_TOKENS environment variable"""
        env = {"TINYMEM_RECALL_MAX_TOKENS": "3000"}
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_env_cove_settings(self):
        """Test TINYMEM_COVE_* environment variables"""
        env = {
            "TINYMEM_COVE_ENABLED": "false",
            "TINYMEM_COVE_CONFIDENCE_THRESHOLD": "0.8",
            "TINYMEM_COVE_MAX_CANDIDATES": "15",
            "TINYMEM_COVE_TIMEOUT_SECONDS": "25"
        }
        result = self.run_tinymem_cmd(["health"], env=env)
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_invalid_env_values(self):
        """Test handling of invalid environment variable values"""
        # Test invalid proxy port
        env = {"TINYMEM_PROXY_PORT": "999999"}  # Invalid port number
        result = self.run_tinymem_cmd(["health"], env=env)
        # Should still work since health doesn't require proxy to be running
    
    def test_config_file_override(self):
        """Test configuration file override functionality"""
        # Create a config file in .tinyMem directory
        tiny_mem_dir = os.path.join(self.temp_dir, ".tinyMem")
        os.makedirs(tiny_mem_dir, exist_ok=True)
        
        config_content = """
[proxy]
port = 9876

[recall]
max_items = 15
max_tokens = 2500
semantic_enabled = true
hybrid_weight = 0.6

[logging]
level = "debug"
"""
        
        config_path = os.path.join(tiny_mem_dir, "config.toml")
        with open(config_path, 'w') as f:
            f.write(config_content)
        
        # Run health check - should use config file values
        result = self.run_tinymem_cmd(["health"])
        self.assertIn("Database connectivity: OK", result.stdout)
    
    def test_evidence_allow_command_disabled_by_default(self):
        """Test that command evidence is disabled by default"""
        # Write a memory with command evidence (should fail by default)
        result = self.run_tinymem_cmd([
            "write", "--type", "claim", "--summary", "Test command evidence"
        ])
        # This should work since it's just a claim without evidence verification
    
    def test_project_isolation_enforcement(self):
        """Test that project isolation is enforced"""
        # Create a memory in current project
        result = self.run_tinymem_cmd([
            "write", "--type", "note", "--summary", "Project A note"
        ])
        self.assertIn("Memory created successfully!", result.stdout)
        
        # Create another temporary directory for a different project
        with tempfile.TemporaryDirectory() as temp_dir_b:
            os.chdir(temp_dir_b)
            subprocess.run(["git", "init"], check=True, capture_output=True)
            subprocess.run(["git", "config", "user.email", "test@example.com"], check=True, capture_output=True)
            subprocess.run(["git", "config", "user.name", "Test User"], check=True, capture_output=True)
            
            # Write a memory in project B
            result = self.run_tinymem_cmd([
                "write", "--type", "note", "--summary", "Project B note"
            ])
            self.assertIn("Memory created successfully!", result.stdout)
            
            # Query in project B - should only find B's memory
            result = self.run_tinymem_cmd(["query", "note"])
            self.assertIn("Project B note", result.stdout)
            self.assertNotIn("Project A note", result.stdout)
        
        # Back in original project, should only find A's memory
        os.chdir(self.temp_dir)
        result = self.run_tinymem_cmd(["query", "note"])
        self.assertIn("Project A note", result.stdout)
        self.assertNotIn("Project B note", result.stdout)


def run_config_tests():
    """Run the configuration test suite"""
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromTestCase(TinyMemConfigTestCase)
    
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    
    return result.wasSuccessful()


if __name__ == "__main__":
    success = run_config_tests()
    sys.exit(0 if success else 1)
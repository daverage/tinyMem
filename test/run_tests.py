#!/usr/bin/env python3
"""
Test runner for TinyMem comprehensive test suite

This script runs all test suites for TinyMem in a coordinated manner.
"""

import os
import sys
import subprocess
import unittest
from pathlib import Path


def build_tinymem():
    """Build the tinymem binary if it doesn't exist"""
    tinymem_path = Path(__file__).parent / "tinymem"
    
    if not tinymem_path.exists():
        print("Building tinymem binary...")
        result = subprocess.run([
            "go", "build", "-tags", "fts5", "-o", str(tinymem_path), 
            "./cmd/tinymem"
        ], cwd=Path(__file__).parent)
        
        if result.returncode != 0:
            print(f"Failed to build tinymem: {result.stderr}")
            return False
    
    return True


def run_individual_test_suite(test_file):
    """Run an individual test suite file"""
    print(f"\n{'='*60}")
    print(f"RUNNING TEST SUITE: {test_file}")
    print(f"{'='*60}")
    
    result = subprocess.run([sys.executable, test_file], 
                          cwd=Path(__file__).parent)
    
    return result.returncode == 0


def main():
    """Main test runner function"""
    print("TinyMem Comprehensive Test Suite Runner")
    print("="*50)
    
    # Build tinymem binary if needed
    if not build_tinymem():
        print("ERROR: Could not build tinymem binary. Exiting.")
        sys.exit(1)
    
    # Define test suites
    test_suites = [
        "test_tinymem.py",      # Basic functionality tests
        "test_tinymem_mcp.py",  # MCP functionality tests  
        "test_tinymem_config.py" # Configuration and environment tests
    ]
    
    # Check that all test files exist
    base_path = Path(__file__).parent
    for test_suite in test_suites:
        test_path = base_path / test_suite
        if not test_path.exists():
            print(f"ERROR: Test file {test_suite} does not exist")
            sys.exit(1)
    
    # Run each test suite
    all_passed = True
    results = {}
    
    for test_suite in test_suites:
        success = run_individual_test_suite(test_suite)
        results[test_suite] = success
        all_passed = all_passed and success
    
    # Print summary
    print(f"\n{'='*60}")
    print("TEST SUITE SUMMARY")
    print(f"{'='*60}")
    
    for test_suite, success in results.items():
        status = "PASS" if success else "FAIL"
        print(f"{test_suite:<30} {status}")
    
    print(f"\nOverall Result: {'ALL TESTS PASSED' if all_passed else 'SOME TESTS FAILED'}")
    
    # Exit with appropriate code
    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    main()
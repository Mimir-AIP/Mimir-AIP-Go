"""Integration tests for ContextService persistence functionality.

This module tests the interaction between ContextService and its storage backends,
verifying proper persistence, loading, and cleanup of context data.

Test Categories:
    - Storage backend integration
    - Data persistence operations
    - Error handling and recovery
    - Performance benchmarks
    - Resource management

Note:
    Performance tests require the 'pytest-benchmark' plugin
"""

import os
import sys
import json
import shutil
import pytest
import tempfile
import logging
from pathlib import Path
from typing import Dict, Any, Optional
from unittest import TestCase
import psutil
from contextlib import contextmanager

from src.ContextService import ContextService

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Test data fixtures
TEST_CONTEXTS = {
    'simple': {'key1': 'value1', 'key2': 'value2'},
    'nested': {
        'user': {
            'name': 'Test User',
            'settings': {'theme': 'dark', 'notifications': True}
        },
        'data': [1, 2, 3]
    },
    'large': {f'key_{i}': f'value_{i}' for i in range(1000)}
}

class ResourceMonitor:
    """Monitors system resources during test execution."""
    
    def __init__(self):
        self.process = psutil.Process()
        self.start_memory = 0
        self.peak_memory = 0
        self.start_time = 0
        self.duration = 0

    def __enter__(self):
        self.start_memory = self.process.memory_info().rss
        self.start_time = pytest.approx(time.time())
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.duration = pytest.approx(time.time()) - self.start_time
        self.peak_memory = self.process.memory_info().rss - self.start_memory

@pytest.fixture
def storage_config():
    """Provides test storage configuration."""
    test_dir = tempfile.mkdtemp(prefix="mimir_test_")
    config = {
        "storage": {
            "enabled": True,
            "backend": "filesystem",
            "base_path": os.path.join(test_dir, "context_data")
        }
    }
    yield config
    shutil.rmtree(test_dir, ignore_errors=True)

@pytest.fixture
def context_service(storage_config):
    """Provides configured ContextService instance."""
    service = ContextService(storage_config)
    yield service
    service.cleanup()

@pytest.mark.integration
class TestContextPersistence(TestCase):
    """Test suite for ContextService persistence functionality.
    
    Tests the interaction between ContextService and storage backends,
    including data persistence, loading, and cleanup operations.
    
    Attributes:
        test_dir: Temporary directory for test data
        config: Storage backend configuration
        context_service: ContextService instance under test
    """

    def setUp(self):
        """Set up test environment before each test."""
        self.test_dir = tempfile.mkdtemp(prefix="mimir_test_")
        self.config = {
            "storage": {
                "enabled": True,
                "backend": "filesystem",
                "base_path": os.path.join(self.test_dir, "context_data")
            }
        }
        self.context_service = ContextService(self.config)

    def tearDown(self):
        """Clean up after each test."""
        self.context_service.cleanup()
        shutil.rmtree(self.test_dir, ignore_errors=True)

    # Core functionality tests
    @pytest.mark.core
    @pytest.mark.parametrize("context_name,context_data", TEST_CONTEXTS.items())
    def test_save_and_load_context(self, context_name: str, context_data: Dict[str, Any]):
        """Test saving and loading different types of context data.
        
        Args:
            context_name: Name of the test context
            context_data: Context data to test
        """
        # Set and save context
        self.context_service.set_context("test_ns", "data", context_data)
        self.assertTrue(self.context_service.save_to_storage("test_ns"))

        # Create new service instance and load data
        new_service = ContextService(self.config)
        self.assertTrue(new_service.load_from_storage("test_ns"))
        
        # Verify data
        loaded_data = new_service.get_context("test_ns", "data")
        self.assertEqual(loaded_data, context_data)
        new_service.cleanup()

    @pytest.mark.core
    def test_namespace_isolation(self):
        """Test that contexts in different namespaces remain isolated."""
        namespaces = ['ns1', 'ns2', 'ns3']
        for ns in namespaces:
            self.context_service.set_context(ns, "key", f"value_{ns}")
            self.assertTrue(self.context_service.save_to_storage(ns))

        new_service = ContextService(self.config)
        for ns in namespaces:
            self.assertTrue(new_service.load_from_storage(ns))
            self.assertEqual(new_service.get_context(ns, "key"), f"value_{ns}")
        new_service.cleanup()

    # Error handling tests
    @pytest.mark.error
    def test_invalid_storage_config(self):
        """Test handling of invalid storage configuration."""
        invalid_configs = [
            {"storage": {"enabled": True, "backend": "invalid"}},
            {"storage": {"enabled": True, "backend": "filesystem"}},  # Missing base_path
            {"storage": {"enabled": True}}  # Missing backend
        ]
        for config in invalid_configs:
            with self.assertRaises(ValueError):
                ContextService(config)

    @pytest.mark.error
    def test_storage_failures(self):
        """Test handling of storage operation failures."""
        # Test with read-only directory
        read_only_dir = os.path.join(self.test_dir, "readonly")
        os.makedirs(read_only_dir)
        os.chmod(read_only_dir, 0o444)  # Read-only

        config = {
            "storage": {
                "enabled": True,
                "backend": "filesystem",
                "base_path": read_only_dir
            }
        }
        service = ContextService(config)
        service.set_context("test", "key", "value")
        
        # Should fail gracefully
        self.assertFalse(service.save_to_storage("test"))
        service.cleanup()

    # Performance tests
    @pytest.mark.performance
    def test_save_performance(self, benchmark):
        """Benchmark save operations with large contexts."""
        large_data = TEST_CONTEXTS['large']
        self.context_service.set_context("perf_test", "data", large_data)

        def run_benchmark():
            self.context_service.save_to_storage("perf_test")

        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        # Verify performance meets requirements
        self.assertLess(monitor.duration, 1.0)  # Should take less than 1 second
        self.assertLess(monitor.peak_memory / (1024 * 1024), 50)  # 50MB max

    @pytest.mark.performance
    def test_load_performance(self, benchmark):
        """Benchmark load operations with large contexts."""
        large_data = TEST_CONTEXTS['large']
        self.context_service.set_context("perf_test", "data", large_data)
        self.context_service.save_to_storage("perf_test")

        def run_benchmark():
            service = ContextService(self.config)
            service.load_from_storage("perf_test")
            service.cleanup()

        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        self.assertLess(monitor.duration, 1.0)
        self.assertLess(monitor.peak_memory / (1024 * 1024), 50)

    # Resource cleanup tests
    @pytest.mark.cleanup
    def test_resource_cleanup(self):
        """Test proper cleanup of storage resources."""
        # Create and use multiple contexts
        for i in range(5):
            ns = f"test_ns_{i}"
            self.context_service.set_context(ns, "data", TEST_CONTEXTS['nested'])
            self.context_service.save_to_storage(ns)

        # Monitor resource usage during cleanup
        with ResourceMonitor() as monitor:
            self.context_service.cleanup()

        # Verify resources were freed
        self.assertLess(monitor.peak_memory / (1024 * 1024), 10)  # 10MB max during cleanup
        
        # Verify storage was cleaned up
        storage_path = os.path.join(self.test_dir, "context_data")
        self.assertFalse(os.path.exists(storage_path))

if __name__ == '__main__':
    pytest.main([__file__])
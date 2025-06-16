"""Unit tests for the PipelineVisualizer component.

This module tests the visualization capabilities of the PipelineVisualizer,
including ASCII tree generation, timing information display, and performance
characteristics with various pipeline structures.

Test Categories:
    - Basic visualization tests
    - Complex pipeline structure tests
    - Error handling and validation
    - Performance benchmarks
    - Resource management

Note:
    Performance tests require the 'pytest-benchmark' plugin
"""

import pytest
import unittest
import time
from typing import Dict, Any, List
import psutil
from pathlib import Path

from src.PipelineVisualizer.AsciiTree import PipelineAsciiTreeVisualizer

# Test fixtures
@pytest.fixture
def basic_pipeline():
    """Provides a basic pipeline structure for testing."""
    return {
        'root': {
            'name': 'Main Pipeline',
            'status': 'success',
            'children': ['process_data'],
            'start_time': 1620000000.0,
            'end_time': 1620000005.5
        },
        'process_data': {
            'name': 'Data Processing',
            'status': 'running',
            'start_time': 1620000001.0
        }
    }

@pytest.fixture
def complex_pipeline():
    """Provides a complex pipeline structure for testing."""
    return {
        'root': {
            'name': 'Complex Pipeline',
            'status': 'success',
            'children': ['fetch', 'process', 'output'],
            'start_time': 1620000000.0,
            'end_time': 1620000010.0
        },
        'fetch': {
            'name': 'Fetch Data',
            'status': 'success',
            'children': ['fetch_web', 'fetch_db'],
            'start_time': 1620000001.0,
            'end_time': 1620000003.0
        },
        'fetch_web': {
            'name': 'Web API',
            'status': 'success',
            'start_time': 1620000001.5,
            'end_time': 1620000002.0
        },
        'fetch_db': {
            'name': 'Database',
            'status': 'success',
            'start_time': 1620000002.0,
            'end_time': 1620000003.0
        },
        'process': {
            'name': 'Process Data',
            'status': 'running',
            'children': ['transform', 'validate'],
            'start_time': 1620000003.0
        },
        'transform': {
            'name': 'Transform',
            'status': 'running',
            'start_time': 1620000003.5
        },
        'validate': {
            'name': 'Validate',
            'status': 'pending'
        },
        'output': {
            'name': 'Output',
            'status': 'pending',
            'children': ['save_db', 'notify']
        },
        'save_db': {
            'name': 'Save to DB',
            'status': 'pending'
        },
        'notify': {
            'name': 'Send Notification',
            'status': 'pending'
        }
    }

class ResourceMonitor:
    """Monitors system resources during test execution."""
    
    def __init__(self):
        self.process = psutil.Process()
        self.start_memory = 0
        self.peak_memory = 0
    
    def __enter__(self):
        self.start_memory = self.process.memory_info().rss
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.peak_memory = self.process.memory_info().rss - self.start_memory

@pytest.mark.unit
class TestPipelineVisualizer(unittest.TestCase):
    """Test suite for PipelineVisualizer.
    
    Tests visualization capabilities, error handling, and performance
    characteristics of the pipeline visualization component.
    """

    def setUp(self):
        """Set up test fixtures before each test."""
        self.visualizer = PipelineAsciiTreeVisualizer()

    # Basic visualization tests
    @pytest.mark.core
    def test_basic_tree_structure(self, basic_pipeline):
        """Test basic ASCII tree structure generation.
        
        Verifies correct tree structure and formatting for a simple pipeline.
        """
        self.visualizer.show_timings = True
        result = self.visualizer.generate_tree(basic_pipeline)
        
        # Verify structure
        assert 'Main Pipeline' in result
        assert 'Data Processing' in result
        
        # Verify timing displays
        assert '(5.5s)' in result  # Completed duration
        assert '(running)' in result  # Active node
        
        # Verify status icons
        assert '✔' in result  # Success icon
        assert '⌛' in result  # Running icon

    @pytest.mark.core
    def test_complex_tree_structure(self, complex_pipeline):
        """Test visualization of complex pipeline structures.
        
        Verifies correct tree structure and formatting for a complex
        pipeline with multiple levels and parallel branches.
        """
        result = self.visualizer.generate_tree(complex_pipeline)
        
        # Verify structure elements
        assert 'Complex Pipeline' in result
        assert 'Fetch Data' in result
        assert 'Process Data' in result
        assert 'Output' in result
        
        # Verify branch handling
        assert '├─' in result  # Non-last child connector
        assert '└─' in result  # Last child connector
        
        # Verify status displays
        assert '(running)' in result
        assert '(pending)' in result

    # Error handling tests
    @pytest.mark.error
    @pytest.mark.parametrize("invalid_input", [
        {},  # Empty pipeline
        {'root': {}},  # Missing required fields
        {'root': {'name': 'Test'}},  # Missing status
        None,  # None input
        {'not_root': {'name': 'Test', 'status': 'success'}}  # Missing root
    ])
    def test_error_handling(self, invalid_input):
        """Test error handling for invalid inputs.
        
        Args:
            invalid_input: Invalid pipeline structure to test
        """
        with pytest.raises((ValueError, KeyError)):
            self.visualizer.generate_tree(invalid_input)

    # Performance tests
    @pytest.mark.performance
    def test_large_pipeline_performance(self, benchmark):
        """Benchmark visualization performance with large pipelines."""
        def create_large_pipeline(size: int) -> Dict[str, Any]:
            """Creates a large pipeline structure for testing."""
            pipeline = {
                'root': {
                    'name': 'Large Pipeline',
                    'status': 'success',
                    'children': [f'node_{i}' for i in range(size)],
                    'start_time': time.time(),
                    'end_time': time.time() + 10
                }
            }
            for i in range(size):
                pipeline[f'node_{i}'] = {
                    'name': f'Node {i}',
                    'status': 'success',
                    'start_time': time.time(),
                    'end_time': time.time() + 1
                }
            return pipeline

        large_pipeline = create_large_pipeline(100)
        
        def run_benchmark():
            self.visualizer.generate_tree(large_pipeline)
        
        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        # Verify performance meets requirements
        self.assertLess(monitor.peak_memory / (1024 * 1024), 10)  # 10MB max

    @pytest.mark.performance
    def test_deep_pipeline_performance(self, benchmark):
        """Benchmark visualization performance with deeply nested pipelines."""
        def create_deep_pipeline(depth: int) -> Dict[str, Any]:
            """Creates a deeply nested pipeline structure for testing."""
            pipeline = {}
            current_id = 'root'
            for i in range(depth):
                pipeline[current_id] = {
                    'name': f'Level {i}',
                    'status': 'success',
                    'children': [f'node_{i}'],
                    'start_time': time.time(),
                    'end_time': time.time() + 1
                }
                current_id = f'node_{i}'
            pipeline[current_id] = {
                'name': f'Level {depth}',
                'status': 'success',
                'start_time': time.time(),
                'end_time': time.time() + 1
            }
            return pipeline

        deep_pipeline = create_deep_pipeline(50)
        
        def run_benchmark():
            self.visualizer.generate_tree(deep_pipeline)
        
        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        self.assertLess(monitor.peak_memory / (1024 * 1024), 10)

    # Display configuration tests
    @pytest.mark.display
    def test_timing_display_options(self, basic_pipeline):
        """Test timing display configuration options."""
        # Test with timings enabled
        self.visualizer.show_timings = True
        result_with_timing = self.visualizer.generate_tree(basic_pipeline)
        assert '(5.5s)' in result_with_timing
        
        # Test with timings disabled
        self.visualizer.show_timings = False
        result_without_timing = self.visualizer.generate_tree(basic_pipeline)
        assert '(5.5s)' not in result_without_timing

    @pytest.mark.display
    def test_status_icon_consistency(self, complex_pipeline):
        """Test consistency of status icons across different node states."""
        result = self.visualizer.generate_tree(complex_pipeline)
        
        # Verify all status types have appropriate icons
        assert '✔' in result  # Success
        assert '⌛' in result  # Running
        assert '⏳' in result  # Pending

if __name__ == '__main__':
    pytest.main([__file__])
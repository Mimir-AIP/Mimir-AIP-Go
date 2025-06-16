"""Unit tests for the ContextValidator.

This module contains comprehensive tests for the ContextValidator component,
which handles validation of context data against defined schemas.

Test Categories:
    - Schema validation tests
    - Data type validation
    - Error handling and validation
    - Performance benchmarks
    - Resource management

Note:
    Performance tests require the 'pytest-benchmark' plugin
"""

import pytest
import unittest
import tempfile
import json
from pathlib import Path
from typing import Dict, Any, List
import psutil
from dataclasses import dataclass

from src.ContextValidator import ContextValidator
from src.data_types import BinaryData, DataReference

# Test fixtures
@pytest.fixture
def validator():
    """Provides a fresh ContextValidator instance for each test."""
    return ContextValidator()

@pytest.fixture
def complex_schema():
    """Provides a complex schema for thorough validation testing."""
    return {
        "type": "object",
        "properties": {
            "string_field": {"type": "string", "minLength": 1},
            "number_field": {"type": "number", "minimum": 0},
            "array_field": {
                "type": "array",
                "items": {"type": "string"},
                "minItems": 1
            },
            "object_field": {
                "type": "object",
                "properties": {
                    "nested_field": {"type": "string"}
                },
                "required": ["nested_field"]
            },
            "binary_field": {"type": "binary"},
            "reference_field": {"type": "reference"}
        },
        "required": ["string_field", "number_field"]
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
class TestContextValidation(unittest.TestCase):
    """Test suite for ContextValidator.
    
    Tests schema validation, data type validation, error handling,
    and performance characteristics of the ContextValidator implementation.
    """

    def setUp(self):
        """Set up test fixtures before each test."""
        self.validator = ContextValidator()
        self.temp_dir = Path(tempfile.mkdtemp())

    def tearDown(self):
        """Clean up after each test."""
        import shutil
        shutil.rmtree(self.temp_dir)

    # Core validation tests
    @pytest.mark.core
    def test_binary_data_validation(self):
        """Test validation of BinaryData type."""
        schema = {"type": "object", "properties": {"image": {"type": "binary"}}}
        data = {"image": BinaryData(b"image_data", "image/jpeg")}
        self.assertTrue(self.validator.validate(data, schema))

    @pytest.mark.core
    def test_reference_validation(self):
        """Test validation of DataReference type."""
        schema = {"type": "object", "properties": {"ref": {"type": "reference"}}}
        data = {"ref": DataReference("data_key")}
        self.assertTrue(self.validator.validate(data, schema))

    @pytest.mark.core
    @pytest.mark.parametrize("test_input,expected", [
        ({"user": {"name": "Alice", "age": 30, "active": True}}, True),
        ({"user": {"name": "Bob", "age": "invalid"}}, False),
        ({"user": {"name": "", "age": -1}}, False),
        ({"user": {}}, False),
    ])
    def test_complex_structure_validation(self, test_input: Dict[str, Any], expected: bool):
        """Test validation of complex nested structures.
        
        Args:
            test_input: Test data to validate
            expected: Expected validation result
        """
        schema = {
            "type": "object",
            "properties": {
                "user": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string", "minLength": 1},
                        "age": {"type": "number", "minimum": 0},
                        "active": {"type": "boolean"}
                    },
                    "required": ["name", "age"]
                }
            }
        }
        self.assertEqual(self.validator.validate(test_input, schema), expected)

    # Error handling tests
    @pytest.mark.error
    def test_invalid_schema(self):
        """Test validation with invalid schema definitions."""
        invalid_schemas = [
            {"type": "invalid_type"},
            {"type": "object", "properties": "not_an_object"},
            {"type": "array", "items": None}
        ]
        data = {"test": "value"}
        for schema in invalid_schemas:
            with self.assertRaises(ValueError):
                self.validator.validate(data, schema)

    @pytest.mark.error
    def test_schema_type_mismatches(self):
        """Test validation when data types don't match schema."""
        schema = {"type": "object", "properties": {"field": {"type": "string"}}}
        invalid_data = [
            {"field": 123},
            {"field": []},
            {"field": {}}
        ]
        for data in invalid_data:
            self.assertFalse(self.validator.validate(data, schema))

    # Performance tests
    @pytest.mark.performance
    def test_validation_performance(self, benchmark):
        """Benchmark validation performance with large schemas."""
        large_schema = {
            "type": "object",
            "properties": {
                f"field_{i}": {"type": "string"} for i in range(100)
            }
        }
        large_data = {
            f"field_{i}": f"value_{i}" for i in range(100)
        }

        def run_benchmark():
            for _ in range(1000):
                self.validator.validate(large_data, large_schema)

        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        # Verify performance meets requirements
        self.assertLess(monitor.peak_memory / (1024 * 1024), 10)  # 10MB max

    @pytest.mark.performance
    def test_nested_validation_performance(self, benchmark):
        """Benchmark validation performance with deeply nested structures."""
        def create_nested_schema(depth: int) -> Dict[str, Any]:
            if depth == 0:
                return {"type": "string"}
            return {
                "type": "object",
                "properties": {
                    "nested": create_nested_schema(depth - 1)
                }
            }

        def create_nested_data(depth: int) -> Dict[str, Any]:
            if depth == 0:
                return "value"
            return {"nested": create_nested_data(depth - 1)}

        schema = create_nested_schema(10)
        data = create_nested_data(10)

        def run_benchmark():
            for _ in range(100):
                self.validator.validate(data, schema)

        with ResourceMonitor() as monitor:
            benchmark(run_benchmark)
            
        self.assertLess(monitor.peak_memory / (1024 * 1024), 10)

    # Custom type validation tests
    @pytest.mark.types
    @pytest.mark.parametrize("test_type,valid_value,invalid_value", [
        ("binary", BinaryData(b"data", "text/plain"), "not_binary"),
        ("reference", DataReference("key"), "not_reference"),
        ("number", 42.0, "not_number"),
        ("boolean", True, "not_boolean"),
        ("array", [], "not_array"),
        ("object", {}, "not_object")
    ])
    def test_type_validation(self, test_type: str, valid_value: Any, invalid_value: Any):
        """Test validation of various data types.
        
        Args:
            test_type: Schema type to test
            valid_value: Value that should pass validation
            invalid_value: Value that should fail validation
        """
        schema = {"type": test_type}
        self.assertTrue(self.validator.validate(valid_value, schema))
        self.assertFalse(self.validator.validate(invalid_value, schema))

if __name__ == '__main__':
    pytest.main([__file__])
import unittest
from src.ContextValidator import ContextValidator
from src.data_types import BinaryData, DataReference

class TestContextValidation(unittest.TestCase):
    def test_binary_data_validation(self):
        validator = ContextValidator()
        schema = {"type": "object", "properties": {"image": {"type": "binary"}}}
        data = {"image": BinaryData(b"image_data", "image/jpeg")}
        self.assertTrue(validator.validate(data, schema))
    
    def test_reference_validation(self):
        validator = ContextValidator()
        schema = {"type": "object", "properties": {"ref": {"type": "reference"}}}
        data = {"ref": DataReference("data_key")}
        self.assertTrue(validator.validate(data, schema))
    
    def test_complex_structure_validation(self):
        validator = ContextValidator()
        schema = {
            "type": "object",
            "properties": {
                "user": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"},
                        "age": {"type": "number"},
                        "active": {"type": "boolean"}
                    }
                }
            }
        }
        data = {"user": {"name": "Alice", "age": 30, "active": True}}
        self.assertTrue(validator.validate(data, schema))

if __name__ == '__main__':
    unittest.main()
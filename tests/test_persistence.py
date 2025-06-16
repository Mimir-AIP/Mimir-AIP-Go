"""
Test script for ContextService persistence functionality.
"""
import os
import shutil
import tempfile
import unittest


from src.ContextService import ContextService

class TestContextPersistence(unittest.TestCase):
    """Test cases for ContextService persistence."""
    
    def setUp(self):
        """Set up test environment."""
        # Create a temporary directory for testing
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
        """Clean up test environment."""
        # Remove the temporary directory after tests
        shutil.rmtree(self.test_dir, ignore_errors=True)
    
    def test_save_and_load_context(self):
        """Test saving and loading context to/from persistent storage."""
        # Set some test data
        self.context_service.set_context("test_ns", "key1", "value1")
        self.context_service.set_context("test_ns", "key2", {"nested": "value"})
        
        # Save to storage
        self.assertTrue(self.context_service.save_to_storage("test_ns"))
        
        # Create a new context service to simulate a restart
        new_context = ContextService(self.config)
        
        # Load from storage
        self.assertTrue(new_context.load_from_storage("test_ns"))
        
        # Verify the data was loaded correctly
        self.assertEqual(new_context.get_context("test_ns", "key1"), "value1")
        self.assertEqual(new_context.get_context("test_ns", "key2"), {"nested": "value"})
    
    def test_save_and_load_all_contexts(self):
        """Test saving and loading all contexts to/from persistent storage."""
        # Set test data in multiple namespaces
        self.context_service.set_context("ns1", "key1", "value1")
        self.context_service.set_context("ns2", "key2", "value2")
        
        # Save all contexts to storage
        self.assertTrue(self.context_service.save_to_storage())
        
        # Create a new context service to simulate a restart
        new_context = ContextService(self.config)
        
        # Load all contexts from storage
        self.assertTrue(new_context.load_from_storage("ns1"))
        self.assertTrue(new_context.load_from_storage("ns2"))
        
        # Verify the data was loaded correctly
        self.assertEqual(new_context.get_context("ns1", "key1"), "value1")
        self.assertEqual(new_context.get_context("ns2", "key2"), "value2")
    
    def test_delete_from_storage(self):
        """Test deleting context from persistent storage."""
        # Set test data
        self.context_service.set_context("test_del", "key1", "value1")
        self.context_service.save_to_storage("test_del")
        
        # Verify the data exists
        self.assertTrue(self.context_service.load_from_storage("test_del"))
        self.assertIsNotNone(self.context_service.get_context("test_del", "key1"))
        
        # Delete the data
        self.assertTrue(self.context_service.delete_from_storage("test_del"))
        
        # Verify the data was deleted
        self.assertFalse(self.context_service.load_from_storage("test_del"))
        self.assertIsNone(self.context_service.get_context("test_del", "key1"))

if __name__ == "__main__":
    unittest.main()

"""
Mock-based tests for the WebInterface plugin.

These tests verify the functionality of the WebInterface plugin
without starting an actual HTTP server.
"""
import os
import sys
import unittest
import tempfile
import shutil
from pathlib import Path
from unittest.mock import patch, MagicMock, ANY

# Add src directory to Python path
src_dir = str(Path(__file__).parent.parent / 'src')
if src_dir not in sys.path:
    sys.path.insert(0, src_dir)

# Import the WebInterface after setting up the path
from Plugins.Web.WebInterface import WebInterface
from Plugins.BasePlugin import BasePlugin

class TestWebInterfacePluginMocked(unittest.TestCase):
    """Test cases for the WebInterface plugin using mocks."""
    
    @classmethod
    def setUpClass(cls):
        """Set up test fixtures before any tests are run."""
        # Create a temporary directory for test files
        cls.test_dir = tempfile.mkdtemp(prefix='mimir_web_test_')
        cls.static_dir = os.path.join(cls.test_dir, 'static')
        os.makedirs(cls.static_dir, exist_ok=True)
    
    @classmethod
    def tearDownClass(cls):
        """Clean up after all tests have run."""
        shutil.rmtree(cls.test_dir, ignore_errors=True)
    
    def setUp(self):
        """Set up test fixtures before each test method is called."""
        # Create a mock for the server and thread
        self.mock_server = MagicMock()
        self.mock_thread = MagicMock()
        self.mock_thread.is_alive.return_value = True
        
        # Patch the server and thread creation
        self.server_patcher = patch('socketserver.TCPServer', return_value=self.mock_server)
        self.thread_patcher = patch('threading.Thread', return_value=self.mock_thread)
        
        self.mock_server_class = self.server_patcher.start()
        self.mock_thread_class = self.thread_patcher.start()
        
        # Create the plugin instance
        self.plugin = WebInterface(port=8081)
    
    def tearDown(self):
        """Clean up after each test method is called."""
        self.server_patcher.stop()
        self.thread_patcher.stop()
    
    def test_initialization(self):
        """Test that the WebInterface initializes correctly with mocks."""
        # Verify the server was created with the correct port
        self.mock_server_class.assert_called_once()
        args, kwargs = self.mock_server_class.call_args
        self.assertEqual(args[0], ("", 8081))
        
        # Verify the thread was started
        self.mock_thread_class.assert_called_once()
        self.mock_thread.start.assert_called_once()
        
        # Verify plugin properties
        self.assertEqual(self.plugin.port, 8081)
        self.assertEqual(self.plugin.server, self.mock_server)
        self.assertEqual(self.plugin.server_thread, self.mock_thread)
        self.assertIsInstance(self.plugin, BasePlugin)
    
    def test_add_section(self):
        """Test adding a section to the dashboard."""
        section_id = "test_section"
        content = {"title": "Test Section", "content": "Test Content"}
        
        # Add a section
        self.plugin.add_section(section_id, content)
        
        # Verify the section was added
        self.assertIn(section_id, self.plugin.dashboard_sections)
        self.assertEqual(self.plugin.dashboard_sections[section_id], content)
    
    def test_remove_section(self):
        """Test removing a section from the dashboard."""
        section_id = "test_section"
        content = {"title": "Test Section", "content": "Test Content"}
        
        # Add and then remove a section
        self.plugin.add_section(section_id, content)
        self.plugin.remove_section(section_id)
        
        # Verify the section was removed
        self.assertNotIn(section_id, self.plugin.dashboard_sections)
    
    def test_cleanup(self):
        """Test that cleanup shuts down the server properly."""
        # Call cleanup
        self.plugin._cleanup()
        
        # Verify the server was shut down
        self.mock_server.shutdown.assert_called_once()
        self.mock_server.server_close.assert_called_once()
    
    def test_execute_pipeline_step_add_section(self):
        """Test executing a pipeline step to add a section."""
        step_config = {
            "operation": "section_add",
            "config": {
                "id": "test_section",
                "content": {"title": "Test", "content": "Test Content"}
            }
        }
        context = {}
        
        # Execute the step
        result = self.plugin.execute_pipeline_step(step_config, context)
        
        # Verify the result and that the section was added
        self.assertEqual(result, context)
        self.assertIn("test_section", self.plugin.dashboard_sections)
    
    def test_execute_pipeline_step_remove_section(self):
        """Test executing a pipeline step to remove a section."""
        # First add a section
        self.plugin.add_section("test_section", {"title": "Test", "content": "Test Content"})
        
        # Then remove it via a pipeline step
        step_config = {
            "operation": "section_remove",
            "config": {"id": "test_section"}
        }
        context = {}
        
        # Execute the step
        result = self.plugin.execute_pipeline_step(step_config, context)
        
        # Verify the result and that the section was removed
        self.assertEqual(result, context)
        self.assertNotIn("test_section", self.plugin.dashboard_sections)


if __name__ == '__main__':
    unittest.main()

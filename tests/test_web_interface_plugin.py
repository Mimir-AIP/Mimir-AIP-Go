"""
Unit tests for the WebInterface plugin.

These tests verify the functionality of the WebInterface plugin,
including server initialization, request handling, and plugin integration.
"""
import os
import sys
import json
import time
import socket
import threading
import unittest
import tempfile
import shutil
import signal
import functools
from pathlib import Path
from unittest.mock import patch, MagicMock, ANY
from http.server import HTTPServer
from http import HTTPStatus
from urllib.parse import urlparse, parse_qs

# Add src directory to Python path
src_dir = str(Path(__file__).parent.parent / 'src')
if src_dir not in sys.path:
    sys.path.insert(0, src_dir)

# Import the WebInterface after setting up the path
from Plugins.Web.WebInterface import WebInterface
from Plugins.BasePlugin import BasePlugin

# Test configuration
TEST_PORT = 8081  # Default test port
TEST_TIMEOUT = 5  # seconds

def timeout(seconds):
    """Timeout decorator to prevent tests from hanging."""
    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # Set the signal handler and a 5-second alarm
            signal.signal(signal.SIGALRM, lambda signum, frame: (_ for _ in ()).throw(TimeoutError(f"Test timed out after {seconds} seconds")))
            signal.alarm(seconds)
            try:
                result = func(*args, **kwargs)
            finally:
                # Disable the alarm
                signal.alarm(0)
            return result
        return wrapper
    return decorator

class TestWebInterfacePlugin(unittest.TestCase):
    """Test cases for the WebInterface plugin."""
    
    @classmethod
    def setUpClass(cls):
        """Set up test fixtures before any tests are run."""
        # Create a temporary directory for test files
        cls.test_dir = tempfile.mkdtemp(prefix='mimir_web_test_')
        cls.static_dir = os.path.join(cls.test_dir, 'static')
        os.makedirs(cls.static_dir, exist_ok=True)
        
        # Create a basic test file in the static directory
        cls.test_file = os.path.join(cls.static_dir, 'test.txt')
        with open(cls.test_file, 'w') as f:
            f.write('Test content')
    
    @classmethod
    def tearDownClass(cls):
        """Clean up after all tests have run."""
        # Remove the temporary directory and all its contents
        shutil.rmtree(cls.test_dir, ignore_errors=True)
    
    def setUp(self):
        """Set up test fixtures before each test method is called."""
        # Use a unique port for each test to avoid conflicts
        self.port = self.find_free_port()
        self.plugin = None
    
    def tearDown(self):
        """Clean up after each test method is called."""
        if self.plugin and hasattr(self.plugin, '_cleanup'):
            self.plugin._cleanup()
    
    def find_free_port(self):
        """Find a free port for testing."""
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(('', 0))
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            return s.getsockname()[1]
    
    @timeout(TEST_TIMEOUT)
    def test_initialization(self):
        """Test that the WebInterface initializes correctly."""
        try:
            with patch('logging.info') as mock_logging:
                self.plugin = WebInterface(port=self.port)
                
                # Verify basic attributes are set
                self.assertEqual(self.plugin.port, self.port)
                self.assertIsNotNone(self.plugin.server_thread)
                self.assertTrue(self.plugin.server_thread.is_alive())
                self.assertIsInstance(self.plugin, BasePlugin)
                
                # Verify logging was called
                mock_logging.assert_any_call(f"[WebInterface] Server started successfully on port {self.port}")
        finally:
            if hasattr(self, 'plugin') and self.plugin:
                self.plugin._cleanup()
    
    @timeout(TEST_TIMEOUT)
    def test_port_in_use_handling(self):
        """Test that the WebInterface handles port conflicts gracefully."""
        test_server = None
        server_thread = None
        
        try:
            # First, start a server on a specific port
            test_port = self.find_free_port()
            test_server = HTTPServer(('', test_port), MagicMock())
            server_thread = threading.Thread(target=test_server.serve_forever, daemon=True)
            server_thread.start()
            
            # Small delay to ensure server is up
            time.sleep(0.5)
            
            # Try to create a WebInterface on the same port
            with patch('logging.error') as mock_logging:
                self.plugin = WebInterface(port=test_port)
                
                # The plugin should try to use a different port
                self.assertNotEqual(self.plugin.port, test_port)
                self.assertTrue(mock_logging.called)
        except Exception as e:
            self.fail(f"Test failed with exception: {str(e)}")
        finally:
            # Clean up the test server
            if test_server:
                test_server.shutdown()
            if server_thread:
                server_thread.join(timeout=1)
            if hasattr(self, 'plugin') and self.plugin:
                self.plugin._cleanup()
    
    @timeout(TEST_TIMEOUT)
    def test_add_and_remove_section(self):
        """Test adding and removing dashboard sections."""
        try:
            self.plugin = WebInterface(port=self.port)
            
            # Add a section
            section_id = "test_section"
            content = {"title": "Test Section", "content": "Test Content"}
            self.plugin.add_section(section_id, content)
            
            # Verify the section was added
            self.assertIn(section_id, self.plugin.dashboard_sections)
            self.assertEqual(self.plugin.dashboard_sections[section_id], content)
            
            # Remove the section
            self.plugin.remove_section(section_id)
            self.assertNotIn(section_id, self.plugin.dashboard_sections)
        finally:
            if hasattr(self, 'plugin') and self.plugin:
                self.plugin._cleanup()
    
    @patch('socketserver.TCPServer')
    def test_cleanup(self, mock_server):
        """Test that cleanup properly shuts down the server."""
        # Create a mock server instance
        mock_server_instance = MagicMock()
        mock_server.return_value = mock_server_instance
        
        # Create the plugin
        self.plugin = WebInterface(port=self.port)
        
        # Call cleanup
        self.plugin._cleanup()
        
        # Verify the server was shut down
        mock_server_instance.shutdown.assert_called_once()
        mock_server_instance.server_close.assert_called_once()
    
    @timeout(TEST_TIMEOUT)
    def test_execute_pipeline_step(self):
        """Test the execute_pipeline_step method with different operations."""
        try:
            self.plugin = WebInterface(port=self.port)
            
            # Test section_add operation
            step_config = {
                "operation": "section_add",
                "config": {
                    "id": "test_section",
                    "content": {"title": "Test", "content": "Test Content"}
                }
            }
            context = {}
            
            result = self.plugin.execute_pipeline_step(step_config, context)
            self.assertEqual(result, context)  # Should return the context unchanged
            self.assertIn("test_section", self.plugin.dashboard_sections)
            
            # Test section_remove operation
            step_config = {
                "operation": "section_remove",
                "config": {"id": "test_section"}
            }
            self.plugin.execute_pipeline_step(step_config, context)
            self.assertNotIn("test_section", self.plugin.dashboard_sections)
        finally:
            if hasattr(self, 'plugin') and self.plugin:
                self.plugin._cleanup()


class TestWebInterfaceRequestHandler(unittest.TestCase):
    """Test cases for the WebInterfaceRequestHandler."""
    
    def setUp(self):
        """Set up test fixtures before each test method is called."""
        self.plugin = MagicMock()
        self.plugin.static_dir = os.path.join(os.path.dirname(__file__), 'test_static')
        os.makedirs(self.plugin.static_dir, exist_ok=True)
        
        # Create a test file in the static directory
        self.test_file = os.path.join(self.plugin.static_dir, 'test.txt')
        with open(self.test_file, 'w') as f:
            f.write('Test content')
    
    def tearDown(self):
        """Clean up after each test method is called."""
        if os.path.exists(self.plugin.static_dir):
            shutil.rmtree(self.plugin.static_dir)
    
    @patch('http.server.SimpleHTTPRequestHandler.send_response')
    @patch('http.server.SimpleHTTPRequestHandler.end_headers')
    @patch('http.server.SimpleHTTPRequestHandler.wfile.write')
    def test_serve_static_file(self, mock_write, mock_end_headers, mock_send_response):
        """Test serving a static file."""
        # Import here to avoid import issues with the patched modules
        from Plugins.Web.WebInterface import WebInterfaceRequestHandler
        
        # Create a mock request
        handler = MagicMock()
        handler.path = '/static/test.txt'
        handler.server = MagicMock()
        handler.send_header = MagicMock()
        handler.wfile = MagicMock()
        
        # Call the method directly
        WebInterfaceRequestHandler._serve_static_file(handler, 'test.txt')
        
        # Verify the file was served
        mock_send_response.assert_called_with(200)
        handler.send_header.assert_any_call('Content-type', 'text/plain')
        handler.send_header.assert_any_call('Content-Length', '12')
        mock_end_headers.assert_called_once()
        self.assertTrue(mock_write.called)


if __name__ == '__main__':
    unittest.main()

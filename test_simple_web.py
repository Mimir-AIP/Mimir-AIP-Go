"""
Test script for SimpleWebServer plugin.
"""
import sys
import time
import logging
import threading
import requests
from pathlib import Path

# Add src directory to Python path
src_dir = str(Path(__file__).parent / 'src')
sys.path.insert(0, src_dir)

from Plugins.Web.SimpleWebServer import SimpleWebServer

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

def test_simple_webserver():
    """Test the simple web server functionality."""
    # Create and start the server
    server = SimpleWebServer(port=8080)
    
    # Add a test route
    def handle_test(handler):
        return 200, {"message": "Test endpoint working!"}
    
    server.add_route("GET", "/test", handle_test)
    
    # Start the server in a separate thread
    server_thread = threading.Thread(target=server.start)
    server_thread.daemon = True
    server_thread.start()
    
    # Give the server a moment to start
    time.sleep(1)
    
    try:
        # Test the root endpoint
        response = requests.get("http://localhost:8080/")
        print(f"Root endpoint status: {response.status_code}")
        print(f"Response: {response.text}")
        
        # Test the health endpoint
        response = requests.get("http://localhost:8080/health")
        print(f"Health endpoint status: {response.status_code}")
        print(f"Response: {response.json()}")
        
        # Test the custom test endpoint
        response = requests.get("http://localhost:8080/test")
        print(f"Test endpoint status: {response.status_code}")
        print(f"Response: {response.json()}")
        
        # Keep the server running for a while
        print("Server is running. Press Ctrl+C to stop...")
        while True:
            time.sleep(1)
            
    except KeyboardInterrupt:
        print("\nStopping server...")
    finally:
        server.stop()
        server_thread.join(timeout=2)
        print("Server stopped.")

if __name__ == "__main__":
    test_simple_webserver()

import pytest
import sys
from pathlib import Path
sys.path.append(str(Path(__file__).parent.parent))
from src.Plugins.WebInterface.WebInterface import app
from fastapi.testclient import TestClient

@pytest.fixture
def client():
    return TestClient(app)

class TestWebInterface:
    def test_file_upload(self, client):
        test_file = ('test.txt', b'Test content')
        response = client.post('/upload', files={'file': test_file})
        assert response.status_code == 200
        assert 'file_id' in response.json()

    def test_websocket_connection(self, client):
        with client.websocket_connect("/ws") as websocket:
            data = websocket.receive_json()
            assert data["type"] == "connection_ack"
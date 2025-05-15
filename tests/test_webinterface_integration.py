import pytest
import sys
from pathlib import Path
sys.path.append(str(Path(__file__).parent.parent))
from src.Plugins.WebInterface.WebInterface import app
from fastapi.testclient import TestClient
import websockets
import asyncio

@pytest.fixture
def client():
    return TestClient(app)

class TestWebInterfaceIntegration:
    def test_llm_chat_flow(self, client):
        # Test complete LLM chat flow
        response = client.post('/llm-query', json={'message': 'Test'})
        assert response.status_code == 200
        assert 'response' in response.json()

    @pytest.mark.asyncio
    async def test_websocket_pipeline_updates(self):
        # Test real-time pipeline updates
        async with websockets.connect('ws://localhost:8000/ws') as websocket:
            await websocket.send('{"type": "subscribe", "channel": "pipeline"}')
            response = await websocket.recv()
            assert '"type":"pipeline_update"' in response
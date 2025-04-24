import dotenv
import os
import pytest
from Plugins.AIModels.GitHubModels.GitHubModels import GitHubModels

dotenv.load_dotenv(os.path.join(os.path.dirname(__file__), "..", "src", "Plugins", "AIModels", "GitHubModels", ".env"))

pytestmark = pytest.mark.usefixtures("monkeypatch")

@pytest.fixture(scope="module")
def plugin():
    """Fixture to instantiate the GitHubModels plugin."""
    return GitHubModels()

def test_get_available_models(plugin):
    """Test that get_available_models returns the correct hardcoded list."""
    models = plugin.get_available_models()
    assert isinstance(models, list)
    assert "openai/gpt-4.1" in models
    assert "anthropic/claude-3-5-sonnet" in models
    assert "google/gemini-2.5-pro" in models

def test_execute_pipeline_step_missing_token(monkeypatch):
    """Test that plugin raises if GITHUB_PAT is missing."""
    # Patch os.getenv to return None for GITHUB_PAT
    monkeypatch.setattr(os, "getenv", lambda key, default=None: None if key == "GITHUB_PAT" else os.environ.get(key, default))
    with pytest.raises(ValueError):
        GitHubModels()

@pytest.mark.skipif(not os.getenv("GITHUB_PAT"), reason="GITHUB_PAT not set; live API test skipped.")
def test_execute_pipeline_step_gpt41(plugin):
    """
    Live test: Call the GPT-4.1 model with a simple prompt and check the response.
    """
    step_config = {
        "plugin": "GitHubModels",
        "config": {
            "model": "openai/gpt-4.1",
            "endpoint": "https://models.github.ai/inference",
            "messages": [
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "What is the capital of France?"}
            ]
        },
        "output": "response"
    }
    context = {}
    result = plugin.execute_pipeline_step(step_config, context)
    assert "response" in result
    assert isinstance(result["response"], str)
    assert "paris" in result["response"].lower()

@pytest.mark.skipif(not os.getenv("GITHUB_PAT"), reason="GITHUB_PAT not set; live API test skipped.")
def test_execute_pipeline_step_invalid_model(plugin):
    """
    Live test: Call with an invalid model and ensure error handling works.
    """
    step_config = {
        "plugin": "GitHubModels",
        "config": {
            "model": "openai/does-not-exist",
            "endpoint": "https://models.github.ai/inference",
            "messages": [
                {"role": "user", "content": "Test"}
            ]
        },
        "output": "response"
    }
    context = {}
    with pytest.raises(RuntimeError):
        plugin.execute_pipeline_step(step_config, context)

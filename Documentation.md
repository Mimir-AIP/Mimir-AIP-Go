# Mimir-AIP Documentation

## Project Overview
Mimir-AIP is a modular, plugin-driven pipeline framework for automating data processing, AI/LLM tasks, and report generation. It is designed for extensibility, robust error handling, and flexible testing.

---

## Getting Started

### Installation
- Clone the repository: `git clone <repo_url>`
- Install dependencies: `pip install -r requirements.txt`
- (Optional) Set up a Python virtual environment for isolation.

### Project Structure
```
Mimir-AIP/
├── src/
│   ├── Plugins/           # All plugin code (AIModels, Input, Output, Data_Processing, etc.)
│   ├── pipelines/         # Pipeline YAML definitions
│   ├── config.yaml        # Main configuration file
│   └── main.py            # Pipeline runner
├── tests/                 # Unit and integration tests
├── requirements.txt       # Python dependencies
├── Documentation.md       # (This file)
└── ...
```

---

## How to Create and Configure Pipelines

- Pipelines are defined as YAML files in `src/pipelines/`.
- Each pipeline is a sequence of steps, each step specifying a plugin, config, and output.
- Example pipeline step:
```yaml
- name: "Generate Section Summary"
  plugin: "LLMFunction"
  config:
    prompt: "Summarize this section: {section_text}"
    model: "openrouter/mistral-7b"
    mock_response: {"section_summary": "This is a mock section summary for testing."}  # For test mode
  output: "section_summary"
```
- Reference your pipeline in `src/config.yaml`:
```yaml
pipelines:
  - name: "BBC News Pipeline"
    file: "pipelines/POC.yaml"
    enabled: true
```

---

## How to Add API Keys (e.g., OpenRouter)

- API keys are loaded via environment variables.
- For OpenRouter, set `OPENROUTER_API_KEY` in your environment:
  - On Mac/Linux: `export OPENROUTER_API_KEY=your_key_here`
  - On Windows: `set OPENROUTER_API_KEY=your_key_here`
- (Optional) Use a `.env` file and a loader like `python-dotenv` if supported.
- The OpenRouter plugin will raise an error if the key is missing.

---

## How to Create New Plugins

- Plugins live under `src/Plugins/<Type>/<PluginName>/<PluginName>.py`.
- Types include: `AIModels`, `Input`, `Output`, `Data_Processing`, etc.
- Each plugin must subclass `BasePlugin` and implement `execute_pipeline_step(self, step_config, context)`.
- Example skeleton:
```python
from Plugins.BasePlugin import BasePlugin

class MyPlugin(BasePlugin):
    plugin_type = "Data_Processing"
    def execute_pipeline_step(self, step_config, context):
        # Your logic here
        return {step_config["output"]: result}
```
- Register your plugin by placing it in the correct directory and naming it appropriately.
- Plugins are auto-discovered by the PluginManager.

---

## How to Use Test Mode

- Enable test mode in `src/config.yaml`:
```yaml
settings:
  test_mode: true
```
- In test mode, LLMFunction and similar plugins will use `mock_response` from the pipeline YAML and avoid real API calls.
- All steps requiring external APIs should have explicit `mock_response` fields for predictable tests.
- Output files are automatically cleaned up at the start of each test-mode run.

---

## Common Questions & Answers (FAQ)

**Q: How do I add a new step to a pipeline?**
A: Edit the pipeline YAML and append a new step with the required plugin and config.

**Q: Why is my API plugin failing?**
A: Check that the required API key is set in your environment.

**Q: How do I debug pipeline errors?**
A: Check the logs in `src/mimir.log` for detailed error messages and context.

**Q: Where are reports and outputs saved?**
A: By default, in the `src/reports/` and `src/output/` directories. Paths can be configured in YAML or `config.yaml`.

**Q: How do I ensure clean test runs?**
A: Use test mode and let the framework handle output cleanup automatically.

---

## Troubleshooting & Best Practices

- Use descriptive step names and outputs in pipeline YAML.
- Always provide `mock_response` for LLM/API steps in test mode.
- Review plugin docstrings for config options and expected formats.
- Use absolute paths for output files if you want to control their location precisely.
- Check logs for `[ERROR]` and `[CLEANUP]` messages.

---

## Directory Structure Reference

- `src/Plugins/` - All plugin code, organized by type.
- `src/pipelines/` - Pipeline YAMLs.
- `src/config.yaml` - Main config and pipeline references.
- `src/reports/` - Generated HTML reports.
- `src/output/` - Other outputs and intermediate files.
- `tests/` - Automated tests.

---

## Contribution Guide (Optional)
- Fork the repo and create a branch for your feature.
- Follow the existing code style and naming conventions.
- Add clear docstrings and comments for new plugins or features.
- Add or update tests in `tests/`.
- Submit a pull request with a detailed description.

---

## Design Decisions & Gotchas
- Plugins are auto-discovered and must follow naming conventions.
- Environment variables are preferred for secrets/API keys (never hardcode them).
- Test mode is strict: all LLM/API steps must have explicit mocks.
- Output directories may be relative to `src/`—use absolute paths if needed.
- Logging is your friend: check `mimir.log` for everything!

---

For further details, see plugin docstrings and example YAML files in `src/pipelines/`.

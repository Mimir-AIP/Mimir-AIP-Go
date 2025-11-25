# Examples

This directory contains examples demonstrating how to use Mimir AIP plugins and pipelines.

## LLM Integration Examples

### 1. OpenAI Plugin (`ai_openai_plugin.go`)
- Implements OpenAI API integration for chat/completion.
- Supports chat and completion operations.
- Usage in YAML pipelines shown in the file comments.

### 2. LLM Integration Example (`llm_integration_example.go`)
- Demonstrates direct usage of an LLM plugin in Go code.
- Shows:
  - Simple chat completion
  - Structured prompt/response
  - Error handling (invalid API key)
- Run with:
  ```bash
  export OPENAI_API_KEY=your-key
  go run examples/llm_integration_example.go
  ```

### 3. Agentic Workflow YAML (`test_pipelines/agentic_workflow_example.yaml`)
- Multi-step pipeline: LLM → transform → output.
- Demonstrates chaining an LLM call with data processing.
- Run with:
  ```bash
  export OPENAI_API_KEY=your-key
  mimir-aip run test_pipelines/agentic_workflow_example.yaml
  ```

### 4. LLM Chain YAML (`test_pipelines/llm_chain_example.yaml`)
- Multi-step LLM chain: generate → summarize → refine → output.
- Shows how to pass outputs between LLM calls.
- Run with:
  ```bash
  export OPENAI_API_KEY=your-key
  mimir-aip run test_pipelines/llm_chain_example.yaml
  ```

## Other Plugin Examples

- `data_model_example_plugin.go`: Shows how to use various data types (JSON, binary, time series, image).
- `data_processing_transformer_plugin.go`: Demonstrates data transformation operations.
- `input_rss_plugin.go`: Example input plugin for fetching RSS feeds.
- `output_json_plugin.go`: Example output plugin for writing JSON files.

## Running Examples

1. Set required environment variables (e.g., `OPENAI_API_KEY`).
2. Use the Mimir AIP CLI to run YAML pipelines:
   ```bash
   mimir-aip run <pipeline-file.yaml>
   ```
3. For Go examples, run directly with `go run`.

## Tips

- Use environment variables for API keys (avoid hardcoding).
- Check output files in `./output/` directory after running pipelines.
- Refer to the comments in each example for configuration options.
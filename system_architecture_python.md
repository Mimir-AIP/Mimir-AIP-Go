# Mimir-AIP-Go System Architecture (Python Version)

This document provides a high-level overview of the current Python-based Mimir-AIP system, illustrating how its components interact, how pipeline YAMLs are consumed and executed, and the flow of data from input to output. This will serve as a reference for replicating the system in Go and planning future enhancements.

---


## System Architecture Diagram

```mermaid
flowchart TD
  %% Main Application
  subgraph Main[Main Application]
    main_py[main.py]
    scheduler[PipelineScheduler.py]
    visualizer[PipelineVisualizer/AsciiTree.py]
    plugin_manager[Plugins/PluginManager.py]
  end
  %% Plugins
  subgraph Plugins[Plugins]
    base_plugin[BasePlugin.py]
    ai_models[AIModels]
    data_processing[Data_Processing]
    input_plugins[Input]
    output_plugins[Output]
  end
  %% Pipelines
  subgraph Pipelines[Pipelines]
    pipeline_yaml[pipeline.yaml, CustomReportPipeline.yaml]
  end
  %% Pipeline Execution
  subgraph Execution[Pipeline Execution]
    exec_pipeline[execute_pipeline(pipeline, plugin_manager, output_dir, test_mode)]
    exec_step[execute_step(step, context, plugin_manager)]
  end
  %% Context
  subgraph Context[Context]
    context_dict[context: dict]
  end

  pipeline_yaml -- "Defines steps/config" --> main_py
  main_py -- "Loads" --> plugin_manager
  plugin_manager -- "Discovers/Loads" --> Plugins
  Plugins -- "Implements" --> base_plugin
  main_py -- "Schedules" --> scheduler
  main_py -- "Visualizes" --> visualizer
  main_py -- "Calls" --> exec_pipeline
  exec_pipeline -- "Iterates steps" --> exec_step
  exec_step -- "Looks up" --> plugin_manager
  exec_step -- "Calls" --> Plugins
  Plugins -- "Updates" --> context_dict
  exec_step -- "Updates" --> context_dict
  exec_pipeline -- "Updates" --> context_dict
  context_dict -- "Passed to" --> Plugins
  context_dict -- "Used by" --> exec_step
  exec_pipeline -- "Outputs" --> OutputPlugins
  OutputPlugins -- "Writes" --> Filesystem_Reports[Filesystem/Reports]
```

---

## Core Files, Classes, and Methods

### main.py
- **main()**: Entry point. Loads `config.yaml`, initializes `PluginManager`, loads pipelines, and executes them.
- **execute_pipeline(pipeline, plugin_manager, output_dir, test_mode=False)**: Runs a pipeline, iterates steps, manages context, visualizes status.
- **execute_step(step, context, plugin_manager)**: Looks up plugin, calls `execute_pipeline_step`, merges results into context.
- **run_scheduled_pipelines(config, plugin_manager, output_dir)**: Handles scheduled pipelines using cron expressions.

### Plugins/PluginManager.py
- **PluginManager**: Discovers and loads plugins by type (AIModels, Data_Processing, Input, Output).
  - `get_plugins(plugin_type=None) -> dict`
  - `get_plugin(plugin_type, name) -> object`
  - `get_all_plugins() -> dict`

### Plugins/BasePlugin.py
- **BasePlugin**: Abstract base class for all plugins.
  - `execute_pipeline_step(step_config: dict, context: dict) -> dict`

### Plugins/AIModels/BaseAIModel/BaseAIModel.py
- **BaseAIModel**: Abstract base for AI model plugins.
  - `chat_completion(model: str, messages: list) -> str`
  - `text_completion(model: str, prompt: str) -> str`
  - `get_available_models() -> list`

### Example Plugin: Plugins/Input/api/api.py
- **ApiPlugin**: Makes HTTP requests.
  - `execute_pipeline_step(step_config, context) -> dict`
    - `step_config['config']` expects: `{url, method, headers, params, data, timeout}`
    - Returns: `{output_key: response_dict}`
  - `make_request(url, method, headers, params, data, timeout) -> dict`

### Pipeline YAML Format
- Each pipeline YAML defines:
  - `name`: Pipeline name
  - `enabled`: true/false
  - `steps`: List of steps
    - `name`: Step name
    - `plugin`: Plugin reference (e.g., `Input.api`)
    - `config`: Dict of parameters for the plugin
    - `output`: (optional) Key to store result in context

#### Example Step
```yaml
- name: FetchAircraftData
  plugin: Input.ADSBdata
  config:
    lat: 54.6079
    lon: -5.9264
    radius: 25
    limit: 10
  output: aircraft_data
```

---

## Data Flow & Parameter Passing
- **Context**: A Python dict passed to every plugin and step, accumulating results.
- **Step Config**: Each step receives a `step_config` dict (from YAML) and the current `context` dict.
- **Plugin Output**: Plugins return a dict of new/updated context variables.
- **Pipeline Execution**: Steps can reference previous outputs via context keys.

---

## Next Steps
- Use this as a reference for the Go rewrite.
- The next section will propose changes for agentic workflows and Go-specific improvements.
---

## Component Descriptions

- **Input Plugins**: Modules that ingest data from various sources (APIs, RSS feeds, web scraping, images, etc.).
- **Data_Processing Plugins**: Transform, aggregate, and enrich input data. Each plugin performs a specific function (formatting, logging, bounding boxes, etc.).
- **AI Models**: LLMs and other AI models that can be called as part of the pipeline for advanced processing or reasoning.
- **Output Plugins**: Generate final outputs (HTML reports, maps, etc.) from processed data.
- **Storage**: Outputs are saved to the file system; future plans may include database storage.
- **Pipeline YAML**: Defines the sequence of plugins and models to execute, their configuration, and data flow.

---

## Pipeline Execution Flow

1. **Pipeline YAML Parsing**: The system reads a pipeline YAML file (e.g., `pipeline.yaml`), which specifies the sequence of plugins and models to use, their configuration, and data flow.
2. **Input Stage**: Input plugins ingest data as defined in the pipeline YAML.
3. **Processing Stage**: Data is passed through a series of data processing plugins, each performing a transformation or enrichment.
4. **AI/LLM Stage**: Data is optionally passed to AI models for further processing, reasoning, or generation.
5. **Output Stage**: Output plugins generate final artifacts (reports, maps, etc.) and save them to storage.
6. **Storage**: Outputs are saved to the file system (or database in the future).

---

## Existing Functionality

- Modular plugin architecture for easy extension.
- YAML-based pipeline definition for flexible orchestration.
- Support for multiple data sources and output formats.
- Integration with LLMs and AI models.
- Pluggable data processing steps.
- Output to HTML reports, maps, and more.

---

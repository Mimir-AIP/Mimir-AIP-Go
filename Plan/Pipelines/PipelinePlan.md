# Pipelines

Pipelines will be used for data ingestion, processing and output generation. There will be 3 types of pipeline; ingestion, processing and output(a pipeline can only be defined as one type). Each pipeline will be defined in a YAML format, specifying the steps. Throughout the life of a pipeline there will be a context window(max size to be determined [TODO]) which will store a list of objects(each step gets an object where data can be stored and accessed by ANY subsequent step in pipeline) context is dynamic so any output from a step will be stored in that step's context object. Steps can perform any action from either the builtin default plugin, OR a custom plugin installed by the user. The default plugin includes a variety of common functions such as; managing the context(CRUD operations on the entire context window or specific objects), making http requests, parsing json data, this will also include some basic conditional logic such as if/else statements, GOTO actions to allow for loops. Custom plugins will follow a standardised format.

Types of pipelines:
- Ingestion pipelines will be used to ingest data into the system, this could be from APIs, databases, file uploads etc. When a pipeline tagged as 'ingestion' is created, mimir will automatically run the pipeline and process the ingested data to extract entities, attributes and relationships which will then be used to automatically generate an ontology for the project.
- Processing pipelines will be used to process data that has already been ingested and stored in the system, this could be for cleaning the data, transforming it, generating new features for ML models etc.
- Output pipelines will be used to generate outputs, this could be for generating reports, sending notifications, exporting data to other systems etc. Output pipelines can be triggered manually by the user or automatically via actions setup within the digital twin, regularly on a schedule or triggered by another pipeline.

Pipeline Execution:
Pipelines can be triggered in 3 ways; manually by the user, automated(via digital twin actions or ontology entity extraction), or regularly on a schedule(jobs)

Jobs:
Jobs are pipelines that are set to run on a regular schedule(uses a cron format for scheduling), when a job is created, user specifies which pipeline(s) to run and the schedule, mimir will then automatically trigger the specified pipeline(s) according to the schedule. 

Pipeline YAML Schema:

- **name**: string (required) - The unique name of the pipeline.
- **type**: string (required) - The type of pipeline. Must be one of: `ingestion`, `processing`, `output`.
- **steps**: array (required) - A list of steps to execute in order.
  - **name**: string (required) - A unique name for the step within the pipeline.
  - **plugin**: string (required) - The name of the plugin to use (e.g., `builtin` for default plugin).
  - **parameters**: object (optional) - A key-value map of parameters to pass to the action.
  - **action**: string (required) - The specific action or method to call on the plugin.
  - **output**: object (optional) - A key-value map of outputs to store in the step's context object. Values can reference context data using templating (e.g., `{{context.step_name.key}}`).

## Pipeline Example

This example demonstrates an ingestion pipeline using only actions from the default plugin. It fetches data from an API, parses the JSON response, and stores the results in context for automatic ontology generation.

```yaml
name: api_data_ingestion
type: ingestion
steps:
  - name: fetch_data
    plugin: default
    parameters:
      url: https://jsonplaceholder.typicode.com/posts/1
      method: GET
    action: http_request
    output:
      response_data: "{{response.body}}"
  - name: parse_response
    plugin: default
    parameters:
      data: "{{context.fetch_data.response_data}}"
    action: parse_json
    output:
      parsed_json: "{{parsed}}"
  - name: check_success
    plugin: default
    parameters:
      condition: "{{context.parse_response.parsed_json.id}}"
      if_true: "store_success"
      if_false: "store_failure"
    action: if_else
    output:
      status: "{{result}}"
``` 
from Plugins.BasePlugin import BasePlugin

class PipelineControl(BasePlugin):
    """
    PipelineControl plugin for pipeline control operations (e.g., goto).
    Supports jumping to a named step in the pipeline.
    """
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Execute a pipeline control operation.
        Supported operations:
            - goto: Jump to a named step in the pipeline.
        Args:
            step_config (dict): Step configuration from pipeline YAML.
            context (dict): Pipeline context.
        Returns:
            dict: Special key '__goto__' if goto is used.
        Raises:
            ValueError: If required config is missing.
            NotImplementedError: If operation is not supported.
        """
        operation = step_config.get("operation")
        config = step_config.get("config", {})
        if operation == "goto":
            step = config.get("step")
            if not step or not isinstance(step, str):
                raise ValueError("'step' (str) must be specified for goto operation in PipelineControl plugin.")
            # Return a special key to signal the runner to jump
            return {"__goto__": step}
        else:
            raise NotImplementedError(f"Operation '{operation}' not supported by PipelineControl plugin.")

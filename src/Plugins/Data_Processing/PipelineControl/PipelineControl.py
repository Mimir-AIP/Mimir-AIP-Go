import time
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
            - halt: Halt the pipeline indefinitely.
        Args:
            step_config (dict): Step configuration from pipeline YAML.
            context (dict): Pipeline context.
        Returns:
            dict: Special key '__goto__' if goto is used.
        Raises:
            ValueError: If required config is missing.
            NotImplementedError: If operation is not supported.
        """
        config = step_config.get("config", {})
        operation = config.get("operation")
        if operation == "goto":
            step = config.get("step")
            if not step or not isinstance(step, str):
                raise ValueError("'step' (str) must be specified for goto operation in PipelineControl plugin.")
            # Return a special key to signal the runner to jump
            return {"__goto__": step}
        elif operation == "halt":
            print("PipelineControl 'halt' operation activated. Pipeline will now wait indefinitely.")
            try:
                while True:
                    time.sleep(1) # Sleep to prevent high CPU usage
            except KeyboardInterrupt:
                print("PipelineControl 'halt' operation interrupted. Exiting.")
            except Exception as e:
                print(f"PipelineControl 'halt' operation encountered an error: {e}")
            return context # Return context to allow pipeline to continue if interrupted
        else:
            raise NotImplementedError(f"Operation '{operation}' not supported by PipelineControl plugin.")

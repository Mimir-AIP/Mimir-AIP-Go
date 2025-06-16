"""
BasePlugin module.

Defines the abstract BasePlugin class that all plugins must extend. Provides execute_pipeline_step interface.
"""
from abc import ABC, abstractmethod
from typing import Dict, Any, Optional

class BasePlugin(ABC):
    """
    Base class for all plugins.

    Plugins extending this class can declare JSON schemas for their expected
    input and output context data using the `_input_context_schema` and
    `_output_context_schema` class attributes. These schemas will be used
    by the ContextService for automatic validation.
    """

    # Optional: Define a JSON schema for the expected input context for this plugin.
    # This schema will be used to validate the 'context' dictionary passed to execute_pipeline_step.
    _input_context_schema: Optional[Dict[str, Any]] = None

    # Optional: Define a JSON schema for the expected output context from this plugin.
    # This schema will be used to validate the dictionary returned by execute_pipeline_step.
    _output_context_schema: Optional[Dict[str, Any]] = None
    
    @abstractmethod
    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for this plugin
        
        Args:
            step_config (dict): Configuration for this step from the pipeline YAML
            context (dict): Current pipeline context with variables
            
        Returns:
            dict: Updated context with any new variables
        """
        pass
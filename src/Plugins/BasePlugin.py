from abc import ABC, abstractmethod

class BasePlugin(ABC):
    """Base class for all plugins"""
    
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

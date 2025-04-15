"""
Plugin for using LLM models as functions

Example usage:
    plugin = LLMFunction()
    plugin.set_llm_plugin("MockAIModel")
    result = plugin.execute_pipeline_step({
        "config": {
            "plugin": "MockAIModel",
            "model": "mock-model-1",
            "function": "Summarize the following text",
            "format": "Output should be a one-sentence summary"
        },
        "input": "text_to_process",
        "output": "processed_result"
    }, {"text_to_process": "Some text to process"})
"""

import logging

from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel
from Plugins.BasePlugin import BasePlugin
from Plugins.PluginManager import PluginManager

class LLMFunction(BasePlugin):
    """
    Plugin for using LLM models as functions
    """

    plugin_type = "Data_Processing"

    def __init__(self, llm_plugin=None, plugin_manager=None, logger=None):
        """
        Initialize the LLMFunction plugin
        
        Args:
            llm_plugin (BaseAIModel, optional): LLM plugin instance to use. Defaults to None.
            plugin_manager (PluginManager, optional): PluginManager instance. Defaults to real PluginManager.
            logger (logging.Logger, optional): Logger instance. Defaults to real logger.
        """
        self.llm_plugin = llm_plugin
        self.plugin_manager = plugin_manager if plugin_manager is not None else PluginManager()
        self.logger = logger if logger is not None else logging.getLogger(__name__)

    def set_llm_plugin(self, plugin_name):
        """
        Set the LLM plugin to use
        
        Args:
            plugin_name (str): Name of the LLM plugin to use
        """
        self.llm_plugin = self.plugin_manager.get_plugin("AIModel", plugin_name)
        if not self.llm_plugin:
            raise ValueError(f"LLM plugin {plugin_name} not found")
        self.logger.debug(f"Loaded LLM plugin: {plugin_name}")

    def execute_pipeline_step(self, step_config, context):
        """
        Execute a pipeline step using the configured LLM plugin
        
        Args:
            step_config (dict): Step configuration
            context (dict): Pipeline context
            
        Returns:
            dict: Updated context with step output
        """
        if not self.llm_plugin:
            raise ValueError("LLM plugin not set. Please call set_llm_plugin() first.")
            
        config = step_config["config"]
        input_data = eval(step_config["input"], context)
        
        # Format the input data based on the function type
        if "function" in config and config["function"]:
            messages = [
                {
                    "role": "user",
                    "content": f"{config['function']}\n\n{input_data}"
                }
            ]
        else:
            messages = [
                {
                    "role": "user",
                    "content": input_data
                }
            ]
        
        self.logger.debug(f"Request to LLM plugin: {messages}")
        
        # Get the response from the LLM plugin
        response = self.llm_plugin.chat_completion(
            model=config["model"],
            messages=messages
        )
        
        self.logger.debug(f"Response from LLM plugin: {response}")
        
        # Format the response based on the format string
        if "format" in config and config["format"]:
            try:
                formatted_response = eval(config["format"], {"response": response})
            except Exception as e:
                raise ValueError(f"Error formatting response: {e}")
        else:
            formatted_response = response
        
        return {step_config["output"]: formatted_response}


if __name__ == "__main__":
    # Example usage
    from Plugins.AIModels.MockAIModel.MockAIModel import MockAIModel
    
    # Create and configure the plugin
    plugin = LLMFunction()
    mock_plugin = MockAIModel()
    plugin.set_llm_plugin(mock_plugin)
    
    # Test with a simple pipeline step
    step_config = {
        "config": {
            "plugin": "MockAIModel",
            "model": "mock-model-1",
            "function": "Summarize the following text",
            "format": "response[:50]"
        },
        "input": "text_to_process",
        "output": "processed_result"
    }
    
    # Test with some input data
    context = {"text_to_process": "This is a test text to process. It should be summarized by the plugin."}
    result = plugin.execute_pipeline_step(step_config, context)
    print(f"Result: {result}")
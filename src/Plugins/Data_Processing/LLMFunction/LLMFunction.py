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
import ast

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
            plugin_manager (PluginManager, optional): PluginManager instance. If None, must be set explicitly before use.
            logger (logging.Logger, optional): Logger instance. Defaults to real logger.
        """
        self.llm_plugin = llm_plugin
        self.plugin_manager = plugin_manager  # Do NOT instantiate PluginManager by default to avoid recursion
        self.logger = logger if logger is not None else logging.getLogger(__name__)

    def set_llm_plugin(self, plugin_name):
        """
        Set the LLM plugin to use
        
        Args:
            plugin_name (str): Name of the LLM plugin to use
        """
        if not self.plugin_manager:
            raise RuntimeError("LLMFunction: plugin_manager must be set before calling set_llm_plugin(). Avoid recursive PluginManager instantiation.")
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
        try:
            logger = logging.getLogger(__name__)
            logger.info("[LLMFunction DEBUG] Entered execute_pipeline_step")
            # Check for test mode
            test_mode = False
            if "test_mode" in context:
                test_mode = context["test_mode"]
            elif "test_mode" in step_config:
                test_mode = step_config["test_mode"]
            if test_mode:
                # Prefer mock_response in step_config, then in step_config['config'] if present
                mock_response = step_config.get("mock_response")
                if not mock_response and "config" in step_config:
                    mock_response = step_config["config"].get("mock_response")
                logger.info(f"[LLMFunction] Test mode active. Using mock_response: {mock_response}")
                result = mock_response if mock_response is not None else "No headline generated."
            else:
                if not self.llm_plugin:
                    logger.error("LLM plugin not set. Please call set_llm_plugin() first.")
                    result = "No headline generated."
                else:
                    config = step_config["config"]
                    try:
                        input_data = eval(step_config["input"], context)
                        if "function" in config and config["function"]:
                            messages = [
                                {
                                    "role": "user",
                                    "content": f"{config['function']}\n\n{input_data}"
                                }
                            ]
                            response = self.llm_plugin.chat_completion(
                                model=config.get("model", ""),
                                messages=messages
                            )
                            if isinstance(response, dict) and 'content' in response:
                                result = response['content']
                            else:
                                result = response if response else "No headline generated."
                        else:
                            result = "No headline generated."
                    except Exception as e:
                        logger.error(f"LLMFunction: Error during LLM call: {e}")
                        result = "No headline generated."
            logger.info(f"[LLMFunction] Output: {result}")
            output_key = step_config["output"]
            context[output_key] = result
            logger.info(f"[LLMFunction] Step output_key: {output_key}, result: {result}")
            logger.info(f"[LLMFunction] Context after step: {context}")
            return {output_key: result}
        except Exception as top_level_e:
            logger.error(f"LLMFunction: Top-level error: {top_level_e}")
            output_key = step_config.get("output", "llm_result")
            context[output_key] = "No headline generated."
            return {output_key: "No headline generated."}

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
    logger = logging.getLogger(__name__)
    logger.info(f"Result: {result}")
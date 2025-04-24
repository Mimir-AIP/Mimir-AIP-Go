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
            result = mock_response
        else:
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
            
            logger.debug(f"Request to LLM plugin: {messages}")
            
            # Get the response from the LLM plugin
            # Try to get the full response if possible, fallback to string
            try:
                response = self.llm_plugin.chat_completion(
                    model=config["model"],
                    messages=messages,
                    return_full_response=True
                )
                logger.debug(f"[DEBUG] Full LLM response: {response}")
                # If 'choices' in response, extract content
                if isinstance(response, dict) and "choices" in response:
                    llm_content = response["choices"][0]["message"]["content"]
                else:
                    llm_content = response
            except TypeError:
                # Backward compatibility: plugin does not support return_full_response
                response = self.llm_plugin.chat_completion(
                    model=config["model"],
                    messages=messages
                )
                llm_content = response
            logger.debug(f"Response from LLM plugin: {llm_content}")
            logger.info(f"LLMFunction: Raw response from LLM: {llm_content}")

            # Format the response based on the format string
            if "format" in config and config["format"]:
                try:
                    # Try to parse the response as a Python literal if possible
                    parsed_response = None
                    try:
                        parsed_response = ast.literal_eval(llm_content)
                        logger.debug(f"Parsed LLM response as literal: {parsed_response}")
                    except Exception:
                        parsed_response = llm_content
                    formatted_response = eval(config["format"], {"response": parsed_response})
                except Exception as e:
                    logger.error(f"LLMFunction: Error formatting response: {e}\nRaw response: {llm_content}")
                    raise ValueError(f"Error formatting response: {e}\nRaw response: {llm_content}")
            else:
                formatted_response = llm_content
            
            result = formatted_response
        
        output_key = step_config["output"]
        context[output_key] = result
        logger.info(f"[LLMFunction] Step output_key: {output_key}, result: {result}")
        logger.info(f"[LLMFunction] Context after step: {context}")
        return {output_key: result}


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
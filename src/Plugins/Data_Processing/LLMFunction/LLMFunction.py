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
        self.llm_plugin = self.plugin_manager.get_plugin("AIModels", plugin_name)
        if not self.llm_plugin:
            # Try case-insensitive match
            for name, plugin in self.plugin_manager.get_plugins("AIModels").items():
                if name.lower() == plugin_name.lower():
                    self.llm_plugin = plugin
                    break
        if not self.llm_plugin:
            raise ValueError(f"LLM plugin {plugin_name} not found")
        self.logger.info(f"Loaded LLM plugin: {plugin_name}")

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
            logger.info(f"[LLMFunction DEBUG] step_config: {step_config}")
            logger.info(f"[LLMFunction DEBUG] context (keys): {list(context.keys())}")
            logger.info(f"[LLMFunction DEBUG] context (headline_text): {context.get('headline_text', 'MISSING')}")
            test_mode = False
            if "test_mode" in context:
                test_mode = context["test_mode"]
            elif "test_mode" in step_config:
                test_mode = step_config["test_mode"]
            if test_mode:
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
                    # Dynamically select LLM plugin based on config
                    plugin_name = config.get("plugin")
                    if plugin_name and self.plugin_manager:
                        plugin_candidate = self.plugin_manager.get_plugin("AIModels", plugin_name)
                        if not plugin_candidate:
                            # Fallback to case-insensitive match
                            for name, inst in self.plugin_manager.get_plugins("AIModels").items():
                                if name.lower() == plugin_name.lower():
                                    plugin_candidate = inst
                                    break
                        if plugin_candidate:
                            self.llm_plugin = plugin_candidate
                        else:
                            logger.error(f"LLMFunction: LLM plugin '{plugin_name}' not found")
                            raise ValueError(f"LLM plugin '{plugin_name}' not found")
                    try:
                        input_expr = step_config.get("input", None)
                        if not input_expr:
                            logger.error("[LLMFunction] No 'input' key specified in step_config.")
                            raise ValueError("No 'input' key specified in step_config.")
                        try:
                            input_data = eval(input_expr, context)
                        except Exception as eval_exc:
                            logger.error(f"[LLMFunction] Could not evaluate input expression '{input_expr}': {eval_exc}")
                            raise
                        if input_data is None:
                            logger.error(f"[LLMFunction] Input data for '{input_expr}' is None or missing in context.")
                            raise ValueError(f"Input data for '{input_expr}' is None or missing in context.")
                        # Construct the prompt with function and format
                        function = config.get('function', '')
                        format = config.get('format', '')
                        prompt = f"{function}\n\n{format}\n\n{input_data}"
                        messages = [
                            {
                                "role": "user",
                                "content": prompt
                            }
                        ]
                        logger.info(f"[LLMFunction DEBUG] Sending messages to mock model: {messages}")
                        response = self.llm_plugin.chat_completion(
                            model=config.get("model", ""),
                            messages=messages
                        )
                        logger.info(f"[LLMFunction DEBUG] Raw response from llm_plugin: {response!r} (type: {type(response)})")
                        if isinstance(response, dict) and 'content' in response:
                            result = response['content']
                        else:
                            result = response if response else "No headline generated."
                        logger.info(f"[LLMFunction DEBUG] Final result assigned: {result!r}")
                    except Exception as e:
                        logger.error(f"LLMFunction: Error during LLM call: {e}")
                        result = f"No headline generated. Error: {e}"
            logger.info(f"[LLMFunction] Output: {result}")
            output_key = step_config["output"]
            context[output_key] = result
            logger.info(f"[LLMFunction] Step output_key: {output_key}, result: {result}")
            logger.info(f"[LLMFunction] Context after step: {context}")
            return {output_key: result}
        except Exception as top_level_e:
            logger.error(f"LLMFunction: Top-level error: {top_level_e}")
            output_key = step_config.get("output", "llm_result")
            context[output_key] = f"No headline generated. Top-level error: {top_level_e}"
            return {output_key: f"No headline generated. Top-level error: {top_level_e}"}

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
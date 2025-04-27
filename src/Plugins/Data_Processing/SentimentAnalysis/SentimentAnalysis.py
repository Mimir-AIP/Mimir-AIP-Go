"""
SentimentAnalysis module.

Performs a simple sentiment analysis on text input, returning a neutral summary or fallback message.
"""
import logging
from Plugins.BasePlugin import BasePlugin

class SentimentAnalysis(BasePlugin):
    """Plugin to perform sentiment analysis on text input, returning a neutral summary or fallback message."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Perform sentiment analysis on text input and update context.

        Args:
            step_config (dict): Step configuration with 'config' dict containing 'input' (str or context key) and optional 'output' (str).
            context (dict): Current pipeline context.

        Returns:
            dict: Mapping of output_key to sentiment result string.
        """
        # Get text input from context
        config = step_config.get("config", {})
        input_key = config.get("input")
        text = None
        if isinstance(input_key, str):
            text = context.get(input_key)
        else:
            text = input_key

        # Naive sentiment logic
        if not text:
            sentiment = "No text provided for sentiment analysis."
        else:
            sentiment = "Neutral sentiment detected."

        output_key = step_config.get("output", "sentiment")
        logging.getLogger(__name__).info(f"[SentimentAnalysis] Input: {text}, Output: {sentiment}")
        return {output_key: sentiment}
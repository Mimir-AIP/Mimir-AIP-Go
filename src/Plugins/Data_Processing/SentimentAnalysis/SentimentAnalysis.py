import logging
from Plugins.BasePlugin import BasePlugin

class SentimentAnalysis(BasePlugin):
    """
    Plugin for performing a simple sentiment analysis on a text input.
    Returns a neutral sentiment summary by default.
    """
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
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

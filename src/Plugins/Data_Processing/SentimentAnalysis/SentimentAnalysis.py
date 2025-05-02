"""
SentimentAnalysis module.

Performs a simple sentiment analysis on text input, returning a neutral summary or fallback message.
"""
import logging
from transformers import pipeline
from Plugins.BasePlugin import BasePlugin

class SentimentAnalysis(BasePlugin):
    """Plugin to perform sentiment analysis on text input, returning a neutral summary or fallback message."""
    plugin_type = "Data_Processing"

    def __init__(self):
        super().__init__()
        self.logger = logging.getLogger(__name__)
        self._analyzer_cache = {}

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
        # Determine source of text data
        data_key = config.get("data_key", config.get("input"))
        text = None
        if isinstance(data_key, str):
            text = context.get(data_key)
        else:
            text = data_key

        # Sentiment analysis via HuggingFace pipeline
        if not text:
            self.logger.warning("[SentimentAnalysis] No text provided.")
            sentiment = "No text provided for sentiment analysis."
        else:
            model_name = config.get("model_name", "distilbert-base-uncased-finetuned-sst-2-english")
            top_k = config.get("top_k", 1)
            if model_name not in self._analyzer_cache:
                self.logger.info(f"[SentimentAnalysis] Loading model '{model_name}'")
                self._analyzer_cache[model_name] = pipeline("sentiment-analysis", model=model_name)
            analyzer = self._analyzer_cache[model_name]
            # Normalize to list
            single = isinstance(text, str)
            texts = [text] if single else list(text)
            results = analyzer(texts, top_k=top_k)
            sentiment = results[0] if single else results

        output_key = step_config.get("output", "sentiment")
        self.logger.info(f"[SentimentAnalysis] Input: {text}, Output: {sentiment}")
        return {output_key: sentiment}
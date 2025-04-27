"""
SentimentAnalysis plugin module.

Performs sentiment analysis on text using HuggingFace pipeline.

Example usage:
    plugin = SentimentAnalysis()
    result = plugin.execute_pipeline_step({
        "config": {
            "data_key": "text",
            "model_name": "distilbert-base-uncased-finetuned-sst-2-english",
            "top_k": 1
        },
        "output": "sentiment_result"
    }, context)
"""
from .SentimentAnalysis import SentimentAnalysis

__all__ = ['SentimentAnalysis']

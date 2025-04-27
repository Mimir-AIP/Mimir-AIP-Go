"""
Plugin for translating text between languages using HuggingFace models.

Example usage:
    plugin = TextTranslator()
    result = plugin.execute_pipeline_step({
        "config": {
            "data_key": "sentences",         # Context key with text or list[str]
            "output_key": "translated",      # Where to store translations
            "source_lang": "en",             # ISO code or 'auto'
            "target_lang": "de",             # ISO code
            "provider_config": {
                "model_name": "Helsinki-NLP/opus-mt-en-de",  # HF model
                "quantize": True                              # Optional dynamic quantization
            },
            "batch_size": 8                    # Optional batching
        },
        "output": "translated"
    }, context)
"""
from .TextTranslator import TextTranslator

__all__ = ['TextTranslator']

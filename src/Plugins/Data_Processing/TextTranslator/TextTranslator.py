"""
Plugin for translating text between languages using HuggingFace MarianMT models.

Config (in step_config['config']):
  data_key: str           # context key holding text or list[str]
  output_key: str         # where to store translations
  source_lang: str        # ISO code or 'auto'
  target_lang: str        # ISO code
  provider_config: {
    model_name: str       # HF model, e.g. Helsinki-NLP/opus-mt-en-de
    quantize: bool        # optional dynamic quantization
  }
  batch_size: int         # optional, default=8

Example pipeline step:
- plugin: TextTranslator
  config:
    data_key: sentences
    output_key: translated
    source_lang: en
    target_lang: de
    provider_config:
      model_name: Helsinki-NLP/opus-mt-en-de
      quantize: True
    batch_size: 8
  output: translated
"""
import logging
from Plugins.BasePlugin import BasePlugin
from transformers import AutoTokenizer, AutoModelForSeq2SeqLM
import torch


class TextTranslator(BasePlugin):
    """
    Translates text using HuggingFace MarianMT models, with optional quantization.
    """
    plugin_type = "Data_Processing"

    def __init__(self):
        super().__init__()
        self.logger = logging.getLogger(__name__)
        self._model_cache = {}

    def execute_pipeline_step(self, step_config, context):
        config = step_config.get('config', {})
        data_key = config.get('data_key')
        output_key = config.get('output_key', step_config.get('output'))
        if not data_key:
            self.logger.error("[TextTranslator] 'data_key' is required in config.")
            raise ValueError("TextTranslator requires 'data_key'.")

        data = context.get(data_key)
        if data is None:
            self.logger.error(f"[TextTranslator] Context key '{data_key}' not found.")
            raise KeyError(f"Context key '{data_key}' not found.")

        # Normalize to list
        single_input = False
        if isinstance(data, str):
            texts = [data]
            single_input = True
        elif isinstance(data, (list, tuple)):
            texts = list(data)
        else:
            self.logger.error(f"[TextTranslator] Unsupported data type: {type(data)}")
            raise TypeError("TextTranslator data must be a string or list of strings.")

        # Load or reuse model/tokenizer
        provider = config.get('provider_config', {})
        model_name = provider.get('model_name')
        if not model_name:
            src = config.get('source_lang', 'en')
            tgt = config.get('target_lang', 'de')
            model_name = f"Helsinki-NLP/opus-mt-{src}-{tgt}"

        quantize = provider.get('quantize', False)
        batch_size = config.get('batch_size', 8)

        if model_name not in self._model_cache:
            self.logger.info(f"[TextTranslator] Loading model '{model_name}'...")
            tokenizer = AutoTokenizer.from_pretrained(model_name)
            model = AutoModelForSeq2SeqLM.from_pretrained(model_name)
            if quantize:
                self.logger.info(f"[TextTranslator] Applying dynamic quantization.")
                model = torch.quantization.quantize_dynamic(
                    model, {torch.nn.Linear}, dtype=torch.qint8
                )
            self._model_cache[model_name] = (tokenizer, model)
        else:
            tokenizer, model = self._model_cache[model_name]

        device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
        model.to(device)

        # Translate in batches
        translations = []
        for i in range(0, len(texts), batch_size):
            batch = texts[i:i+batch_size]
            inputs = tokenizer(batch, return_tensors='pt', padding=True, truncation=True).to(device)
            with torch.no_grad():
                outputs = model.generate(**inputs)
            decoded = tokenizer.batch_decode(outputs, skip_special_tokens=True)
            translations.extend(decoded)

        result = translations[0] if single_input else translations
        return {output_key: result}

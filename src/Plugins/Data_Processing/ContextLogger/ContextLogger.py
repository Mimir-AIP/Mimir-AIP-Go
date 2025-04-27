"""
ContextLogger module.

Diagnostic plugin that logs context keys and optionally values after each pipeline step.
"""
from Plugins.BasePlugin import BasePlugin
import logging
import json

class ContextLogger(BasePlugin):
    """Logs the current context keys and optionally values for debugging."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """Execute the context logging step by reporting context keys and optionally values.

        Args:
            step_config (dict): Step configuration containing 'config' with optional 'log_values'.
            context (dict): Current pipeline context to log.

        Returns:
            dict: Empty dict (no context modifications).
        """
        config = step_config.get('config', {})
        log_values = config.get('log_values', False)
        logger = logging.getLogger(__name__)
        logger.info(f"[ContextLogger] Context keys: {list(context.keys())}")
        # Also log as ERROR for guaranteed visibility
        logger.error(f"[ContextLogger] [ERROR LEVEL] Context keys: {list(context.keys())}")
        if log_values:
            # Log a JSON dump of the context (truncate long values)
            safe_context = {k: (v if isinstance(v, (int, float, bool)) or v is None or len(str(v)) < 500 else str(v)[:500] + '...') for k, v in context.items()}
            logger.info(f"[ContextLogger] Context values: {json.dumps(safe_context, indent=2)}")
            logger.error(f"[ContextLogger] [ERROR LEVEL] Context values: {json.dumps(safe_context, indent=2)}")
            # Explicitly log traffic_image_path and image_base64 if present
            for key in ["traffic_image_path", "image_base64"]:
                value = context.get(key, None)
                if value is not None:
                    logger.info(f"[ContextLogger] Explicit {key}: {value}")
        return {}
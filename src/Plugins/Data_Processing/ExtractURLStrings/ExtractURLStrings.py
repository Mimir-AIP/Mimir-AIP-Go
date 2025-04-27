"""
ExtractURLStrings module.

Extracts URL strings from a list of dictionaries in the pipeline context,
preferring 'link' and falling back to 'FirstURL'.
"""

from Plugins.BasePlugin import BasePlugin
import logging

class ExtractURLStrings(BasePlugin):
    """Plugin to extract URL strings from a list of dicts in the pipeline context."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Extract URLs from input data and map them to an output key.

        Args:
            step_config (dict): Step definition; includes:
                config (dict): With 'input' (str or List[dict]).
                output (str): Context key for storing results.
            context (dict): Current pipeline context.

        Returns:
            dict: {output_key: List[str]} of extracted URLs or fallback list.
        """
        config = step_config.get("config", {})
        input_data = config.get("input")
        if isinstance(input_data, str) and input_data in context:
            data = context[input_data]
        else:
            data = input_data
        logger = logging.getLogger(__name__)
        logger.info(f"[ExtractURLStrings] Received input: {data}")
        url_list = []
        try:
            if isinstance(data, list):
                for item in data:
                    if isinstance(item, dict):
                        if 'link' in item:
                            url_list.append(item['link'])
                        elif 'FirstURL' in item:
                            url_list.append(item['FirstURL'])
            logger.info(f"[ExtractURLStrings] Output url_list: {url_list}")
            if not url_list:
                logger.warning("[ExtractURLStrings] No URLs extracted; returning empty list. Returning friendly message.")
                # Return a friendlier message for empty lists
                return {step_config["output"]: ["No news URLs found."]}
            return {step_config["output"]: url_list}
        except Exception as e:
            logger.error(f"ExtractURLStrings: Error extracting URLs: {e}")
            return {step_config["output"]: []}
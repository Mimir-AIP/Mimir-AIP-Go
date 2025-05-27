import requests
import sys
import os

# Add the project root directory to the Python path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../../..')))

from src.Plugins.BasePlugin import BasePlugin

class Parsera(BasePlugin):
    plugin_type = "Input"
    BASEURL = "https://api.parsera.org/v1"

    def __init__(self):
        pass

    def llm_specs(self):
        """
        Fetch LLM specifications from the Parsera API.

        Returns:
            list: LLM specifications if successful, None otherwise
        """
        try:
            response = requests.get(f"{self.BASEURL}/llm-specs")
            if response.status_code == 200:
                return response.json()
            else:
                print(f"Error: {response.status_code}")
                return None
        except Exception as e:
            print(f"Exception: {str(e)}")
            return None

    def filter_models(self, models, name=None, provider=None, min_context_window=None,
                     min_output_tokens=None, capabilities=None):
        """
        Filter models based on various criteria.

        Args:
            models (list): List of model specifications to filter
            name (str, optional): Partial match for model name
            provider (str, optional): Exact match for provider
            min_context_window (int, optional): Minimum context window size
            min_output_tokens (int, optional): Minimum output tokens
            capabilities (list, optional): List of required capabilities

        Returns:
            list: Filtered list of models that match all criteria
        """
        if not models:
            return []

        filtered = []

        for model in models:
            match = True

            if name and name.lower() not in model.get('name', '').lower():
                match = False

            if provider and model.get('provider', '') != provider:
                match = False

            if min_context_window is not None:
                context_window = model.get('context_window')
                if context_window is None or context_window < min_context_window:
                    match = False

            if min_output_tokens is not None:
                max_output_tokens = model.get('max_output_tokens')
                if max_output_tokens is None or max_output_tokens < min_output_tokens:
                    match = False

            if capabilities:
                model_caps = set(model.get('capabilities', []))
                if not all(cap in model_caps for cap in capabilities):
                    match = False

            if match:
                filtered.append(model)

        return filtered

    def execute_pipeline_step(self, step_config, context):
        """
        Execute the pipeline step by calling the Parsera API.

        Args:
            step_config (dict): Configuration for this step
            context (dict): Current pipeline context

        Returns:
            dict: Updated context with API response or filtered models
        """
        try:
            # Check if we need to filter models
            filter_config = step_config.get("filter", {})
            if filter_config:
                # Get models from the API
                response = requests.get(f"{self.BASEURL}/llm-specs")
                if response.status_code == 200:
                    models = response.json()

                    # Apply filters
                    filtered_models = self.filter_models(
                        models,
                        name=filter_config.get("name"),
                        provider=filter_config.get("provider"),
                        min_context_window=filter_config.get("min_context_window"),
                        min_output_tokens=filter_config.get("min_output_tokens"),
                        capabilities=filter_config.get("capabilities")
                    )

                    # Update context with filtered models
                    context["filtered_parsera_models"] = filtered_models
                    return context
                else:
                    print(f"Error: {response.status_code}")
                    return context
            else:
                # If no filtering, perform a regular API call
                endpoint = step_config.get("endpoint", "")
                if not endpoint:
                    print("Error: No endpoint specified in step_config")
                    return context

                response = requests.get(f"{self.BASEURL}/{endpoint}")
                if response.status_code == 200:
                    context["parsera_response"] = response.json()
                    return context
                else:
                    print(f"Error: {response.status_code}")
                    return context
        except Exception as e:
            print(f"Exception: {str(e)}")
            return context
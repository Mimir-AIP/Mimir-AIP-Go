"""
Plugin for using LLM models as functions
"""

from Plugins.AIModels.BaseAIModel.BaseAIModel import BaseAIModel

class LLMFunction:
    """
    Plugin for using LLM models as functions
    """

    plugin_type = "Data Processing"

    def __init__(self, llm_plugin: BaseAIModel):
        """
        Initialize the LLMFunction plugin

        Args:
            llm_plugin (BaseAIModel): An instance of a plugin that inherits from BaseAIModel
        """
        if not isinstance(llm_plugin, BaseAIModel):
            raise ValueError("llm_plugin must be an instance of BaseAIModel")
        self.llm_plugin = llm_plugin

    def process_data(self, model, function, format_str, data):
        """
        Process data using an LLM model as a function

        Args:
            model (str): Model identifier (e.g., 'meta-llama/llama-3-8b-instruct:free')
            function (str): Description of the function to perform
            format_str (str): Expected format of the output
            data (str): Input data to process

        Returns:
            str: Processed data from the LLM
        """
        messages = [
            {"role": "user", "content": f"{function} {format_str} {data}"}
        ]
        
        response = self.llm_plugin.chat_completion(model=model, messages=messages)
        return response

if __name__ == "__main__":
    # Example usage
    from Plugins.AIModels.OpenRouter.OpenRouter import OpenRouter
    
    # Create OpenRouter plugin instance
    llm = OpenRouter()
    
    # Create LLMFunction with the OpenRouter plugin
    plugin = LLMFunction(llm)
    
    # Test parameters
    model = "meta-llama/llama-3-8b-instruct:free"
    function = "Summarize the following text"
    format_str = "Output should be a one-sentence summary"
    data = """
    This is a test document that contains multiple sentences.
    It has information about various topics.
    The goal is to create a concise summary.
    """
    
    try:
        result = plugin.process_data(model, function, format_str, data)
        print(f"Summary: {result}")
    except ValueError as e:
        print(f"Error: {e}")
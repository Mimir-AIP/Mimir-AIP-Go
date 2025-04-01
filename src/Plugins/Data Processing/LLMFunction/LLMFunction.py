#TODO create plugin which allows LLM to be used in place of code as a function
from Plugins.PluginManager import PluginManager

def LLMFunction(plugin, model, function, format, data):
    plugin_manager = PluginManager()
    llm_plugin = plugin_manager.get_plugin(plugin)
    if not llm_plugin:
        raise ValueError(f"Plugin {plugin} not found")
    LLM_Plugin = llm_plugin
    messages = [
        {"role": "user", "content": function + " " +format + " " + data}
    ]
    LLM_Plugin.generate_response(model, messages)


if __name__ == "__main__":
    plugin = "OpenRouter"
    model = "meta-llama/llama-3-8b-instruct:free"
    function = "You are a program block that takes RSS feeed items and determines importance on a scale of 1-10"
    format = "return score in the format {'score': 10}"
    data = "This is a test"
    try: 
        response = LLMFunction(plugin, model, function, format, data)
        print(response)
    except ValueError as e:
        print(e)
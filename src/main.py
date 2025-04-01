from Plugins.PluginManager import PluginManager
import yaml


def main():
    """Main entry point of the application"""
    # Step 1: Initialize the PluginManager
    plugin_manager = PluginManager()
    
    # Step 2: Load all plugins
    try:
        plugins = plugin_manager.get_all_plugins()
        if not plugins:
            print("No plugins found. Please ensure there are plugins available in the Plugins folder.")
            return
    except Exception as e:
        print(f"Failed to load plugins: {e}")
        return

    print(f"Loaded plugins: {', '.join(plugins.keys())}")

    """
    # Step 3: Load the pipeline configuration
    pipeline_config = None
    try:
        # Load pipeline.yaml file that contains pipeline actions or configuration
        with open("pipeline.yaml", "r") as f:
            pipeline_config = yaml.safe_load(f)
    except FileNotFoundError:
        print("Error: pipeline.yaml file not found. Please provide a configuration file.")
        return
    except yaml.YAMLError as e:
        print(f"Error parsing pipeline.yaml: {e}")
        return
    except Exception as e:
        print(f"An unexpected error occurred while loading pipeline.yaml: {e}")
        return

    # Step 4: Validate the pipeline configuration
    if pipeline_config is None or not isinstance(pipeline_config, dict):
        print("Invalid pipeline configuration. Please check your pipeline.yaml file.")
        return

    
    # Example pipeline structure:
    # pipeline.yaml sample:
    # tasks:
    #   - name: "Generate LLM Response"
    #     model: "OpenRouter"
    #     prompt: "Explain the importance of AI in modern software development."
    print("Pipeline configuration loaded successfully.")

    # Step 5: Execute the pipeline based on configuration
    tasks = pipeline_config.get("tasks", [])
    if not tasks:
        print("No tasks defined in the pipeline. Exiting.")
        return

    for task in tasks:
        task_name = task.get("name")
        model_name = task.get("model")
        prompt = task.get("prompt")

        if not task_name or not model_name or not prompt:
            print(f"Skipping incomplete task configuration: {task}")
            continue

        print(f"Executing task: {task_name} using model: {model_name}")

        # Retrieve the plugin
        plugin = plugin_manager.get_plugin(model_name)
        if plugin is None:
            print(f"Error: Model '{model_name}' is not available or not loaded as a plugin.")
            continue

        try:
            # Generate response using the plugin
            response = plugin.generate_response(prompt)
            print(f"Task '{task_name}' completed. Response:\n{response}")
        except Exception as e:
            print(f"An error occurred while executing task '{task_name}': {e}")

    # Step 6: Connect to or create a vector database for storage (optional for your exact use case)
    print("Pipeline execution completed.")"
    """

    #POC implementation harcoded for now
    print("Starting POC pipeline to generate reports for important news stories")
    while True:
        #Get BBC news RSS feed
        url = "http://feeds.bbci.co.uk/news/world/rss.xml"
        RSSFeed = PluginManager().get_plugin("RSS-Feed").get_feed(url, "BBC-News")
        RSSFeed.fetch_feed()
        print(RSSFeed.data)

        #for story in feed
        for story in RSSFeed.data["items"]:
            #Check if story is important using LLMFunction plugin
            LLMFunction = PluginManager().get_plugin("LLMFunction")
            plugin = "OpenRouter"
            model = "meta-llama/llama-3-8b-instruct:free"
            function = "You are a program block that takes RSS feeed items and determines importance on a scale of 1-10"
            format = "return score in the format {'score': 10}"
            article_to_score = story["title"]
            try: 
                response = LLMFunction.LLMFunction(plugin, model, function, format, article_to_score)
                print(response)
            except ValueError as e:
                print(e)
            #if LLMFunction determines story is important carry out research and generate report
            #parse response e.g. {"score": 10}
            if "score" in response:
                if response["score"] > 7:
                    #carry out research and generate report
                    #use LLMFunction plugin to generate search queries
                    LLMFunction = PluginManager().get_plugin("LLMFunction")
                    plugin = "OpenRouter"
                    model = "meta-llama/llama-3-8b-instruct:free"
                    function = "You are a program block that generates search queries for a given topic"
                    format = "return search queries in the format {'search_queries': ['query1', 'query2', 'query3']}"
                    topic_to_search = story["title"]
                    try: 
                        response = LLMFunction.LLMFunction(plugin, model, function, format, topic_to_search)
                        print(response)
                    except ValueError as e:
                        print(e)
                    #use websearch plugin to generate search results
                    WebSearch = PluginManager().get_plugin("WebSearch")
                    results = ""
                    for query in response["search_queries"]:
                        results += WebSearch.search(query)
                    #use LLMFunction plugin to generate report from search results
                    LLMFunction = PluginManager().get_plugin("LLMFunction")
                    plugin = "OpenRouter"
                    model = "meta-llama/llama-3-8b-instruct:free"
                    function = "You are a program block that generates a report from search results"
                    format = "return report in the format {'report': 'report text'}"
                    data = results
                    try: 
                        response = LLMFunction.LLMFunction(plugin, model, function, format, data)
                        print(response)
                    except ValueError as e:
                        print(e)
                    #output report
                    print("Report: " + topic_to_search + "\n" + response["report"])






if __name__ == "__main__":
    main()
#use the PluginManager to load all plugins from the plugins folder
import plugins.PluginManager
from plugins.PluginManager import load_plugins
import yaml

def main():
    """Main entry point of the application"""
    # Load all plugins
    plugin_manager = plugins.PluginManager.PluginManager()
    plugins = plugin_manager.load_plugins()
    pipeline_config = None
    try:
        # Load pipeline.yaml
        with open("pipeline.yaml", "r") as f:
            pipeline_config = yaml.safe_load(f)
    except FileNotFoundError:
        print("Error: pipeline.yaml file not found")
    except yaml.YAMLError as e:
        print(f"Error parsing pipeline.yaml: {e}")
    except Exception as e:
        print(f"An unexpected error occurred: {e}")

    if pipeline_config is None:
        return
    #run pipeline

    #create/connect to vector database for storage
    



if __name__ == "__main__":
    main()
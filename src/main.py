from Plugins.PluginManager import PluginManager
import yaml #used to load pipelines
import os
import logging



def main():
    """Main entry point of the application"""
    # Step 1: Load main configuration
    try:
        with open("config.yaml", "r") as f:
            config = yaml.safe_load(f)
    except FileNotFoundError:
        print("Error: config.yaml not found. Please ensure the configuration file exists.")
        return
    except yaml.YAMLError as e:
        print(f"Error parsing config.yaml: {e}")
        return
    except Exception as e:
        print(f"An unexpected error occurred while loading config.yaml: {e}")
        return

    # Setup based on configuration
    pipeline_dir = config.get("settings", {}).get("pipeline_directory", "pipelines")
    output_dir = config.get("settings", {}).get("output_directory", "output")
    log_level = config.get("settings", {}).get("log_level", "INFO")

    # Create output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)

    # Configure logging
    logging.basicConfig(
        level=getattr(logging, log_level),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    logger = logging.getLogger(__name__)

    # Step 2: Initialize the PluginManager
    plugin_manager = PluginManager()
    
    # Step 3: Load all plugins
    try:
        plugins = plugin_manager.get_all_plugins()
        if not plugins:
            logger.error("No plugins found. Please ensure there are plugins available in the Plugins folder.")
            return
    except Exception as e:
        logger.error(f"Failed to load plugins: {e}")
        return

    logger.info(f"Loaded plugins: {', '.join(plugins.keys())}")

    # Step 4: Load and execute enabled pipelines
    pipelines = config.get("pipelines", [])
    if not pipelines:
        logger.warning("No pipelines defined in configuration.")
        return

    for pipeline_config in pipelines:
        if not pipeline_config.get("enabled", False):
            logger.info(f"Skipping disabled pipeline: {pipeline_config.get('name', 'Unnamed')}")
            continue

        pipeline_file = pipeline_config.get("file")
        if not pipeline_file:
            logger.error(f"No file specified for pipeline: {pipeline_config.get('name', 'Unnamed')}")
            continue

        # Load pipeline definition
        try:
            with open(pipeline_file, "r") as f:
                pipeline_def = yaml.safe_load(f)
        except FileNotFoundError:
            logger.error(f"Pipeline file not found: {pipeline_file}")
            continue
        except yaml.YAMLError as e:
            logger.error(f"Error parsing pipeline file {pipeline_file}: {e}")
            continue
        except Exception as e:
            logger.error(f"An unexpected error occurred while loading {pipeline_file}: {e}")
            continue

        logger.info(f"Executing pipeline: {pipeline_config.get('name', 'Unnamed Pipeline')}")
        
        # Execute each pipeline
        for pipeline in pipeline_def.get("pipelines", []):
            try:
                execute_pipeline(pipeline, plugin_manager, output_dir)
            except Exception as e:
                logger.error(f"Error executing pipeline {pipeline.get('name', 'Unnamed')}: {e}")


def execute_pipeline(pipeline, plugin_manager, output_dir):
    """Execute a single pipeline definition"""
    logger = logging.getLogger(__name__)
    logger.info(f"Starting pipeline: {pipeline.get('name', 'Unnamed Pipeline')}")
    
    # Initialize pipeline context
    context = {}
    
    for step in pipeline["steps"]:
        if step.get("iterate"):
            # Handle iteration over items
            try:
                # Get the data to iterate over from context
                logger.debug(f"Current context: {context}")
                logger.debug(f"Evaluating iterate expression: {step['iterate']}")
                # Pass the context dictionary to eval
                data = eval(step["iterate"], {"__builtins__": {}}, {"context": context})
                logger.debug(f"Data to iterate over: {data}")
                for item in data:
                    # Create a new context for each iteration that includes both the item and the pipeline context
                    iteration_context = {"item": item, "output_dir": output_dir, **context}
                    logger.debug(f"Starting iteration with context: {iteration_context}")
                    for substep in step["steps"]:
                        logger.debug(f"Executing step: {substep.get('name', 'unnamed')}")
                        logger.debug(f"Step input: {iteration_context}")
                        updated_context = execute_step(substep, iteration_context, plugin_manager)
                        logger.debug(f"Step output: {updated_context}")
                        # Update the pipeline context with any changes from the iteration
                        context.update({k: v for k, v in iteration_context.items() if k != "item"})
                        logger.debug(f"Updated context after step {substep.get('name', 'unnamed')}: {context}")
            except Exception as e:
                logger.error(f"Error in iteration step: {e}")
                raise
        else:
            try:
                logger.debug(f"Executing step: {step.get('name', 'unnamed')}")
                logger.debug(f"Step input: {context}")
                updated_context = execute_step(step, {"output_dir": output_dir, **context}, plugin_manager)
                logger.debug(f"Step output: {updated_context}")
                if updated_context:
                    context.update(updated_context)
                    logger.debug(f"Updated context: {context}")
            except Exception as e:
                logger.error(f"Error in step: {e}")
                raise


def execute_step(step, context, plugin_manager):
    """Execute a single pipeline step"""
    logger = logging.getLogger(__name__)
    try:
        # Get plugin instance
        plugin_name = step["plugin"]
        
        # Try to find the plugin in each type
        plugin_instance = None
        for plugin_type in ["Input", "Output", "Data_Processing", "AIModels"]:
            plugin_instance = plugin_manager.get_plugin(plugin_type, plugin_name)
            if plugin_instance:
                break
                
        if not plugin_instance:
            logger.error(f"Plugin {plugin_name} not found in any plugin type")
            return

        logger.info(f"Executing step: {step.get('name', 'unnamed')}")

        # Let the plugin handle its own step execution
        updated_context = plugin_instance.execute_pipeline_step(step, context)
        if updated_context:
            logger.debug(f"Updated context: {updated_context}")
            context.update(updated_context)

        # Handle conditional execution
        if "condition" in step:
            condition_result = eval(step["condition"], context)
            if condition_result and "steps" in step:
                for substep in step["steps"]:
                    execute_step(substep, context, plugin_manager)

    except Exception as e:
        logger.error(f"Error executing step {step.get('name', 'unnamed')}: {e}")
        raise  # Re-raise to handle in caller


if __name__ == "__main__":
    main()
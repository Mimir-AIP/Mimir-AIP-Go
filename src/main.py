from Plugins.PluginManager import PluginManager
import yaml #used to load pipelines
import os
import logging
from PipelineVisualizer.AsciiTree import PipelineAsciiTreeVisualizer

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
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.FileHandler("mimir.log", mode="w"),
            logging.StreamHandler()
        ]
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

        # Robustly resolve pipeline_file relative to project root
        project_root = os.path.dirname(os.path.abspath(__file__))
        pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file

        # Load pipeline definition
        try:
            with open(pipeline_file_path, "r") as f:
                pipeline_def = yaml.safe_load(f)
        except FileNotFoundError:
            logger.error(f"Pipeline file not found: {pipeline_file_path}")
            continue
        except yaml.YAMLError as e:
            logger.error(f"Error parsing pipeline file {pipeline_file_path}: {e}")
            continue
        except Exception as e:
            logger.error(f"An unexpected error occurred while loading {pipeline_file_path}: {e}")
            continue

        logger.info(f"Executing pipeline: {pipeline_config.get('name', 'Unnamed Pipeline')}")
        
        # Execute each pipeline
        test_mode = config.get("settings", {}).get("test_mode", False)
        for pipeline in pipeline_def.get("pipelines", []):
            try:
                execute_pipeline(pipeline, plugin_manager, output_dir, test_mode=test_mode)
            except Exception as e:
                logger.error(f"Error executing pipeline {pipeline.get('name', 'Unnamed')}: {e}")


def execute_pipeline(pipeline, plugin_manager, output_dir, test_mode=False):
    """Execute a single pipeline definition with ASCII tree visualization"""
    logger = logging.getLogger(__name__)
    logger.info(f"Starting pipeline: {pipeline.get('name', 'Unnamed Pipeline')}")
    
    # Automated cleanup for test mode
    if test_mode:
        import os
        for fname in ["section_summaries.json", "reports/report.html"]:
            fpath = os.path.join(os.path.dirname(__file__), fname) if not fname.startswith("reports/") else os.path.join(os.path.dirname(__file__), "..", fname)
            try:
                if os.path.exists(fpath):
                    os.remove(fpath)
                    logger.info(f"[CLEANUP] Removed old file: {fpath}")
            except Exception as e:
                logger.warning(f"[CLEANUP] Could not remove {fpath}: {e}")

    # Initialize pipeline context and status tracking
    context = {"output_dir": output_dir, "test_mode": test_mode}
    step_statuses = {step.get('name', f'step_{i}'): 'pending' for i, step in enumerate(pipeline["steps"])}

    def render_tree(highlight_idx=None, runtime_info=None):
        tree = PipelineAsciiTreeVisualizer.build_tree_from_pipeline(pipeline, step_statuses, runtime_info=runtime_info)
        PipelineAsciiTreeVisualizer.render(tree, highlight_path=[highlight_idx] if highlight_idx is not None else None)

    iteration_tracking = {}
    for idx, step in enumerate(pipeline["steps"]):
        step_name = step.get('name', f'step_{idx}')
        try:
            step_statuses[step_name] = 'running'
            # For iterative steps, track iteration count, statuses, and labels
            if step.get("iterate"):
                data = []
                labels = []
                try:
                    data = eval(step["iterate"], {"__builtins__": {}}, {"context": context})
                    # Try to extract a label for each item if possible (e.g., title, name, id, str(item))
                    for item in data:
                        label = None
                        if isinstance(item, dict):
                            for key in ["title", "name", "id"]:
                                if key in item and isinstance(item[key], str):
                                    label = item[key][:40]  # Truncate for display
                                    break
                        if not label:
                            label = str(item)[:40]
                        labels.append(label)
                except Exception:
                    pass
                iter_count = len(data) if isinstance(data, list) else 0
                iteration_tracking[step_name] = {'count': iter_count, 'statuses': ['pending'] * iter_count, 'labels': labels}
                render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
                if iter_count > 0:
                    for i, item in enumerate(data):
                        iteration_context = {"item": item, "output_dir": output_dir, **context}
                        for substep in step["steps"]:
                            execute_step(substep, iteration_context, plugin_manager)
                        if "section_summaries" in iteration_context:
                            context["section_summaries"] = iteration_context["section_summaries"]
                        iteration_tracking[step_name]['statuses'][i] = 'completed'
                        render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
                step_statuses[step_name] = 'completed'
                render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
            else:
                render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
                execute_step(step, context, plugin_manager)
                step_statuses[step_name] = 'completed'
                render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
        except Exception as e:
            logger.error(f"Error in step: {e}")
            step_statuses[step_name] = 'failed'
            render_tree(highlight_idx=idx, runtime_info={'iterations': iteration_tracking})
            raise


def execute_step(step, context, plugin_manager):
    """Execute a single pipeline step"""
    logger = logging.getLogger(__name__)
    # Evaluate step condition if present
    if "condition" in step:
        try:
            # Check for None context variables in condition
            condition_vars = [var.split("[")[0].split(".")[0] for var in step["condition"].replace("'", "").replace("]", "").split() if var.isidentifier()]
            for var in condition_vars:
                if var in context and context[var] is None:
                    logger.warning(f"[STEP SKIP] Skipping step '{step.get('name', 'unnamed')}' because context variable '{var}' is None (likely due to API failure or prior error).")
                    return
            condition_result = eval(step["condition"], {"__builtins__": {}}, context)
            if not condition_result:
                logger.info(f"[STEP SKIP] Skipping step '{step.get('name', 'unnamed')}' due to failing condition: {step['condition']}")
                return
        except Exception as e:
            logger.error(f"[STEP SKIP] Error evaluating condition for step {step.get('name', 'unnamed')}: {e}")
            return
    try:
        # PATCH: Evaluate condition before executing the step itself
        if "condition" in step:
            try:
                condition_result = eval(step["condition"], {"__builtins__": {}}, context)
                if not condition_result:
                    logger.info(f"[STEP SKIP] Skipping step '{step.get('name', 'unnamed')}' due to failing condition: {step['condition']}")
                    return
            except Exception as e:
                logger.error(f"Error evaluating condition for step {step.get('name', 'unnamed')}: {e}")
                return
        logger.info(f"[STEP DEBUG] Step: {step.get('name', 'unnamed')}, Plugin: {step.get('plugin', None)}")
        logger.info(f"[STEP DEBUG] Context keys: {list(context.keys())}")
        logger.info(f"[STEP DEBUG] Context 'item' value: {context.get('item', None)} (type: {type(context.get('item', None))})")
        # If this step is a container (no plugin, only substeps), just execute substeps if present
        if "plugin" not in step:
            if "steps" in step:
                for substep in step["steps"]:
                    execute_step(substep, context, plugin_manager)
            return
        # Get plugin instance
        plugin_name = step["plugin"]
        # HARD GUARD: If this is a WebScraping step, only allow string URLs
        if plugin_name == "WebScraping":
            item = context.get('item', None)
            if not (isinstance(item, str) and item.startswith('http')):
                logger.error(f"[HARD GUARD] Skipping WebScraping step: item is not a URL string. Got: {item} (type: {type(item)})")
                return
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
        logger.debug(f"Updated context: {updated_context}")
        if updated_context:
            context.update(updated_context)
        # Handle conditional execution for substeps
        if "condition" in step:
            # Patch: use pipeline context as locals for eval
            condition_result = eval(step["condition"], {"__builtins__": {}}, context)
            if condition_result and "steps" in step:
                for substep in step["steps"]:
                    execute_step(substep, context, plugin_manager)

    except Exception as e:
        logger.error(f"Error executing step {step.get('name', 'unnamed')}: {e}")
        raise  # Re-raise to handle in caller


if __name__ == "__main__":
    main()
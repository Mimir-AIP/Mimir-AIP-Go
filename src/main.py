"""
Mimir-AIP Main Module

Entry point for pipeline execution. Loads config.yaml, initializes plugins,
and executes configured pipelines with ASCII visualization of step statuses.
"""
import os
import sys
import logging
import argparse
import yaml #used to load pipelines
import datetime  # Added for scheduler loop
import time      # Added for scheduler loop sleep
from Plugins.PluginManager import PluginManager
from PipelineVisualizer.AsciiTree import PipelineAsciiTreeVisualizer
import threading
import signal
from PipelineScheduler import CronSchedule

# Configure logging BEFORE any other imports that might log
logging.basicConfig(
    level=logging.DEBUG,  # Set to DEBUG for detailed diagnostics
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("mimir.log", mode="w"),
        logging.StreamHandler()
    ],
    force=True
)
logger = logging.getLogger(__name__)
logger.info("[Test] Logging to file and console should work now.")

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

    # (Optional) Adjust log level based on config
    logging.getLogger().setLevel(getattr(logging, log_level))
    logger.info(f"[Startup] CWD: {os.getcwd()}, Python exec: {sys.executable}")

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
        """Render the pipeline tree with current statuses and highlighting.

        Args:
            highlight_idx (int, optional): Index of the active step to highlight.
            runtime_info (dict, optional): Runtime info for iteration statuses.
        """
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
    """Execute a single pipeline step using the correct plugin lookup logic."""
    logger = logging.getLogger(__name__)
    plugin_ref = step.get('plugin')
    if not plugin_ref:
        logger.error(f"No plugin specified for step: {step.get('name', 'Unnamed')}")
        return

    # Split plugin_ref like 'Output.HTMLReport' or 'Data_Processing.Delay'
    if '.' in plugin_ref:
        plugin_type, plugin_name = plugin_ref.split('.', 1)
    else:
        plugin_type, plugin_name = None, plugin_ref

    # Try to get plugin by type and name
    plugin_instance = None
    if plugin_type:
        plugins_of_type = plugin_manager.get_plugins(plugin_type)
        if plugins_of_type and plugin_name in plugins_of_type:
            plugin_instance = plugins_of_type[plugin_name]
        else:
            logger.error(f"Plugin {plugin_ref} not found in plugin type {plugin_type}")
            return
    else:
        # Try all plugins if type not specified
        for type_name, plugins_of_type in plugin_manager.get_plugins().items():
            if plugin_name in plugins_of_type:
                plugin_instance = plugins_of_type[plugin_name]
                break
        if not plugin_instance:
            logger.error(f"Plugin {plugin_ref} not found in any plugin type")
            return

    # Execute the pipeline step
    try:
        result = plugin_instance.execute_pipeline_step(step, context)
        # Defensive logging: log type and sample of every value added to context
        if result:
            for k, v in result.items():
                logger.info(f"[ContextUpdate] Key: {k}, Type: {type(v)}, Sample: {str(v)[:300]}")
            # Robust context update: merge all result keys into context, do not overwrite
            for k, v in result.items():
                context[k] = v
    except Exception as e:
        logger.error(f"Error executing step {step.get('name', 'Unnamed')}: {e}")


def run_scheduled_pipelines(config, plugin_manager, output_dir):
    """Run scheduled pipelines, patched to avoid IndexError on empty schedules."""
    logger = logging.getLogger(__name__)
    pipelines = config.get("pipelines", [])
    schedules = [(p, p.get("schedule")) for p in pipelines if p.get("enabled", False) and p.get("schedule")]
    if not schedules:
        logger.info("No scheduled pipelines to run. Exiting scheduler loop.")
        return
    # ... (rest of original logic)
    scheduled = []
    manual = []
    for pipeline_config, sched_expr in schedules:
        try:
            sched = CronSchedule(sched_expr)
            scheduled.append((pipeline_config, sched))
        except Exception as e:
            logger.error(f"Invalid schedule for pipeline {pipeline_config.get('name')}: {e}")
        else:
            manual.append(pipeline_config)

    # Run manual pipelines immediately
    for pipeline_config in manual:
        logger.info(f"[SCHEDULER] Running manual pipeline: {pipeline_config.get('name')}")
        pipeline_file = pipeline_config.get("file")
        if not pipeline_file:
            logger.error(f"No file specified for pipeline: {pipeline_config.get('name', 'Unnamed')}")
            continue
        project_root = os.path.dirname(os.path.abspath(__file__))
        pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file
        try:
            with open(pipeline_file_path, "r") as f:
                pipeline_def = yaml.safe_load(f)
        except Exception as e:
            logger.error(f"Failed to load pipeline {pipeline_file_path}: {e}")
            continue
        for pipeline in pipeline_def.get("pipelines", []):
            try:
                execute_pipeline(pipeline, plugin_manager, output_dir)
            except Exception as e:
                logger.error(f"Error executing pipeline {pipeline.get('name', 'Unnamed')}: {e}")

    """
    Scheduler loop for scheduled pipelines
    """
    def scheduler_loop():
        logger.info("[SCHEDULER] Starting scheduler loop for pipelines...")
        next_runs = []
        for pipeline_config, sched in scheduled:
            next_run = sched.next_run()
            next_runs.append((next_run, pipeline_config, sched))
            logger.info(f"[SCHEDULER] Pipeline '{pipeline_config.get('name')}' scheduled for {next_run}")
        while True:
            now = datetime.datetime.now().replace(second=0, microsecond=0)
            # Find the soonest next run
            next_runs.sort()
            soonest, pipeline_config, sched = next_runs[0]
            sleep_secs = (soonest - now).total_seconds()
            if sleep_secs > 0:
                logger.info(f"[SCHEDULER] Sleeping for {sleep_secs:.1f} seconds until next pipeline: {pipeline_config.get('name')}")
                time.sleep(min(sleep_secs, 60))  # Sleep in chunks in case of signal
                continue
            # Time to run the pipeline
            logger.info(f"[SCHEDULER] Running scheduled pipeline: {pipeline_config.get('name')} at {now}")
            pipeline_file = pipeline_config.get("file")
            if not pipeline_file:
                logger.error(f"No file specified for pipeline: {pipeline_config.get('name', 'Unnamed')}")
                next_run = sched.next_run(now)
                next_runs[0] = (next_run, pipeline_config, sched)
                continue
            project_root = os.path.dirname(os.path.abspath(__file__))
            pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file
            try:
                with open(pipeline_file_path, "r") as f:
                    pipeline_def = yaml.safe_load(f)
            except Exception as e:
                logger.error(f"Failed to load pipeline {pipeline_file_path}: {e}")
                next_run = sched.next_run(now)
                next_runs[0] = (next_run, pipeline_config, sched)
                continue
            for pipeline in pipeline_def.get("pipelines", []):
                try:
                    execute_pipeline(pipeline, plugin_manager, output_dir)
                except Exception as e:
                    logger.error(f"Error executing pipeline {pipeline.get('name', 'Unnamed')}: {e}")
            # Schedule next run
            next_run = sched.next_run(now)
            next_runs[0] = (next_run, pipeline_config, sched)

    # Run scheduler loop in foreground (can be made a thread if needed)
    try:
        scheduler_loop()
    except KeyboardInterrupt:
        logger.info("[SCHEDULER] Shutting down scheduler loop.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Mimir-AIP Pipeline Runner")
    parser.add_argument('--pipeline', type=str, help='Name of pipeline to run once (manual trigger)')
    args = parser.parse_args()

    try:
        with open("config.yaml", "r") as f:
            config = yaml.safe_load(f)
    except Exception as e:
        print(f"Error loading config.yaml: {e}")
        exit(1)

    pipeline_dir = config.get("settings", {}).get("pipeline_directory", "pipelines")
    output_dir = config.get("settings", {}).get("output_directory", "output")
    log_level = config.get("settings", {}).get("log_level", "INFO")
    os.makedirs(output_dir, exist_ok=True)
    plugin_manager = PluginManager()

    if args.pipeline:
        # Manual trigger: run the specified pipeline once
        pipelines = config.get('pipelines', [])
        selected = next((p for p in pipelines if p.get('name') == args.pipeline), None)
        if not selected:
            print(f"Pipeline '{args.pipeline}' not found in config.yaml.")
            exit(1)
        # Load the pipeline YAML
        pipeline_file = selected.get('file')
        if not pipeline_file:
            print(f"No file specified for pipeline '{args.pipeline}'.")
            exit(1)
        pipeline_path = os.path.join(pipeline_dir, os.path.basename(pipeline_file))
        try:
            with open(pipeline_path, 'r') as pf:
                pipeline_yaml = yaml.safe_load(pf)
        except Exception as e:
            print(f"Error loading pipeline YAML: {e}")
            exit(1)
        # Find the actual pipeline definition (list under 'pipelines')
        pipeline_defs = pipeline_yaml.get('pipelines', [])
        if not pipeline_defs:
            print(f"No pipelines found in {pipeline_file}.")
            exit(1)
        # Run the first (or only) pipeline in the file
        execute_pipeline(pipeline_defs[0], plugin_manager, output_dir)
    else:
        # Improved default behavior: run enabled pipeline(s) or prompt user
        pipelines = config.get('pipelines', [])
        enabled_pipelines = [p for p in pipelines if p.get('enabled', False)]
        if not enabled_pipelines:
            print("Error: No enabled pipelines found in config.yaml. Please enable at least one pipeline or specify --pipeline <name>.")
            exit(1)
        elif len(enabled_pipelines) == 1:
            selected = enabled_pipelines[0]
            print(f"No pipeline specified. Running the only enabled pipeline: {selected.get('name')}")
        else:
            print("Multiple enabled pipelines found. Please select one to run:")
            for idx, p in enumerate(enabled_pipelines, 1):
                print(f"  {idx}. {p.get('name')}")
            while True:
                try:
                    choice = input(f"Enter a number (1-{len(enabled_pipelines)}): ").strip()
                    num = int(choice)
                    if 1 <= num <= len(enabled_pipelines):
                        selected = enabled_pipelines[num-1]
                        break
                    else:
                        print(f"Invalid selection. Please enter a number between 1 and {len(enabled_pipelines)}.")
                except (ValueError, KeyboardInterrupt):
                    print("Input cancelled or invalid. Exiting.")
                    exit(1)
        pipeline_file = selected.get('file')
        if not pipeline_file:
            print(f"No file specified for pipeline '{selected.get('name')}'.")
            exit(1)
        pipeline_path = os.path.join(pipeline_dir, os.path.basename(pipeline_file))
        try:
            with open(pipeline_path, 'r') as pf:
                pipeline_yaml = yaml.safe_load(pf)
        except Exception as e:
            print(f"Error loading pipeline YAML: {e}")
            exit(1)
        pipeline_defs = pipeline_yaml.get('pipelines', [])
        if not pipeline_defs:
            print(f"No pipelines found in {pipeline_file}.")
            exit(1)
        execute_pipeline(pipeline_defs[0], plugin_manager, output_dir)
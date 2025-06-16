MIMIR_HEADER = r"""
 ███╗   ███╗██╗███╗   ███╗██╗██████╗      █████╗ ██╗██████╗
████╗ ████║██║████╗ ████║██║██╔══██╗    ██╔══██╗██║██╔══██╗
██╔████╔██║██║██╔████╔██║██║██████╔╝    ███████║██║██████╔╝
██║╚██╔╝██║██║██║╚██╔╝██║██║██╔══██╗    ██╔══██║██║██╔═══╝
██║ ╚═╝ ██║██║██║ ╚═╝ ██║██║██║  ██║    ██║  ██║██║██║
╚═╝     ╚═╝╚═╝╚═╝     ╚═╝╚═╝╚═╝  ╚═╝    ╚═╝  ╚═╝╚═╝╚═╝
                                                            
Pipeline Execution Engine - Version 2.0
"""
import os
import sys
import logging
import argparse
import time
import datetime
import json
from time import sleep
import yaml # Used to load pipelines
from typing import Any, Dict, List, Optional, Tuple
from Plugins.PluginManager import PluginManager
from PipelineVisualizer.AsciiTree import PipelineAsciiTreeVisualizer
import threading
import signal
from PipelineScheduler import CronSchedule
from pipeline_parser.pipeline_parser import PipelineParser # Import the new parser
from pipeline_parser.control_graph import ControlGraph # Import the ControlGraph
from pipeline_parser.stateful_executor import StatefulExecutor # Import the StatefulExecutor
from pipeline_parser.ast_nodes import RootNode # Import RootNode for type hints
from ContextService import ContextService # Import ContextService

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

    # Step 2: Initialize ContextService and PluginManager with config
    print(f"[DEBUG main.py] Config loaded and initializing ContextService and PluginManager.")
    context_service = ContextService(config=config) # Initialize ContextService with config
    plugin_manager = PluginManager(config=config, context_service=context_service) # Pass ContextService to PluginManager
    
    # Step 3: Load all plugins with visual feedback
    print("\n🔌 Loading plugins...")
    loaded_plugins = {}
    try:
        plugins = plugin_manager.get_all_plugins()
        if not plugins:
            print("❌ No plugins found. Please ensure there are plugins available in the Plugins folder.")
            return
            
        for plugin_type, plugin_dict in plugins.items():
            print(f"  ⚙️ {plugin_type}:")
            for plugin_name in plugin_dict.keys():
                print(f"    ✅ {plugin_name}")
                loaded_plugins[f"{plugin_type}.{plugin_name}"] = True
                sleep(0.1)  # Small delay for visual effect
                
        print(f"\n✔ Successfully loaded {len(loaded_plugins)} plugins")
    except Exception as e:
        print(f"❌ Failed to load plugins: {e}")
        return

    # Step 4: Initialize PipelineParser
    pipeline_parser = PipelineParser()

    # Step 5: Load and execute enabled pipelines
    pipelines_to_process = config.get("pipelines", [])
    if not pipelines_to_process:
        logger.warning("No pipelines defined in configuration.")
        return

    for pipeline_config_entry in pipelines_to_process:
        if not pipeline_config_entry.get("enabled", False):
            logger.info(f"Skipping disabled pipeline: {pipeline_config_entry.get('name', 'Unnamed')}")
            continue

        pipeline_file = pipeline_config_entry.get("file")
        if not pipeline_file:
            logger.error(f"No file specified for pipeline: {pipeline_config_entry.get('name', 'Unnamed')}")
            continue

        # Robustly resolve pipeline_file relative to project root
        project_root = os.path.dirname(os.path.abspath(__file__))
        pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file

        # Load pipeline definition using PipelineParser
        try:
            with open(pipeline_file_path, "r") as f:
                yaml_content = f.read()
            
            pipeline_def = pipeline_parser.parse(yaml_content)
            if pipeline_def is None:
                logger.error(f"Failed to parse pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                continue
            
            if not pipeline_parser.validate(pipeline_def):
                logger.error(f"Validation failed for pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                continue

        except FileNotFoundError:
            logger.error(f"Pipeline file not found: {pipeline_file_path}")
            continue
        except Exception as e:
            logger.error(f"An unexpected error occurred while loading or parsing {pipeline_file_path}: {e}")
            continue

        logger.info(f"Executing pipeline: {pipeline_config_entry.get('name', 'Unnamed Pipeline')}")
        
        # Execute each pipeline defined within the loaded YAML file
        test_mode = config.get("settings", {}).get("test_mode", False)
        # Convert to AST and build control graph
        pipeline_ast = pipeline_parser.to_ast(pipeline_def)
        control_graph = ControlGraph(pipeline_ast)

        # Check for cycles
        cycles = control_graph.detect_cycles()
        if cycles:
            logger.error(f"Cycles detected in pipeline '{pipeline_config_entry.get('name', 'Unnamed')}': {cycles}")
            # Depending on policy, you might want to raise an error or skip execution
            continue
        
        # Log graph errors if any
        graph_errors = control_graph.get_errors()
        if graph_errors:
            for error in graph_errors:
                logger.error(f"Control Graph error for pipeline '{pipeline_config_entry.get('name', 'Unnamed')}': {error}")
            continue

        # You can also generate the DOT string for visualization if needed
        # dot_string = control_graph.visualize()
        # logger.debug(f"DOT representation for pipeline '{pipeline_config_entry.get('name', 'Unnamed')}':\n{dot_string}")

        # Execute each pipeline defined within the loaded YAML file
        test_mode = config.get("settings", {}).get("test_mode", False)
        # Run scheduled pipelines if configured
        if config.get("settings", {}).get("enable_scheduler", False):
            pipeline_runner = PipelineRunner(config, plugin_manager, context_service)
            # Enable visual debugging if configured
            if config.get('visual_debugging', {}).get('enabled', False):
                pipeline_runner.enable_visual_debugging(config['visual_debugging'])
            pipeline_runner.scheduler_loop()

        # Execute each pipeline defined within the loaded YAML file
        for pipeline_dict in pipeline_def.get("pipelines", []): # pipeline_def is still the dict, not AST
            try:
                # Convert the specific pipeline dict to AST for StatefulExecutor
                single_pipeline_ast = pipeline_parser.to_ast({"pipelines": [pipeline_dict]})
                if single_pipeline_ast is None:
                    logger.error(f"Failed to convert pipeline '{pipeline_dict.get('name', 'Unnamed')}' to AST.")
                    continue

                execute_pipeline(pipeline_dict, single_pipeline_ast, plugin_manager, output_dir, test_mode=test_mode, context_service=context_service)
            except Exception as e:
                logger.error(f"Error executing pipeline {pipeline_dict.get('name', 'Unnamed')}: {e}")


def execute_pipeline(pipeline_dict: Dict[str, Any], pipeline_ast: RootNode, plugin_manager: PluginManager, output_dir: str, test_mode: bool = False, context_service: Optional[ContextService] = None, run_count: int = 1, next_run_time: Optional[datetime] = None):
    """Execute a pipeline with support for single, scheduled, or continuous execution using StatefulExecutor."""
    logger = logging.getLogger(__name__)
    execution_mode = pipeline_dict.get('execution_mode', 'single')
    pipeline_name = pipeline_dict.get('name', 'Unnamed Pipeline')
    logger.info(f"Starting pipeline: {pipeline_name} (mode: {execution_mode})")

    # Initialize StatefulExecutor
    initial_context = {"output_dir": output_dir, "test_mode": test_mode}
    executor = StatefulExecutor(pipeline_ast, plugin_manager, context_service, initial_context)
    visualizer = PipelineAsciiTreeVisualizer()

    def render_tree_from_executor_state():
        """Render the pipeline tree with current statuses and highlighting from executor state."""
        full_runtime_info = {
            'execution_mode': execution_mode,
            'run_count': run_count,
        }
        if next_run_time:
            full_runtime_info['next_run'] = next_run_time.strftime("%Y-%m-%d %H:%M:%S")

        tree_data = executor.get_pipeline_execution_state()
        # The root of the visualizer expects a single tree, not the 'root' key itself
        # We need to pass the actual pipeline node from the executor's state
        if 'root' in tree_data and 'children' in tree_data['root'] and tree_data['root']['children']:
            # Assuming the first child of 'root' is the actual pipeline we want to visualize
            # This might need adjustment based on how _initialize_execution_state builds the tree
            visualizer.render(tree_data['root'], runtime_info=full_runtime_info)
        else:
            logger.warning("No valid pipeline structure found in executor state for visualization.")


    def _execute_single_run_with_executor(current_run_count: int = 1, next_run_time: Optional[datetime] = None):
        """Execute one complete run of the pipeline using StatefulExecutor."""
        logger.info(f"Executing single run for pipeline: {pipeline_name}")
        
        # Execute the pipeline using the StatefulExecutor
        success = executor.execute_pipeline(pipeline_name)
        
        # Render the final state
        render_tree_from_executor_state()

        if not success:
            logger.error(f"Pipeline '{pipeline_name}' execution failed. Errors: {executor.get_errors()}")
            raise RuntimeError(f"Pipeline execution failed: {executor.get_errors()}")

    # Handle different execution modes
    if execution_mode == 'continuous':
        run_count = 0
        while True:
            run_count += 1
            logger.info(f"Starting continuous pipeline run #{run_count}")
            try:
                _execute_single_run_with_executor(current_run_count=run_count)
            except KeyboardInterrupt:
                logger.info("Continuous execution interrupted by user")
                break
            except Exception as e:
                logger.error(f"Error in continuous run: {e}")
                time.sleep(5)  # Brief pause before retry
    elif execution_mode == 'scheduled':
        # This part will be handled by run_scheduled_pipelines, which will call execute_pipeline
        # with appropriate next_run_time. For now, we'll just execute once if called directly.
        logger.info(f"Scheduled pipeline '{pipeline_name}' called directly. Executing once.")
        _execute_single_run_with_executor(run_count=run_count, next_run_time=next_run_time)
    else: # single execution mode
        _execute_single_run_with_executor()

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


class PipelineRunner:
    """Main pipeline runner with scheduler and visual debugging support."""
    
    def __init__(self, config: Dict[str, Any], plugin_manager: PluginManager, context_service: ContextService):
        self.config = config
        self.plugin_manager = plugin_manager
        self.context_service = context_service
        self.running = False
        self.visual_debugging_enabled = False
        self.visual_debug_config = {}
        
        # Initialize visual debugging if configured
        if config.get('visual_debugging', {}).get('enabled', False):
            self.enable_visual_debugging(config.get('visual_debugging', {}))
    
    def enable_visual_debugging(self, debug_config: Dict[str, Any]):
        """Enable visual debugging with the specified configuration."""
        self.visual_debugging_enabled = True
        self.visual_debug_config = debug_config
        logger.info(f"Visual debugging enabled with config: {debug_config}")
    
    def scheduler_loop(self):
        """Main scheduler loop that runs continuously checking for scheduled pipelines."""
        logger.info("Starting pipeline scheduler loop...")
        self.running = True
        
        # Initialize pipeline parser
        pipeline_parser = PipelineParser()
        
        while self.running:
            try:
                current_time = datetime.now()
                
                # Check each pipeline for scheduled execution
                for pipeline_config in self.config.get('pipelines', []):
                    if not pipeline_config.get('enabled', True):
                        continue
                        
                    schedule = pipeline_config.get('schedule')
                    if not schedule:
                        continue
                        
                    pipeline_name = pipeline_config['name']
                    
                    # Check if pipeline should run now
                    cron_schedule = CronSchedule(schedule)
                    if cron_schedule.is_now(current_time):
                        logger.info(f"Executing scheduled pipeline: {pipeline_name}")
                        
                        try:
                            # Load and parse the pipeline file
                            pipeline_file = pipeline_config.get('file')
                            if not pipeline_file:
                                logger.error(f"No file specified for pipeline: {pipeline_name}")
                                continue
                                
                            project_root = os.path.dirname(os.path.abspath(__file__))
                            pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file
                            
                            with open(pipeline_file_path, "r") as f:
                                yaml_content = f.read()
                            
                            pipeline_def = pipeline_parser.parse(yaml_content)
                            if pipeline_def is None:
                                logger.error(f"Failed to parse pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                                continue
                            
                            if not pipeline_parser.validate(pipeline_def):
                                logger.error(f"Validation failed for pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                                continue
                            
                            # Convert to AST
                            pipeline_ast = pipeline_parser.to_ast(pipeline_def)
                            if pipeline_ast is None:
                                logger.error(f"Failed to convert pipeline to AST: {pipeline_name}")
                                continue
                            
                            # Create StatefulExecutor with proper parameters
                            initial_context = {
                                "output_dir": self.config.get("settings", {}).get("output_directory", "output"),
                                "test_mode": self.config.get("settings", {}).get("test_mode", False)
                            }
                            
                            executor = StatefulExecutor(
                                ast=pipeline_ast,
                                plugin_manager=self.plugin_manager,
                                initial_context=initial_context
                            )
                            
                            # Enable visual debugging if configured
                            if self.visual_debugging_enabled:
                                self._setup_visual_debugging(executor)
                            
                            # Execute the pipeline - get the first pipeline from the definition
                            pipeline_defs = pipeline_def.get("pipelines", [])
                            if pipeline_defs:
                                pipeline_to_run = pipeline_defs[0]
                                pipeline_name = pipeline_to_run.get('name', 'Unnamed')
                                
                                # Add visual debug context if enabled
                                if self.visual_debugging_enabled:
                                    # Update initial context with debug info
                                    if 'context' not in pipeline_to_run:
                                        pipeline_to_run['context'] = {}
                                    pipeline_to_run['context'].update({
                                        'debug': True,
                                        'debug_output_dir': self.visual_debug_config.get('output_dir', 'debug_output'),
                                        'debug_timestamp': datetime.now().strftime('%Y%m%d_%H%M%S')
                                    })
                                
                                # Execute the pipeline
                                result = executor.execute_pipeline(pipeline_name)
                                
                                if result:
                                    logger.info(f"Pipeline {pipeline_name} completed successfully")
                                    
                                    # Generate visual debug output if enabled
                                    if self.visual_debugging_enabled:
                                        try:
                                            self._generate_debug_output(executor, pipeline_name)
                                        except Exception as e:
                                            logger.error(f"Error generating debug output: {e}", exc_info=True)
                                else:
                                    errors = executor.get_errors()
                                    logger.error(f"Pipeline {pipeline_name} failed: {errors}")
                                    
                                    # Still generate debug output on failure if enabled
                                    if self.visual_debugging_enabled and self.visual_debug_config.get('debug_on_error', True):
                                        try:
                                            self._generate_debug_output(executor, f"{pipeline_name}_error")
                                        except Exception as e:
                                            logger.error(f"Error generating error debug output: {e}", exc_info=True)
                            else:
                                logger.error(f"No pipeline definitions found in {pipeline_file_path}")
                                
                        except Exception as e:
                            logger.error(f"Error executing scheduled pipeline {pipeline_name}: {e}")
                
                # Sleep for a short interval before checking again
                time.sleep(60)  # Check every minute
                
            except KeyboardInterrupt:
                logger.info("Scheduler loop interrupted by user")
                self.running = False
                break
            except Exception as e:
                logger.error(f"Error in scheduler loop: {e}")
                time.sleep(60)  # Wait before retrying
    
    def _setup_visual_debugging(self, executor: StatefulExecutor) -> None:
        """
        Setup visual debugging for the executor.
        
        Args:
            executor: The StatefulExecutor instance to configure
        """
        if not self.visual_debugging_enabled or not self.visual_debug_config:
            return
            
        try:
            # Configure executor debug settings
            if 'log_level' in self.visual_debug_config:
                log_level = self.visual_debug_config['log_level'].upper()
                if hasattr(logging, log_level):
                    executor.set_log_level(getattr(logging, log_level))
                    logger.debug(f"Set executor log level to: {log_level}")
            
            # Enable step timing if requested
            if self.visual_debug_config.get('enable_timing', False):
                if hasattr(executor, 'enable_step_timing'):
                    executor.enable_step_timing()
                    logger.debug("Enabled step timing for executor")
            
            # Configure any additional debug options
            if 'options' in self.visual_debug_config:
                for option, value in self.visual_debug_config['options'].items():
                    if hasattr(executor, f'set_{option}'):
                        getattr(executor, f'set_{option}')(value)
                        logger.debug(f"Set executor option: {option} = {value}")
            
            logger.info("Visual debugging configured for pipeline execution")
            
        except Exception as e:
            logger.error(f"Error setting up visual debugging: {e}", exc_info=True)
    
    def _generate_debug_output(self, executor: StatefulExecutor, pipeline_name: str) -> None:
        """
        Generate visual debug output after pipeline execution.
        
        Args:
            executor: The StatefulExecutor instance
            pipeline_name: Name of the pipeline being executed
        """
        if not self.visual_debugging_enabled or not self.visual_debug_config:
            logger.debug("Visual debugging not enabled, skipping debug output generation")
            return
            
        try:
            # Ensure output directory exists
            output_dir = os.path.abspath(self.visual_debug_config.get('output_dir', 'debug_output'))
            os.makedirs(output_dir, exist_ok=True)
            logger.debug(f"Saving debug output to: {output_dir}")
            
            # Get execution state from executor
            execution_state = executor.get_pipeline_execution_state()
            if not execution_state:
                logger.warning("No execution state available for debug output")
                return
            
            # Generate timestamp for filenames
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            
            # Save raw execution state as JSON
            debug_file = os.path.join(output_dir, f"{pipeline_name}_debug_{timestamp}.json")
            try:
                with open(debug_file, 'w', encoding='utf-8') as f:
                    json.dump(execution_state, f, indent=2, default=str)
                logger.info(f"Saved debug output to: {debug_file}")
            except (IOError, TypeError) as e:
                logger.error(f"Failed to save debug output: {e}")
            
            # Generate ASCII tree visualization if enabled
            if self.visual_debug_config.get('ascii_tree', True):
                try:
                    visualizer = PipelineAsciiTreeVisualizer()
                    if execution_state and 'root' in execution_state:
                        tree_output = visualizer.render(execution_state['root'])
                        if tree_output:
                            tree_file = os.path.join(output_dir, f"{pipeline_name}_tree_{timestamp}.txt")
                            with open(tree_file, 'w', encoding='utf-8') as f:
                                f.write(tree_output)
                            logger.info(f"Saved ASCII tree visualization to: {tree_file}")
                except Exception as e:
                    logger.error(f"Error generating ASCII tree: {e}", exc_info=True)
                    
        except Exception as e:
            logger.error(f"Unexpected error in _generate_debug_output: {e}", exc_info=True)
            
    def stop(self):
        """Stop the scheduler loop."""
        self.running = False


def execute_step(step, context, plugin_manager, return_result=False, context_service=None):
    """Execute a single pipeline step using the correct plugin lookup logic. Optionally return the result."""
    logger = logging.getLogger(__name__)
    plugin_ref = step.get('plugin')
    step_name = step.get('name', 'Unnamed')
    
    logger.info(f"Executing step: {step_name} with plugin: {plugin_ref}")
    
    if not plugin_ref:
        logger.error(f"No plugin specified for step: {step_name}")
        return None if return_result else None

    # Split plugin_ref like 'Output.HTMLReport' or 'Data_Processing.Delay'
    if '.' in plugin_ref:
        plugin_type, plugin_name = plugin_ref.split('.', 1)
    else:
        plugin_type, plugin_name = None, plugin_ref            # Try to get plugin by type and name
    plugin_instance = None
    if plugin_type:
        plugins_of_type = plugin_manager.get_plugins(plugin_type)
        if plugins_of_type and plugin_name in plugins_of_type:
            plugin_instance = plugins_of_type[plugin_name]
            if hasattr(plugin_instance, 'plugin_manager'):
                plugin_instance.plugin_manager = plugin_manager  # Ensure plugin has access to plugin manager
        else:
            logger.error(f"Plugin {plugin_ref} not found in plugin type {plugin_type}")
            return None if return_result else None
    else:
        # Try all plugins if type not specified
        for type_name, plugins_of_type in plugin_manager.get_plugins().items():
            if plugin_name in plugins_of_type:
                plugin_instance = plugins_of_type[plugin_name]
                break
        if not plugin_instance:
            logger.error(f"Plugin {plugin_ref} not found in any plugin type")
            return None if return_result else None

    # Execute the pipeline step
    try:
        logger.info(f"Calling execute_pipeline_step on {plugin_ref} for step: {step_name}")
        result = plugin_instance.execute_pipeline_step(step, context)
        logger.info(f"Step {step_name} completed successfully")
        # Defensive logging: log type and sample of every value added to context
        if result:
            for k, v in result.items():
                logger.info(f"[ContextUpdate] Key: {k}, Type: {type(v)}, Sample: {str(v)[:300]}")
                # Use context_service to set context, allowing for schema validation
                if context_service:
                    # Determine schema_id for output validation if available from plugin
                    plugin_output_schema_id = f"{plugin_type}.{plugin_name}_output"
                    try:
                        context_service.set_context(
                            namespace="pipeline_context", # Assuming a default namespace for pipeline context
                            key=k,
                            value=v,
                            overwrite=True, # Always overwrite for step outputs
                            schema_id=plugin_output_schema_id # Pass schema_id for validation
                        )
                    except Exception as validation_e:
                        logger.error(f"Context validation failed for key '{k}' from plugin '{plugin_ref}': {validation_e}")
                        # Depending on policy, you might want to re-raise or just log and continue
                        # For now, we'll log and continue to avoid breaking existing pipelines
                        pass
                else:
                    # Fallback to direct context update if no context_service is provided
                    context[k] = v
        return result if return_result else None
    except Exception as e:
        logger.error(f"Error executing step {step.get('name', 'Unnamed')}: {e}")
        return None if return_result else None


def run_scheduled_pipelines(config, plugin_manager, output_dir, context_service=None):
    """Run scheduled pipelines, patched to avoid IndexError on empty schedules."""
    logger = logging.getLogger(__name__)
    pipeline_parser = PipelineParser() # Instantiate parser for scheduled runs

    pipelines = config.get("pipelines", [])
    schedules = [(p, p.get("schedule")) for p in pipelines if p.get("enabled", False) and p.get("schedule")]
    if not schedules:
        logger.info("No scheduled pipelines to run. Exiting scheduler loop.")
        return
    
    scheduled_pipelines_info = []
    for pipeline_config, sched_expr in schedules:
        try:
            sched = CronSchedule(sched_expr)
            # Load and parse the pipeline definition for scheduled pipelines
            pipeline_file = pipeline_config.get("file")
            if not pipeline_file:
                logger.error(f"No file specified for scheduled pipeline: {pipeline_config.get('name', 'Unnamed')}")
                continue
            project_root = os.path.dirname(os.path.abspath(__file__))
            pipeline_file_path = os.path.join(project_root, pipeline_file) if not os.path.isabs(pipeline_file) else pipeline_file
            
            with open(pipeline_file_path, "r") as f:
                yaml_content = f.read()
            
            pipeline_def = pipeline_parser.parse(yaml_content)
            if pipeline_def is None:
                logger.error(f"Failed to parse scheduled pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                continue
            
            if not pipeline_parser.validate(pipeline_def):
                logger.error(f"Validation failed for scheduled pipeline file {pipeline_file_path}. Errors: {pipeline_parser.get_errors()}")
                continue
            
            # Assuming a single pipeline definition per file for simplicity in scheduled context
            # If multiple pipelines are in one file, this needs adjustment.
            pipeline_dict = pipeline_def.get("pipelines", [])[0] if pipeline_def.get("pipelines") else None
            if not pipeline_dict:
                logger.error(f"No pipeline definition found in {pipeline_file_path}")
                continue

            pipeline_ast = pipeline_parser.to_ast({"pipelines": [pipeline_dict]})
            if pipeline_ast is None:
                logger.error(f"Failed to convert scheduled pipeline '{pipeline_dict.get('name', 'Unnamed')}' to AST.")
                continue

            control_graph = ControlGraph(pipeline_ast)
            cycles = control_graph.detect_cycles()
            if cycles:
                logger.error(f"Cycles detected in scheduled pipeline '{pipeline_config.get('name', 'Unnamed')}': {cycles}")
                continue
            graph_errors = control_graph.get_errors()
            if graph_errors:
                for error in graph_errors:
                    logger.error(f"Control Graph error for scheduled pipeline '{pipeline_config.get('name', 'Unnamed')}': {error}")
                continue

            scheduled_pipelines_info.append((pipeline_config, sched, pipeline_dict, pipeline_ast))
        except Exception as e:
            logger.error(f"Error setting up scheduled pipeline {pipeline_config.get('name')}: {e}")

    if not scheduled_pipelines_info:
        logger.info("No valid scheduled pipelines to run. Exiting scheduler loop.")
        return

    """
    Scheduler loop for scheduled pipelines
    """
    def scheduler_loop():
        logger.info("[SCHEDULER] Starting scheduler loop for pipelines...")
        next_runs = []
        for pipeline_config, sched, pipeline_dict, pipeline_ast in scheduled_pipelines_info:
            next_run = sched.next_run()
            next_runs.append((next_run, pipeline_config, sched, pipeline_dict, pipeline_ast))
            logger.info(f"[SCHEDULER] Pipeline '{pipeline_config.get('name')}' scheduled for {next_run}")
        
        while True:
            now = datetime.datetime.now().replace(second=0, microsecond=0)
            # Find the soonest next run
            next_runs.sort()
            soonest, pipeline_config, sched, pipeline_dict, pipeline_ast = next_runs[0]
            
            sleep_secs = (soonest - now).total_seconds()
            if sleep_secs > 0:
                logger.info(f"[SCHEDULER] Sleeping for {sleep_secs:.1f} seconds until next pipeline: {pipeline_config.get('name')}")
                time.sleep(min(sleep_secs, 60))  # Sleep in chunks in case of signal
                continue
            
            # Time to run the pipeline
            logger.info(f"[SCHEDULER] Running scheduled pipeline: {pipeline_config.get('name')} at {now}")
            try:
                # Pass the parsed AST to execute_pipeline
                execute_pipeline(pipeline_dict, pipeline_ast, plugin_manager, output_dir, context_service=context_service, run_count=1, next_run_time=now)
            except Exception as e:
                logger.error(f"Error executing scheduled pipeline {pipeline_config.get('name', 'Unnamed')}: {e}")
            
            # Calculate next run time and update in the list
            next_run = sched.next_run(now)
            next_runs[0] = (next_run, pipeline_config, sched, pipeline_dict, pipeline_ast)
            logger.info(f"[SCHEDULER] Pipeline '{pipeline_config.get('name')}' next run scheduled for {next_run}")
    # Check if we should run in scheduler mode or single-run mode
    if config.get("settings", {}).get("enable_scheduler", False) or any(p.get('execution_mode') == 'continuous' for p in pipelines):
        # Run scheduler in the main thread for better signal handling
        logger.info("Starting scheduler in main thread...")
        try:
            scheduler_loop()
        except KeyboardInterrupt:
            logger.info("Scheduler interrupted. Shutting down...")
        except Exception as e:
            logger.error(f"Unexpected error in scheduler: {e}", exc_info=True)
    else:
        # Single run mode - just run once and exit
        logger.info("No continuous or scheduled pipelines. Running once and exiting.")
        try:
            scheduler_loop()
        except Exception as e:
            logger.error(f"Error during pipeline execution: {e}", exc_info=True)
            return


if __name__ == "__main__":
    print(MIMIR_HEADER)
    
    parser = argparse.ArgumentParser(
        description="Mimir-AIP Pipeline Runner",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""Examples:
  Run specific pipeline: python main.py --pipeline my_pipeline
  List available pipelines: python main.py --list
  Run in verbose mode: python main.py --verbose
""")
    parser.add_argument('--pipeline', type=str,
                      help='Name of pipeline to run once (manual trigger)')
    parser.add_argument('--list', action='store_true',
                      help='List all available pipelines')
    parser.add_argument('--verbose', action='store_true',
                      help='Enable verbose output (shows detailed execution info)')
    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
        print("\n🔍 Verbose mode enabled - showing detailed output\n")

    if args.list:
        try:
            with open("config.yaml", "r") as f:
                config = yaml.safe_load(f)
            print("\n📋 Available Pipelines:")
            for pipeline in config.get("pipelines", []):
                status = "🟢" if pipeline.get("enabled", False) else "🔴"
                print(f"  {status} {pipeline.get('name', 'Unnamed')}")
                if args.verbose:
                    print(f"    File: {pipeline.get('file', 'Not specified')}")
                    print(f"    Description: {pipeline.get('description', 'None')}\n")
            exit(0)
        except Exception as e:
            print(f"❌ Error listing pipelines: {e}")
            exit(1)

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
    print(f"[DEBUG main.py] Config loaded and passed to PluginManager: {config}")
    context_service = ContextService()
    plugin_manager = PluginManager(config=config)

    if args.pipeline:
        # Manual trigger: run the specified pipeline once
        pipelines = config.get('pipelines', [])
        selected = next((p for p in pipelines if p.get('name') == args.pipeline), None)
        if not selected:
            print(f"Pipeline '{args.pipeline}' not found in config.yaml.")
            exit(1)
        # Load the pipeline YAML using the parser
        pipeline_file = selected.get('file')
        if not pipeline_file:
            print(f"No file specified for pipeline '{args.pipeline}'.")
            exit(1)
        pipeline_path = os.path.join(pipeline_dir, os.path.basename(pipeline_file))
        try:
            with open(pipeline_path, 'r') as pf:
                yaml_content = pf.read()
            
            pipeline_parser = PipelineParser() # Instantiate parser for this scope
            pipeline_yaml = pipeline_parser.parse(yaml_content)
            if pipeline_yaml is None:
                print(f"Failed to parse pipeline file {pipeline_path}. Errors: {pipeline_parser.get_errors()}")
                exit(1)
            
            if not pipeline_parser.validate(pipeline_yaml):
                print(f"Validation failed for pipeline file {pipeline_path}. Errors: {pipeline_parser.get_errors()}")
                exit(1)

        except Exception as e:
            print(f"Error loading or parsing pipeline YAML: {e}")
            exit(1)
        # Find the actual pipeline definition (list under 'pipelines')
        pipeline_defs = pipeline_yaml.get('pipelines', [])
        if not pipeline_defs:
            print(f"No pipelines found in {pipeline_file}.")
            exit(1)
        # Run the first (or only) pipeline in the file
        # Convert to AST for execution
        pipeline_parser = PipelineParser()
        pipeline_ast = pipeline_parser.to_ast({"pipelines": pipeline_defs})
        if pipeline_ast:
            execute_pipeline(pipeline_defs[0], pipeline_ast, plugin_manager, output_dir, context_service=context_service)
        else:
            logger.error(f"Failed to convert pipeline to AST: {pipeline_defs[0].get('name', 'Unnamed')}")
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
                yaml_content = pf.read()
            
            pipeline_parser = PipelineParser() # Instantiate parser for this scope
            pipeline_yaml = pipeline_parser.parse(yaml_content)
            if pipeline_yaml is None:
                print(f"Failed to parse pipeline file {pipeline_path}. Errors: {pipeline_parser.get_errors()}")
                exit(1)
            
            if not pipeline_parser.validate(pipeline_yaml):
                print(f"Validation failed for pipeline file {pipeline_path}. Errors: {pipeline_parser.get_errors()}")
                exit(1)

        except Exception as e:
            print(f"Error loading or parsing pipeline YAML: {e}")
            exit(1)
        pipeline_defs = pipeline_yaml.get('pipelines', [])
        if not pipeline_defs:
            print(f"No pipelines found in {pipeline_file}.")
            exit(1)
        # Convert to AST for execution
        pipeline_parser = PipelineParser()
        pipeline_ast = pipeline_parser.to_ast({"pipelines": pipeline_defs})
        if pipeline_ast:
            execute_pipeline(pipeline_defs[0], pipeline_ast, plugin_manager, output_dir, context_service=context_service)
        else:
            logger.error(f"Failed to convert pipeline to AST: {pipeline_defs[0].get('name', 'Unnamed')}")
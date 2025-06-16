import logging
import ast
from datetime import datetime # Import datetime for timing
from typing import Any, Dict, List, Optional, Tuple

from .ast_nodes import RootNode, PipelineNode, StepNode, ConfigNode
from Plugins.PluginManager import PluginManager # Assuming PluginManager is accessible
from ..ContextService import ContextService # Added for direct service calls

logger = logging.getLogger(__name__)

class StatefulExecutor:
    SERVICE_CALL_PREFIX = "SERVICE_CALL."
    """
    Executes a pipeline based on its Abstract Syntax Tree (AST), managing the execution state.
    Supports step pointer tracking, loop stack management, conditional evaluation,
    break/continue handling, and safe expression parsing.
    """

    def __init__(self, ast: RootNode, plugin_manager: PluginManager, context_service: ContextService, initial_context: Dict[str, Any]):
        """
        Initializes the StatefulExecutor.

        Args:
            ast (RootNode): The root node of the pipeline's Abstract Syntax Tree.
            plugin_manager (PluginManager): An instance of the PluginManager to execute steps.
            initial_context (Dict[str, Any]): The initial context for pipeline execution.
        """
        self.ast = ast
        self.plugin_manager = plugin_manager
        self.context_service = context_service # Added
        self.context = initial_context
        self._current_pipeline_node: Optional[PipelineNode] = None
        self._current_step_index: int = 0
        self._loop_stack: List[Dict[str, Any]] = []  # Stores state for nested loops
        self.errors: List[str] = []
        self._break_flag: bool = False
        self._continue_flag: bool = False
        self._execution_state: Dict[str, Any] = {} # New: Stores the current execution state for visualization

    def execute_pipeline(self, pipeline_name: str) -> bool:
        """
        Executes a specific pipeline by name from the loaded AST.

        Args:
            pipeline_name (str): The name of the pipeline to execute.

        Returns:
            bool: True if the pipeline executed successfully, False otherwise.
        """
        self._current_pipeline_node = next(
            (p for p in self.ast.pipelines if p.name == pipeline_name), None
        )

        if not self._current_pipeline_node:
            self.errors.append(f"Pipeline '{pipeline_name}' not found in AST.")
            logger.error(f"Pipeline '{pipeline_name}' not found in AST.")
            return False

        logger.info(f"Starting execution of pipeline: {self._current_pipeline_node.name}")
        self._current_step_index = 0
        self._loop_stack = [] # Reset loop stack for new pipeline execution
        self._initialize_execution_state() # Initialize the execution state

        while self._current_step_index < len(self._current_pipeline_node.steps):
            step_node = self._current_pipeline_node.steps[self._current_step_index]

            try:
                # Handle break/continue flags from nested steps
                if self._break_flag:
                    self._break_flag = False
                    if self._loop_stack:
                        # Jump to end of current loop
                        self._current_step_index = self._loop_stack[-1]["original_step_index"]
                        self._loop_stack.pop()
                    continue

                if self._continue_flag:
                    self._continue_flag = False
                    if self._loop_stack:
                        # Jump to next iteration
                        self._process_current_iteration()
                    continue

                if step_node.iterate:
                    self._handle_iteration_step(step_node)
                else:
                    self._execute_step_by_type(step_node)

                self._current_step_index += 1

            except Exception as e:
                self.errors.append(f"Error executing step '{step_node.name}': {e}")
                logger.error(f"Error executing step '{step_node.name}': {e}")
                return False # Halt pipeline on first error

        logger.info(f"Pipeline '{pipeline_name}' execution completed.")
        return True

    def _execute_step_by_type(self, step_node: StepNode, current_context: Optional[Dict[str, Any]] = None):
        """
        Executes a step based on its type, dispatching to the appropriate handler.

        Args:
            step_node: The step node to execute
            current_context: Optional context override for nested steps
        """
        if step_node.type == "plugin":
            self._execute_plugin_step(step_node, current_context)
        elif step_node.type == "conditional":
            self._execute_conditional_step(step_node, current_context)
        elif step_node.type == "jump":
            self._execute_jump_step(step_node)
        elif step_node.type == "set_context":
            self._execute_set_context(step_node)
        elif step_node.type == "load_context":
            self._execute_load_context(step_node)
        elif step_node.type == "append_context":
            self._execute_append_context(step_node)
        elif step_node.type == "save_context":
            self._execute_save_context(step_node)
        elif step_node.type == "context_operation":
            self._execute_context_operation(step_node, current_context)
        else:
            raise ValueError(f"Unknown step type: {step_node.type}")

    def _execute_set_context(self, step_node: StepNode) -> None:
        """
        Executes a set_context step by setting a value in the context service.

        Args:
            step_node: The set_context step node
        """
        logger.info(f"Executing set_context step: {step_node.name}")
        self.context_service.set_value(
            context=self.context,
            path=step_node.path,
            value=step_node.value,
            overwrite=step_node.overwrite if step_node.overwrite is not None else True,
            actor="pipeline"
        )

    def _execute_load_context(self, step_node: StepNode) -> None:
        """
        Executes a load_context step by loading data into the context.

        Args:
            step_node: The load_context step node
        """
        logger.info(f"Executing load_context step: {step_node.name}")
        loaded_data = self.context_service.load_context(
            source_type=step_node.source,
            path=step_node.path,
            config=step_node.config.data if step_node.config else {},
            actor="pipeline"
        )
        # Store the loaded data in context if not already handled by load_context
        if loaded_data is not None:
            self.context_service.set_value(
                context=self.context,
                path=step_node.path,
                value=loaded_data,
                overwrite=True,
                actor="pipeline"
            )

    def _execute_append_context(self, step_node: StepNode) -> None:
        """
        Executes an append_context step by appending to a context list.

        Args:
            step_node: The append_context step node
        """
        logger.info(f"Executing append_context step: {step_node.name}")
        self.context_service.append_value(
            context=self.context,
            path=step_node.path,
            value=step_node.value,
            create_if_missing=step_node.create_if_missing if step_node.create_if_missing is not None else False,
            actor="pipeline"
        )

    def _execute_save_context(self, step_node: StepNode) -> None:
        """
        Executes a save_context step by saving context data to a destination.

        Args:
            step_node: The save_context step node
        """
        logger.info(f"Executing save_context step: {step_node.name}")
        self.context_service.save_context(
            destination_type=step_node.destination,
            path=step_node.path,
            data=self.context_service.get_value(
                context=self.context,
                path=step_node.path,
                actor="pipeline"
            ),
            config=step_node.config.data if step_node.config else {},
            actor="pipeline"
        )
        """
        Dispatches step execution based on the step's type.
        """
        context_to_use = current_context if current_context is not None else self.context
        self._update_step_state(step_node.name, 'running', start_time=datetime.now())

        try:
            if step_node.type == "plugin":
                self._execute_plugin_step(step_node, context_to_use)
            elif step_node.type == "set_context":
                self._execute_set_context(step_node, context_to_use)
            elif step_node.type == "load_context":
                self._execute_load_context(step_node, context_to_use)
            elif step_node.type == "append_context":
                self._execute_append_context(step_node, context_to_use)
            elif step_node.type == "save_context":
                self._execute_save_context(step_node, context_to_use)
            else:
                raise ValueError(f"Unsupported step type: {step_node.type}")
            
            self._update_step_state(step_node.name, 'completed', end_time=datetime.now())

        except Exception as e:
            self._update_step_state(step_node.name, 'failed', end_time=datetime.now(), error=str(e))
            logger.error(f"Exception in _execute_step_by_type for step '{step_node.name}': {e}", exc_info=True)
            raise

    def _execute_plugin_step(self, step_node: StepNode, context_to_use: Dict[str, Any]):
        """
        Executes a plugin step.
        """
        logger.info(f"Executing plugin step: {step_node.name} (Plugin: {step_node.plugin})")

        plugin_ref = step_node.plugin
        step_config = step_node.config.data if step_node.config else {}

        # Handle plugin execution (existing logic)
        plugin_type, plugin_name = None, plugin_ref
        if '.' in plugin_ref:
            plugin_type, plugin_name = plugin_ref.split('.', 1)

        plugin_instance = None
        if plugin_type:
            plugin_instance = self.plugin_manager.get_plugin(plugin_type, plugin_name)
        else:
            for p_type, p_dict in self.plugin_manager.get_all_plugins().items():
                if plugin_name in p_dict:
                    plugin_instance = p_dict[plugin_name]
                    break

        if not plugin_instance:
            raise ValueError(f"Plugin '{plugin_ref}' not found.")

        if step_node.use_plugin_manager and hasattr(plugin_instance, 'plugin_manager'):
            plugin_instance.plugin_manager = self.plugin_manager

        mock_step_dict = {
            "name": step_node.name,
            "plugin": step_node.plugin,
            "config": step_config,
            "input": step_node.input,
            "output": step_node.output,
            "iterate": step_node.iterate,
            "use_plugin_manager": step_node.use_plugin_manager,
            "steps": [self._step_node_to_dict(s) for s in step_node.steps]
        }

        result = plugin_instance.execute_pipeline_step(mock_step_dict, context_to_use)

        if result:
            for k, v in result.items():
                context_to_use[k] = v
                logger.info(f"[ContextUpdate] Key: {k} from plugin '{plugin_ref}', Type: {type(v)}, Sample: {str(v)[:100]}")

    def _execute_set_context(self, step_node: StepNode, context_to_use: Dict[str, Any]):
        """
        Executes a set_context step, setting a value in the context.
        """
        logger.info(f"Executing set_context step: {step_node.name} (Path: {step_node.path})")
        if not step_node.path:
            raise ValueError("set_context step requires a 'path' parameter.")
        
        # Evaluate the value if it's an expression
        value_to_set = self._evaluate_expression(str(step_node.value)) if isinstance(step_node.value, str) and step_node.value.startswith("ctx.") else step_node.value

        try:
            self.context_service.set_value(
                context=context_to_use,
                path=step_node.path,
                value=value_to_set,
                overwrite=step_node.overwrite
            )
            logger.info(f"[ContextUpdate] Set context path '{step_node.path}' with value. Sample: {str(value_to_set)[:100]}")
        except Exception as e:
            raise RuntimeError(f"Failed to set context for path '{step_node.path}': {e}") from e

    def _execute_load_context(self, step_node: StepNode, context_to_use: Dict[str, Any]):
        """
        Executes a load_context step, loading data into the context.
        """
        logger.info(f"Executing load_context step: {step_node.name} (Path: {step_node.path}, Source: {step_node.source})")
        if not step_node.path or not step_node.source:
            raise ValueError("load_context step requires 'path' and 'source' parameters.")
        
        config = step_node.config.data if step_node.config else {}

        try:
            loaded_data = self.context_service.load_context(
                source_type=step_node.source,
                path=step_node.path,
                config=config
            )
            if loaded_data is not None:
                # Assuming load_context returns the data to be placed at the path
                self.context_service.set_value(context_to_use, step_node.path, loaded_data, overwrite=True)
                logger.info(f"[ContextUpdate] Loaded context from '{step_node.source}' to path '{step_node.path}'. Sample: {str(loaded_data)[:100]}")
            else:
                logger.warning(f"load_context from '{step_node.source}' to path '{step_node.path}' returned no data.")
        except Exception as e:
            raise RuntimeError(f"Failed to load context for path '{step_node.path}' from source '{step_node.source}': {e}") from e

    def _execute_append_context(self, step_node: StepNode, context_to_use: Dict[str, Any]):
        """
        Executes an append_context step, appending a value to a list in the context.
        """
        logger.info(f"Executing append_context step: {step_node.name} (Path: {step_node.path})")
        if not step_node.path:
            raise ValueError("append_context step requires a 'path' parameter.")
        
        # Evaluate the value if it's an expression
        value_to_append = self._evaluate_expression(str(step_node.value)) if isinstance(step_node.value, str) and step_node.value.startswith("ctx.") else step_node.value

        try:
            self.context_service.append_value(
                context=context_to_use,
                path=step_node.path,
                value=value_to_append,
                create_if_missing=step_node.create_if_missing
            )
            logger.info(f"[ContextUpdate] Appended value to context path '{step_node.path}'. Sample: {str(value_to_append)[:100]}")
        except Exception as e:
            raise RuntimeError(f"Failed to append to context for path '{step_node.path}': {e}") from e

    def _execute_save_context(self, step_node: StepNode, context_to_use: Dict[str, Any]):
        """
        Executes a save_context step, saving context data to a destination.
        """
        logger.info(f"Executing save_context step: {step_node.name} (Path: {step_node.path}, Destination: {step_node.destination})")
        if not step_node.path or not step_node.destination:
            raise ValueError("save_context step requires 'path' and 'destination' parameters.")
        
        config = step_node.config.data if step_node.config else {}

        try:
            # Retrieve the value to save from the context
            value_to_save = self.context_service.get_value(context_to_use, step_node.path)
            
            self.context_service.save_context(
                destination_type=step_node.destination,
                path=step_node.path, # This path is used for naming/structuring the saved data
                data=value_to_save,
                config=config
            )
            logger.info(f"[ContextUpdate] Saved context from path '{step_node.path}' to destination '{step_node.destination}'. Sample: {str(value_to_save)[:100]}")
        except Exception as e:
            raise RuntimeError(f"Failed to save context from path '{step_node.path}' to destination '{step_node.destination}': {e}") from e

    def _handle_iteration_step(self, step_node: StepNode):
        """
        Handles steps with 'iterate' property, pushing loop state onto the stack.
        """
        logger.info(f"Handling iteration step: {step_node.name} (Iterate: {step_node.iterate})")

        # Evaluate the iteration expression
        try:
            iterable_data = self._evaluate_expression(step_node.iterate)
            if not isinstance(iterable_data, (list, tuple)):
                raise TypeError("Iteration data must be a list or tuple.")
        except Exception as e:
            raise ValueError(f"Failed to evaluate iteration expression '{step_node.iterate}': {e}")

        if not iterable_data:
            logger.info(f"Iteration step '{step_node.name}' has no data to iterate over. Skipping nested steps.")
            return

        # Push loop state onto the stack
        self._loop_stack.append({
            "step_node": step_node,
            "iterable_data": iterable_data,
            "current_iteration_index": 0,
            "original_step_index": self._current_step_index,
            "nested_steps_executed": False # Flag to ensure nested steps are executed at least once if data exists
        })

        # Immediately execute the first iteration
        self._process_current_iteration()

    def _process_current_iteration(self):
        """
        Processes the current iteration of the innermost loop.
        """
        if not self._loop_stack:
            raise RuntimeError("Attempted to process iteration with an empty loop stack.")

        current_loop = self._loop_stack[-1]
        step_node = current_loop["step_node"]
        iterable_data = current_loop["iterable_data"]
        current_iteration_index = current_loop["current_iteration_index"]

        if current_iteration_index < len(iterable_data):
            item = iterable_data[current_iteration_index]
            logger.info(f"Executing iteration {current_iteration_index + 1}/{len(iterable_data)} for step '{step_node.name}'")

            # Create a new context for the iteration, including 'item'
            iteration_context = {**self.context, "item": item}

            # Update iteration status
            self._update_iteration_state(step_node.name, current_iteration_index, 'running')

            # Execute all nested steps for the current iteration
            for nested_step in step_node.steps:
                # Pass the iteration_context to nested steps
                self._execute_single_step(nested_step, iteration_context)

            # Merge changes from iteration_context back to main context if needed
            for k, v in iteration_context.items():
                if k not in self.context or self.context[k] != v:
                    self.context[k] = v
                    logger.debug(f"Merged '{k}' from iteration context to main context.")
            
            # Update iteration status to completed
            self._update_iteration_state(step_node.name, current_iteration_index, 'completed')

            current_loop["current_iteration_index"] += 1
            current_loop["nested_steps_executed"] = True

            # If there are more iterations, we need to "loop back"
            # This implies the main execution loop needs to be aware of the loop stack.
            # For now, this method will just complete one iteration.
            # The main `execute_pipeline` loop will need to call this repeatedly.
        else:
            # All iterations for this loop are complete, pop from stack
            logger.info(f"All iterations for step '{step_node.name}' completed. Popping from loop stack.")
            self._loop_stack.pop()
            # Update the parent step's status to completed if all iterations are done
            self._update_step_state(step_node.name, 'completed')
            # The main loop will then increment _current_step_index to move to the next top-level step.

    def _step_node_to_dict(self, step_node: StepNode) -> Dict[str, Any]:
        """Converts a StepNode back to a dictionary for compatibility with existing plugin interface."""
        d = {
            "name": step_node.name,
            "plugin": step_node.plugin,
        }
        if step_node.config:
            d["config"] = step_node.config.data
        if step_node.input:
            d["input"] = step_node.input
        if step_node.output:
            d["output"] = step_node.output
        if step_node.iterate:
            d["iterate"] = step_node.iterate
        if step_node.use_plugin_manager is not None:
            d["use_plugin_manager"] = step_node.use_plugin_manager
        if step_node.steps:
            d["steps"] = [self._step_node_to_dict(s) for s in step_node.steps]
        return d

    def _evaluate_expression(self, expr: str) -> Any:
        """
        Safely evaluates a pipeline expression using the current context.
        Supports basic operations and context variable access.

        Args:
            expr (str): The expression to evaluate

        Returns:
            Any: The evaluated result

        Raises:
            ValueError: For invalid expressions or unsafe operations
        """
        try:
            # Basic expression parsing with context access
            if "." in expr:
                parts = expr.split(".", 1)
                if parts[0] != "context":
                    raise ValueError(f"Invalid expression root '{parts[0]}' - only 'context' is allowed")
                return self._get_nested_context_value(parts[1])

            # Simple arithmetic expressions
            if expr.isdigit():
                return int(expr)
            if expr.replace(".", "", 1).isdigit():
                return float(expr)
            if expr in ("true", "True"):
                return True
            if expr in ("false", "False"):
                return False

            # Context variable access
            if expr in self.context:
                return self.context[expr]

            # Handle expressions with operators
            operators = ['==', '!=', '<', '<=', '>', '>=', '+', '-', '*', '/', '%']
            for op in operators:
                if op in expr:
                    left, right = expr.split(op, 1)
                    left_val = self._evaluate_expression(left.strip())
                    right_val = self._evaluate_expression(right.strip())

                    # Apply type coercion
                    left_val, right_val = self._coerce_types(left_val, right_val, op)

                    # Perform operation
                    if op == '==': return left_val == right_val
                    if op == '!=': return left_val != right_val
                    if op == '<': return left_val < right_val
                    if op == '<=': return left_val <= right_val
                    if op == '>': return left_val > right_val
                    if op == '>=': return left_val >= right_val
                    if op == '+': return left_val + right_val
                    if op == '-': return left_val - right_val
                    if op == '*': return left_val * right_val
                    if op == '/':
                        if right_val == 0:
                            raise ValueError("Division by zero")
                        return left_val / right_val
                    if op == '%':
                        if right_val == 0:
                            raise ValueError("Modulo by zero")
                        return left_val % right_val

            raise ValueError(f"Unrecognized expression format: {expr}")

        except Exception as e:
            raise ValueError(f"Error evaluating expression '{expr}': {e}") from e

    def _coerce_types(self, left: Any, right: Any, operator: str) -> Tuple[Any, Any]:
        """Attempt to coerce types for safe comparison/operation"""
        # Numeric coercion (int, float, numeric strings)
        if operator in ('<', '<=', '>', '>=', '+', '-', '*', '/', '%'):
            try:
                if isinstance(left, str) and not isinstance(right, str):
                    left = float(left) if '.' in left else int(left)
                elif isinstance(right, str) and not isinstance(left, str):
                    right = float(right) if '.' in right else int(right)
            except (ValueError, TypeError):
                pass

        # Boolean coercion for equality comparisons
        if operator in ('==', '!='):
            if isinstance(left, bool) and not isinstance(right, bool):
                right = str(right).lower() in ('true', '1', 'yes')
            elif isinstance(right, bool) and not isinstance(left, bool):
                left = str(left).lower() in ('true', '1', 'yes')

        # String coercion for addition (concatenation)
        if operator == '+':
            left = str(left)
            right = str(right)

        return left, right

    def _get_nested_context_value(self, path: str) -> Any:
        """
        Gets a nested value from context using dot notation.

        Args:
            path (str): The dot-separated path to the value

        Returns:
            Any: The value at the specified path

        Raises:
            KeyError: If any part of the path is missing
        """
        current = self.context
        for part in path.split("."):
            if isinstance(current, dict):
                current = current[part]
            else:
                current = getattr(current, part)
        return current

    def break_loop(self):
        """Signals to break out of the current loop."""
        self._break_flag = True
        logger.debug("Break signal received")

    def continue_loop(self):
        """Signals to continue to next iteration of current loop."""
        self._continue_flag = True
        logger.debug("Continue signal received")

    def get_errors(self) -> List[str]:
        """Returns the list of errors encountered during execution."""
        return self.errors

    def _initialize_execution_state(self):
        """
        Initializes the _execution_state dictionary based on the pipeline AST.
        This creates a hierarchical structure mirroring the pipeline steps,
        with initial 'pending' statuses.
        """
        if not self._current_pipeline_node:
            return

        def _build_state_node(step_node: StepNode) -> Dict[str, Any]:
            node_state = {
                'name': step_node.name,
                'status': 'pending',
                'start_time': None,
                'end_time': None,
                'error': None,
                'children': []
            }
            if step_node.iterate:
                # For iterative steps, children will be iterations, not sub-steps
                node_state['iterate'] = True
                node_state['iterations'] = {
                    'count': 0,
                    'statuses': [],
                    'labels': []
                }
            elif step_node.steps:
                node_state['children'] = [_build_state_node(s) for s in step_node.steps]
            return node_state

        self._execution_state = _build_state_node(self._current_pipeline_node)
        # The root node of the visualizer expects a 'root' key, and the pipeline itself as its child
        self._execution_state = {
            'root': {
                'name': self._current_pipeline_node.name,
                'status': 'pending',
                'start_time': None,
                'end_time': None,
                'error': None,
                'children': [_build_state_node(s) for s in self._current_pipeline_node.steps]
            }
        }
        logger.debug("Execution state initialized.")

    def _update_step_state(self, step_name: str, status: str,
                           start_time: Optional[datetime] = None,
                           end_time: Optional[datetime] = None,
                           error: Optional[str] = None):
        """
        Updates the status and other details of a specific step in the execution state.
        This method traverses the _execution_state to find and update the correct step.
        """
        def _find_and_update(node: Dict[str, Any], target_name: str):
            if node.get('name') == target_name:
                node['status'] = status
                if start_time:
                    node['start_time'] = start_time
                if end_time:
                    node['end_time'] = end_time
                if error:
                    node['error'] = error
                return True
            for child in node.get('children', []):
                if _find_and_update(child, target_name):
                    return True
            return False

        # Start search from the root of the current pipeline's steps
        if self._execution_state and 'root' in self._execution_state:
            _find_and_update(self._execution_state['root'], step_name)
            logger.debug(f"Updated state for step '{step_name}': status={status}")

    def _update_iteration_state(self, step_name: str, iteration_index: int, status: str,
                                label: Optional[str] = None):
        """
        Updates the status of a specific iteration within an iterative step.
        """
        def _find_and_update_iteration(node: Dict[str, Any], target_name: str):
            if node.get('name') == target_name and node.get('iterate'):
                iterations = node.get('iterations')
                if iterations and iteration_index < iterations['count']:
                    iterations['statuses'][iteration_index] = status
                    if label:
                        iterations['labels'][iteration_index] = label
                    logger.debug(f"Updated iteration {iteration_index} for step '{step_name}': status={status}")
                    return True
            for child in node.get('children', []):
                if _find_and_update_iteration(child, target_name):
                    return True
            return False

        if self._execution_state and 'root' in self._execution_state:
            _find_and_update_iteration(self._execution_state['root'], step_name)

    def get_pipeline_execution_state(self) -> Dict[str, Any]:
        """
        Returns the current execution state of the pipeline for visualization.
        """
        return self._execution_state
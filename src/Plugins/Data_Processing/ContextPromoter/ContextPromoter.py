"""
ContextPromoter module.

Promotes context values from nested or previous contexts into the main pipeline context.
"""

from Plugins.BasePlugin import BasePlugin
import logging
import ast

class ContextPromoter(BasePlugin):
    """Plugin to copy values from nested or previous context into the main pipeline context.

    Supports nested expressions via AST parsing and falls back to flat assignment if needed.
    """
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """Copy value from source expression to target context key.

        Args:
            step_config (dict): Contains 'source' (str: expression or context key) and 'target' (str: expression or context key).
            context (dict): Current pipeline context dictionary.

        Returns:
            dict: Mapping of target key to assigned value.
        """
        logger = logging.getLogger(__name__)
        source = step_config["source"]
        target = step_config["target"]
        value = None
        # Validate source expression before evaluation
        if not isinstance(source, str) or not source.strip():
            logger.warning(f"[ContextPromoter] Invalid source expression: {source}")
            value = context.get(source)
            return {target: value}
            
        try:
            # Parse and validate AST first
            ast.parse(source, mode='eval')
            value = eval(source, {}, context)
            logger.info(f"[ContextPromoter] Evaluated source '{source}' to value: {value}")
        except SyntaxError as e:
            logger.warning(f"[ContextPromoter] Syntax error in source '{source}': {e}")
            value = context.get(source)
        except Exception as e:
            logger.warning(f"[ContextPromoter] Exception evaluating '{source}': {e}. Fallback value: {value}")
            value = context.get(source)
        logger.info(f"[ContextPromoter] Context keys at promotion: {list(context.keys())}")
        # Generic nested assignment logic
        try:
            self._assign_nested(context, target, value, logger)
        except Exception as e:
            logger.warning(f"[ContextPromoter] Failed nested assignment for target '{target}': {e}. Falling back to flat assignment.")
            if value is not None:
                context[target] = value
                logger.info(f"[ContextPromoter] Set context['{target}'] = {value}")
            else:
                logger.info(f"[ContextPromoter] Did not set context['{target}'] because value is None")
        return {target: value}

    def _assign_nested(self, context, target_expr, value, logger):
        """Assign a value to a nested context expression using AST parsing.

        Args:
            context (dict): Pipeline context dictionary.
            target_expr (str): Target assignment expression (e.g., 'foo["bar"][0]')
            value: Value to assign.
            logger (logging.Logger): Logger for diagnostic messages.
        """
        # Parse the target expression
        node = ast.parse(target_expr, mode='eval').body
        # Traverse to the parent object
        obj, final_key = self._resolve_parent(context, node)
        # Assign the value
        if isinstance(obj, dict) and isinstance(final_key, str):
            obj[final_key] = value
            logger.info(f"[ContextPromoter] Set nested dict: ...['{final_key}'] = {value}")
            logger.info(f"[ContextPromoter] Parent object after assignment: {obj}")
        elif isinstance(obj, list) and isinstance(final_key, int):
            obj[final_key] = value
            logger.info(f"[ContextPromoter] Set nested list: ...[{final_key}] = {value}")
            logger.info(f"[ContextPromoter] Parent object after assignment: {obj}")
        else:
            raise ValueError(f"Unsupported assignment target: {type(obj)}, key: {final_key}")

    def _resolve_parent(self, context, node):
        """Resolve parent object and final key/index from AST node for assignment.

        Args:
            context (dict): Pipeline context.
            node (ast.AST): AST node representing the target expression.

        Returns:
            tuple: (parent_obj, final_key/index) for the assignment.
        """
        # If node is a Name, we can't assign to its parent, so raise
        if isinstance(node, ast.Name):
            return context, node.id
        elif isinstance(node, ast.Subscript):
            parent_obj, key = self._resolve_parent(context, node.value)
            # Evaluate the key
            if isinstance(node.slice, ast.Constant):
                idx = node.slice.value
            elif hasattr(ast, 'Index') and isinstance(node.slice, ast.Index):
                idx = node.slice.value.value
            else:
                idx = eval(compile(ast.Expression(node.slice), '<ast>', 'eval'), {}, context)
            return parent_obj[key], idx
        else:
            raise ValueError(f"Unsupported AST node for assignment: {ast.dump(node)}")
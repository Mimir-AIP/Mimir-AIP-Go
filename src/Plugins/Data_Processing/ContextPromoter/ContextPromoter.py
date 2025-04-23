"""
Plugin for promoting a variable from a nested or previous context into the main context.
"""

from Plugins.BasePlugin import BasePlugin
import logging
import ast

class ContextPromoter(BasePlugin):
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Copies the value of a source key or expression to a target key (supports nested assignment) in the main context.
        step_config:
          source: the key or expression to copy from (e.g., 'foo', 'foo["bar"]', or 'foo[0]["bar"]')
          target: the key or nested expression to copy to (e.g., 'foo', 'foo["bar"]', 'foo[0]["bar"]')
        """
        logger = logging.getLogger(__name__)
        source = step_config["source"]
        target = step_config["target"]
        value = None
        try:
            value = eval(source, {}, context)
            logger.info(f"[ContextPromoter] Evaluated source '{source}' to value: {value}")
        except Exception as e:
            value = context.get(source)
            logger.warning(f"[ContextPromoter] Exception evaluating '{source}': {e}. Fallback value: {value}")
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
        """
        Assigns value to the nested target specified by target_expr within context.
        Supports dict and list traversal, e.g., foo['bar'][0]['baz'].
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
        """
        Traverse the AST to get the parent object and final key/index for assignment.
        Returns (parent_obj, final_key).
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

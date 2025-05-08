"""
RuleEngine module.

A generic rule engine for evaluating conditions and applying rules in a systematic way.
"""

from typing import Dict, List, Any, Union, Callable
import operator
import logging
from Plugins.BasePlugin import BasePlugin


class RuleEngine(BasePlugin):
    """
    Generic rule engine plugin for evaluating conditions and applying rules.
    Provides functionality for:
    - Rule definition and management
    - Condition evaluation
    - Action execution
    - Rule chaining
    """

    plugin_type = "Data_Processing"

    def __init__(self):
        """Initialize the RuleEngine plugin"""
        self.rules = {}  # Rule ID -> rule definition
        self.operators = {
            "eq": operator.eq,
            "ne": operator.ne,
            "lt": operator.lt,
            "gt": operator.gt,
            "le": operator.le,
            "ge": operator.ge,
            "in": lambda x, y: x in y,
            "not_in": lambda x, y: x not in y,
            "contains": lambda x, y: y in x,
            "not_contains": lambda x, y: y not in x,
            "matches": lambda x, y: bool(x and y and str(y) in str(x)),
            "starts_with": str.startswith,
            "ends_with": str.endswith
        }
        self.logger = logging.getLogger(__name__)

    def add_rule(self, rule_id: str, conditions: List[Dict], actions: List[Dict], 
                priority: int = 0, enabled: bool = True) -> None:
        """Add or update a rule
        
        Args:
            rule_id: Unique identifier for the rule
            conditions: List of condition dictionaries
            actions: List of action dictionaries
            priority: Rule priority (higher runs first)
            enabled: Whether rule is active
        """
        self.rules[rule_id] = {
            "conditions": conditions,
            "actions": actions,
            "priority": priority,
            "enabled": enabled
        }
        self.logger.info(f"Added/updated rule: {rule_id}")

    def remove_rule(self, rule_id: str) -> None:
        """Remove a rule
        
        Args:
            rule_id: ID of rule to remove
        """
        if rule_id in self.rules:
            del self.rules[rule_id]
            self.logger.info(f"Removed rule: {rule_id}")

    def evaluate_condition(self, condition: Dict[str, Any], facts: Dict[str, Any]) -> bool:
        """Evaluate a single condition against facts
        
        Args:
            condition: Condition dictionary with field, operator, value
            facts: Dictionary of facts to evaluate against
            
        Returns:
            Boolean indicating if condition is met
        """
        field = condition["field"]
        op = condition["operator"]
        expected = condition["value"]
        
        if field not in facts:
            return False
            
        actual = facts[field]
        op_func = self.operators.get(op)
        
        if op_func is None:
            self.logger.warning(f"Unknown operator: {op}")
            return False
            
        try:
            return op_func(actual, expected)
        except Exception as e:
            self.logger.error(f"Error evaluating condition: {str(e)}")
            return False

    def evaluate_conditions(self, conditions: List[Dict[str, Any]], 
                          facts: Dict[str, Any]) -> bool:
        """Evaluate all conditions for a rule
        
        Args:
            conditions: List of condition dictionaries
            facts: Facts to evaluate against
            
        Returns:
            Boolean indicating if all conditions are met
        """
        return all(self.evaluate_condition(cond, facts) for cond in conditions)

    def execute_action(self, action: Dict[str, Any], facts: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a single action
        
        Args:
            action: Action dictionary with type and parameters
            facts: Current facts dictionary
            
        Returns:
            Updated facts dictionary
        """
        action_type = action["type"]
        
        if action_type == "set":
            field = action["field"]
            value = action["value"]
            facts[field] = value
            
        elif action_type == "delete":
            field = action["field"]
            if field in facts:
                del facts[field]
                
        elif action_type == "increment":
            field = action["field"]
            amount = action.get("amount", 1)
            if field in facts:
                facts[field] = facts[field] + amount
                
        elif action_type == "append":
            field = action["field"]
            value = action["value"]
            if field not in facts:
                facts[field] = []
            facts[field].append(value)
            
        else:
            self.logger.warning(f"Unknown action type: {action_type}")
            
        return facts

    def execute_actions(self, actions: List[Dict[str, Any]], 
                       facts: Dict[str, Any]) -> Dict[str, Any]:
        """Execute all actions for a rule
        
        Args:
            actions: List of action dictionaries
            facts: Current facts dictionary
            
        Returns:
            Updated facts dictionary
        """
        for action in actions:
            facts = self.execute_action(action, facts)
        return facts

    def evaluate_rules(self, facts: Dict[str, Any]) -> Dict[str, Any]:
        """Evaluate and execute all applicable rules
        
        Args:
            facts: Dictionary of facts to evaluate against
            
        Returns:
            Updated facts after applying matching rules
        """
        # Sort rules by priority
        sorted_rules = sorted(
            [(rid, r) for rid, r in self.rules.items() if r["enabled"]],
            key=lambda x: x[1]["priority"],
            reverse=True
        )
        
        for rule_id, rule in sorted_rules:
            if self.evaluate_conditions(rule["conditions"], facts):
                self.logger.info(f"Rule matched: {rule_id}")
                facts = self.execute_actions(rule["actions"], facts)
                
        return facts

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for this plugin
        
        Args:
            step_config: Configuration dictionary containing:
                - operation: Type of operation to perform
                    - 'add': Add/update rules
                    - 'remove': Remove rules
                    - 'evaluate': Evaluate rules against facts
                - rules: Rule definitions for add operation
                - rule_ids: Rule IDs for remove operation
                - facts: Facts dictionary for evaluate operation
            context: Pipeline context
            
        Returns:
            Dictionary containing operation results
        """
        config = step_config.get("config", {})
        operation = config.get("operation")
        
        try:
            if operation == "add":
                rules = config.get("rules", [])
                for rule in rules:
                    self.add_rule(
                        rule["id"],
                        rule["conditions"],
                        rule["actions"],
                        rule.get("priority", 0),
                        rule.get("enabled", True)
                    )
                return {step_config["output"]: {"added": len(rules)}}
                
            elif operation == "remove":
                rule_ids = config.get("rule_ids", [])
                for rule_id in rule_ids:
                    self.remove_rule(rule_id)
                return {step_config["output"]: {"removed": len(rule_ids)}}
                
            elif operation == "evaluate":
                facts = config.get("facts", {})
                updated_facts = self.evaluate_rules(facts)
                return {step_config["output"]: updated_facts}
                
            else:
                raise ValueError(f"Unknown operation: {operation}")
                
        except Exception as e:
            raise ValueError(f"Error in RuleEngine plugin: {str(e)}")
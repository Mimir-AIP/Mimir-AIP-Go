import yaml
import jsonschema
import logging
from typing import Dict, Any, List, Optional
from .ast_nodes import RootNode, PipelineNode, StepNode, ConfigNode # Import AST nodes

logger = logging.getLogger(__name__)

class PipelineParser:
    """
    Parses and validates pipeline YAML configurations, converting them into an Abstract Syntax Tree (AST).
    """

    def __init__(self, schema_path: str = "src/pipeline_parser/pipeline_schema.json"):
        """
        Initializes the PipelineParser with a path to the JSON schema for validation.

        Args:
            schema_path (str): The path to the JSON schema file.
        """
        self.schema_path = schema_path
        self.schema = self._load_schema()
        self.errors: List[str] = []

    def _load_schema(self) -> Dict[str, Any]:
        """
        Loads the JSON schema from the specified path.

        Returns:
            Dict[str, Any]: The loaded JSON schema.

        Raises:
            FileNotFoundError: If the schema file does not exist.
            yaml.YAMLError: If there's an error parsing the schema file.
        """
        try:
            with open(self.schema_path, "r") as f:
                return yaml.safe_load(f)
        except FileNotFoundError:
            logger.error(f"Schema file not found: {self.schema_path}")
            raise
        except yaml.YAMLError as e:
            logger.error(f"Error parsing schema file {self.schema_path}: {e}")
            raise

    def parse(self, yaml_content: str) -> Optional[Dict[str, Any]]:
        """
        Parses the given YAML content into a Python dictionary.

        Args:
            yaml_content (str): The YAML content as a string.

        Returns:
            Optional[Dict[str, Any]]: The parsed YAML content as a dictionary, or None if parsing fails.
        """
        self.errors = []  # Clear previous errors
        try:
            parsed_data = yaml.safe_load(yaml_content)
            return parsed_data
        except yaml.YAMLError as e:
            self.errors.append(f"YAML parsing error: {e}")
            logger.error(f"YAML parsing error: {e}")
            return None

    def validate(self, data: Dict[str, Any]) -> bool:
        """
        Validates the parsed pipeline data against the loaded JSON schema.

        Args:
            data (Dict[str, Any]): The parsed pipeline data (Python dictionary).

        Returns:
            bool: True if validation passes, False otherwise.
        """
        self.errors = []  # Clear previous errors
        try:
            jsonschema.validate(instance=data, schema=self.schema)
            return True
        except jsonschema.ValidationError as e:
            self.errors.append(f"Validation error: {e.message} (Path: {e.path})")
            logger.error(f"Validation error: {e.message} (Path: {e.path})")
            return False
        except Exception as e:
            self.errors.append(f"An unexpected error occurred during validation: {e}")
            logger.error(f"An unexpected error occurred during validation: {e}")
            return False

    def to_ast(self, data: Dict[str, Any]) -> RootNode:
        """
        Converts the validated pipeline data into an Abstract Syntax Tree (AST).

        Args:
            data (Dict[str, Any]): The validated pipeline data.

        Returns:
            RootNode: The AST representation of the pipeline.
        """
        logger.info("Converting validated data to AST.")
        pipelines_ast: List[PipelineNode] = []

        for pipeline_data in data.get("pipelines", []):
            steps_ast: List[StepNode] = []
            for step_data in pipeline_data.get("steps", []):
                steps_ast.append(self._create_step_node(step_data))
            
            pipelines_ast.append(
                PipelineNode(
                    name=pipeline_data["name"],
                    description=pipeline_data["description"],
                    steps=steps_ast,
                    version=pipeline_data.get("version"),
                    enabled=pipeline_data.get("enabled", False),
                    execution_mode=pipeline_data.get("execution_mode", "single"),
                    plugin_manager_required=pipeline_data.get("plugin_manager_required"),
                    schedule=pipeline_data.get("schedule")
                )
            )
        return RootNode(pipelines=pipelines_ast)

    def _create_step_node(self, step_data: Dict[str, Any]) -> StepNode:
        """Helper to create a StepNode, handling nested steps and control flow constructs."""
        step_type = step_data.get("type", "plugin")
        nested_steps_ast: List[StepNode] = []

        if "steps" in step_data:
            for nested_step_data in step_data["steps"]:
                nested_steps_ast.append(self._create_step_node(nested_step_data))

        # Handle different step types
        if step_type == "conditional":
            # Conditional step
            return StepNode(
                name=step_data["name"],
                type="conditional",
                condition=step_data.get("condition"),
                steps=nested_steps_ast
            )
        elif step_type == "jump":
            # Jump step
            return StepNode(
                name=step_data["name"],
                type="jump",
                to=step_data["to"],
                steps=nested_steps_ast
            )
        elif step_type == "set_context":
            return StepNode(
                name=step_data["name"],
                type="set_context",
                path=step_data["path"],
                value=step_data["value"],
                overwrite=step_data.get("overwrite", True),
                steps=nested_steps_ast
            )
        elif step_type == "load_context":
            return StepNode(
                name=step_data["name"],
                type="load_context",
                path=step_data["path"],
                source=step_data["source"],
                config=ConfigNode(step_data["config"]) if "config" in step_data else None,
                steps=nested_steps_ast
            )
        elif step_type == "append_context":
            return StepNode(
                name=step_data["name"],
                type="append_context",
                path=step_data["path"],
                value=step_data["value"],
                create_if_missing=step_data.get("create_if_missing", False),
                steps=nested_steps_ast
            )
        elif step_type == "save_context":
            return StepNode(
                name=step_data["name"],
                type="save_context",
                path=step_data["path"],
                destination=step_data["destination"],
                config=ConfigNode(step_data["config"]) if "config" in step_data else None,
                steps=nested_steps_ast
            )
        elif step_type == "context_operation":
            # New context operation step
            return StepNode(
                name=step_data["name"],
                type="context_operation",
                service_name=step_data.get("service_name", "context_service"),
                method_name=step_data["method_name"],
                method_args=step_data["method_args"],
                steps=nested_steps_ast
            )
        else:
            # Regular plugin step (default)
            # Handle extended iterate field
            iterate_value = step_data.get("iterate")
            if isinstance(iterate_value, dict):
                # Convert iterate object to string for backward compatibility
                iterate_value = {
                    "items": iterate_value.get("items"),
                    "as": iterate_value.get("as", "item"),
                    "index": iterate_value.get("index")
                }

            return StepNode(
                name=step_data["name"],
                type="plugin",
                plugin=step_data["plugin"],
                config=ConfigNode(step_data["config"]) if "config" in step_data else None,
                input=step_data.get("input"),
                output=step_data.get("output"),
                label=step_data.get("label"),
                iterate=iterate_value,
                use_plugin_manager=step_data.get("use_plugin_manager"),
                steps=nested_steps_ast
            )

    def get_errors(self) -> List[str]:
        """
        Returns a list of errors encountered during parsing or validation.

        Returns:
            List[str]: A list of error messages.
        """
        return self.errors
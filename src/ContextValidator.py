"""
ContextValidator module.

Provides a service for validating pipeline context data against JSON schemas.
"""

import logging
from typing import Any, Dict, Optional, Union
from jsonschema import validate, ValidationError, SchemaStore, RefResolver

class ContextValidationError(Exception):
    """Custom exception for context validation errors."""
    pass

class ContextValidator:
    """
    Service for validating context data using JSON schemas.

    This class manages a collection of JSON schemas and provides methods
    to validate context data against these schemas. It supports schema
    referencing and custom schema definitions.

    Attributes:
        logger: Logger instance for logging validation operations and errors.
        _schema_store: A jsonschema.SchemaStore instance to manage and resolve schemas.
        _resolver: A jsonschema.RefResolver for resolving JSON references within schemas.
    """

    def __init__(self) -> None:
        """
        Initializes the ContextValidator with an empty schema store.
        """
        self.logger = logging.getLogger(__name__)
        self._schema_store = SchemaStore()
        # Initialize RefResolver with a base URI (can be a dummy one if not loading from external URIs)
        self._resolver = RefResolver.from_schema({}, store=self._schema_store)
        self.logger.info("ContextValidator initialized.")

    def register_schema(self, schema_id: str, schema: Dict[str, Any]) -> None:
        """
        Registers a JSON schema with a given ID.

        Args:
            schema_id: A unique identifier for the schema (e.g., "my_plugin_input_schema").
            schema: The JSON schema dictionary.

        Raises:
            ValueError: If the schema_id is empty or schema is not a dictionary.
        """
        if not schema_id:
            raise ValueError("Schema ID cannot be empty.")
        if not isinstance(schema, dict):
            raise ValueError("Schema must be a dictionary.")

        # Add the schema to the store, using the schema_id as its URI
        # The resolver will then be able to find it using "schema_id" or "#/definitions/schema_id"
        self._schema_store.add(schema_id, schema)
        self.logger.debug(f"Schema '{schema_id}' registered successfully.")

    def register_binary_data_schema(self, format: str) -> None:
        """
        Registers a JSON schema for binary data with the given format.

        Args:
            format: The media type format of the binary data (e.g., "image/jpeg", "audio/wav").
        """
        # Base binary data schema
        binary_schema = {
            "type": "object",
            "properties": {
                "__type__": {"type": "string", "const": "binary"},
                "format": {"type": "string", "const": format},
                "encoding": {"type": "string", "const": "base64"},
                "data": {"type": "string"}
            },
            "required": ["__type__", "format", "encoding", "data"]
        }

        # Add format-specific properties
        if format.startswith("audio/"):
            binary_schema["properties"].update({
                "duration": {"type": "number", "minimum": 0},
                "sample_rate": {"type": "integer", "minimum": 1},
                "channels": {"type": "integer", "minimum": 1}
            })
        elif format.startswith("image/"):
            binary_schema["properties"].update({
                "dimensions": {
                    "type": "object",
                    "properties": {
                        "width": {"type": "integer", "minimum": 1},
                        "height": {"type": "integer", "minimum": 1}
                    },
                    "required": ["width", "height"]
                }
            })
        elif format.startswith("video/"):
            binary_schema["properties"].update({
                "duration": {"type": "number", "minimum": 0},
                "frame_rate": {"type": "number", "minimum": 1},
                "dimensions": {
                    "type": "object",
                    "properties": {
                        "width": {"type": "integer", "minimum": 1},
                        "height": {"type": "integer", "minimum": 1}
                    },
                    "required": ["width", "height"]
                }
            })
        elif format.startswith("application/pdf"):
            binary_schema["properties"].update({
                "pages": {"type": "integer", "minimum": 1}
            })

        # Register the schema with a format-specific ID
        schema_id = f"binary_data_{format.replace('/', '_')}"
        self.register_schema(schema_id, binary_schema)
        self.logger.info(f"Registered binary data schema for format: {format}")

    def validate_context(self, data: Any, schema_id: str) -> None:
        """
        Validates the given data against a registered JSON schema.

        Args:
            data: The context data to validate.
            schema_id: The ID of the schema to validate against.

        Raises:
            ContextValidationError: If validation fails or the schema is not found.
            ValueError: If schema_id is empty.
        """
        if not schema_id:
            raise ValueError("Schema ID cannot be empty for validation.")

        try:
            # Retrieve the schema from the store using its ID
            schema = self._schema_store.get(schema_id)
            if schema is None:
                raise ContextValidationError(f"Schema '{schema_id}' not found in validator store.")

            validate(instance=data, schema=schema, resolver=self._resolver)
            self.logger.debug(f"Context data successfully validated against schema '{schema_id}'.")
        except ValidationError as e:
            self.logger.error(f"Context validation failed against schema '{schema_id}': {e.message}")
            if hasattr(e, 'path') and e.path:
                self.logger.error(f"Validation path: {list(e.path)}")
            if hasattr(e, 'schema') and e.schema:
                self.logger.debug(f"Expected schema: {e.schema}")
            if hasattr(e, 'instance') and e.instance:
                self.logger.debug(f"Invalid value: {e.instance}")
            raise ContextValidationError(f"Context validation failed: {e.message}") from e
        except Exception as e:
            self.logger.error(f"An unexpected error occurred during validation: {e}")
            raise ContextValidationError(f"An unexpected error occurred during validation: {e}") from e

    def get_registered_schemas(self) -> Dict[str, Dict[str, Any]]:
        """
        Returns a dictionary of all currently registered schemas.

        Returns:
            Dict[str, Dict[str, Any]]: A dictionary where keys are schema IDs and values are the schemas.
        """
        # The SchemaStore doesn't directly expose its internal dictionary,
        # so we'll iterate through known URIs if we need to list them.
        # For now, a direct access is not straightforward without internal knowledge.
        # This method might need adjustment based on jsonschema library's future API.
        self.logger.warning("get_registered_schemas is a placeholder. Direct access to SchemaStore content is limited.")
        # TODO: Find a better way to expose registered schemas from jsonschema.SchemaStore if needed for debugging/listing.
        return {} # Placeholder for now
"""
ContextService module.

Provides a centralized service for managing pipeline context with support for
data storage, namespacing, and basic access control.
"""

import threading
import logging
import sys
import time
import os
import uuid # Added for unique key generation
from pathlib import Path
from typing import Any, Dict, Optional, List, Type, Union, BinaryIO, Tuple # Added Tuple
import json
import os
import base64
import ast # Added for _parse_if_str_for_aggregation

# Import data_types for binary data structures
from src.data_types import BinaryDataType, create_binary_data # Added
# Import storage components for binary objects
from src.context.storage import StorageBackend, FilesystemBackend, StorageBackendError # Added

# Import the new ContextValidator
from src.ContextValidator import ContextValidator, ContextValidationError
from src.audit_logger import AuditLogger
from src.access_control.PermissionManager import PermissionManager

# Import persistence components
try:
    from src.context import PersistenceManager, FilesystemBackend
except ImportError:
    # Fallback for when the context package is not available
    class DummyPersistence:
        def __init__(self, *args, **kwargs):
            pass
        def save_context(self, *args, **kwargs):
            return False
        def load_context(self, *args, **kwargs):
            return {}
        def delete_context(self, *args, **kwargs):
            return False
    
    class DummyBackend:
        pass
    
    PersistenceManager = type('PersistenceManager', (), {'__init__': DummyPersistence.__init__})
    FilesystemBackend = type('FilesystemBackend', (), {})

class ContextService:
    """
    Centralized service for managing pipeline context.

    This service provides a robust and flexible way to store, retrieve, and
    manage contextual data across different parts of the pipeline. It supports
    namespacing to prevent key collisions, includes a placeholder for
    access control mechanisms, and handles context merging and versioning,
    now with integrated data validation.

    Attributes:
        _context_store: The in-memory dictionary for storing context data.
                        Structured as {namespace: {key: value}}.
        _context_lock: A re-entrant lock for thread-safe operations on the context store.
        _context_history: History of context states for snapshots.
        _history_lock: Lock for history operations.
        _validator: An instance of ContextValidator for schema-based data validation.
        logger: Logger instance for logging service operations and errors.
        _permission_manager: An instance of PermissionManager for access control.
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None) -> None:
        """
        Initializes the ContextService with an empty context store, thread locks, history,
        and instances of ContextValidator and AuditLogger.
        
        Args:
            config: Optional configuration dictionary. If provided, should contain a 'storage' key
                   with persistence configuration.
        """
        self.logger = logging.getLogger(__name__)
        self._context_store: Dict[str, Dict[str, Any]] = {}
        self._context_lock = threading.RLock() # Use RLock for re-entrant locking
        self._context_history: Dict[int, Dict[str, Any]] = {}
        self._history_lock = threading.Lock()
        self._validator = ContextValidator() # Initialize the ContextValidator
        self.audit_logger = AuditLogger() # Initialize the AuditLogger
        self._performance_metrics: Dict[str, Dict[str, Any]] = {} # Stores metrics like access_time, size
        
        # Initialize persistence if configured
        self._persistence = None
        if config and config.get('storage', {}).get('enabled', False):
            storage_config = config['storage']
            backend_type = storage_config.get('backend', 'filesystem')
            
            try:
                if backend_type == 'filesystem':
                    base_path = storage_config.get('base_path', 'context_data')
                    self._persistence = PersistenceManager(
                        FilesystemBackend,
                        base_path=base_path
                    )
                    self.logger.info(f"Initialized filesystem persistence with path: {base_path}")
                else:
                    self.logger.warning(f"Unsupported backend type: {backend_type}")
            except Exception as e:
                self.logger.error(f"Failed to initialize persistence backend: {e}", exc_info=True)

        # Initialize PermissionManager
        access_control_config = config.get('access_control', {}) if config else {}
        self._permission_manager = PermissionManager(
            policies=access_control_config.get('policies', []),
            enabled=access_control_config.get('enabled', True)
        )
        self.logger.info(f"ContextService initialized. Schema validation: {'enabled' if self.schema_validator else 'disabled'}. Binary storage: {'configured' if self._binary_storage_backend else 'not configured'}.")

        # Initialize dedicated binary storage backend
        self._binary_storage_backend: Optional[StorageBackend] = None
        if config and config.get('storage', {}).get('enabled', False):
            storage_config = config['storage']
            backend_type = storage_config.get('backend', 'filesystem')
            # binary_storage_config can be a separate section or reuse parts of 'storage'
            binary_storage_specific_config = storage_config.get('binary_storage_options', {})

            try:
                if backend_type == 'filesystem':
                    default_binary_base_path = Path(storage_config.get('base_path', 'context_data')) / 'binaries'
                    binary_base_path = binary_storage_specific_config.get('base_path', str(default_binary_base_path))
                    self._binary_storage_backend = FilesystemBackend(base_path=binary_base_path)
                    self.logger.info(f"Initialized FilesystemBackend for binary objects at: {binary_base_path}")
                # Potentially add other backend types for binary storage here in the future
                else:
                    self.logger.warning(f"Unsupported backend type '{backend_type}' for dedicated binary object storage.")
            except Exception as e:
                self.logger.error(f"Failed to initialize dedicated binary storage backend: {e}", exc_info=True)

    # --- Methods to replace plugin functionalities --- 

    def _parse_if_str_for_aggregation(self, val: Any) -> Any:
        """Helper to attempt to parse a string-encoded list or dict for aggregation.
           Uses ast.literal_eval for safety.
        """
        if isinstance(val, str):
            try:
                parsed = ast.literal_eval(val)
                if isinstance(parsed, (list, dict)):
                    self.logger.debug(f"Parsed string to {type(parsed)} for aggregation.")
                    return parsed
            except (ValueError, SyntaxError, TypeError) as e: # More specific exceptions
                self.logger.debug(f"String is not a parsable list/dict for aggregation: {e}, using original string.")
        return val

    def set_multiple_contexts(self, namespace: str, items: Dict[str, Any], actor: str, 
                              overwrite_all: bool = True, schema_ids: Optional[Dict[str, str]] = None) -> Dict[str, bool]:
        """Sets multiple key-value pairs in a given namespace.

        Args:
            namespace: The namespace to set items in.
            items: A dictionary of key-value pairs to set.
            actor: The actor performing the operation.
            overwrite_all: If True, all existing keys will be overwritten.
            schema_ids: Optional dictionary mapping item keys to their schema_ids.

        Returns:
            A dictionary mapping each key from 'items' to a boolean indicating success.
        """
        results = {}
        if not isinstance(items, dict):
            self.logger.error(f"[set_multiple_contexts] 'items' must be a dictionary. Actor: {actor}")
            raise ValueError("'items' must be a dictionary.")

        for key, value in items.items():
            try:
                schema_id = schema_ids.get(key) if schema_ids else None
                self.set_context(namespace, key, value, 
                                 overwrite=overwrite_all, 
                                 actor=actor, 
                                 schema_id=schema_id)
                results[key] = True
            except Exception as e:
                self.logger.error(f"[set_multiple_contexts] Error setting {namespace}.{key}: {e}. Actor: {actor}", exc_info=True)
                results[key] = False
        return results

    def append_to_context_list(self, namespace: str, list_key: str, item_to_append: Any, 
                               actor: str, parse_strings: bool = False, 
                               required_sub_keys: Optional[List[str]] = None) -> bool:
        """Appends an item to a list stored in a context key.
           If the key doesn't exist or isn't a list, it initializes/replaces it with a new list.
        """
        try:
            current_list = self.get_context(namespace, list_key, actor=actor, default=[])
            
            if not isinstance(current_list, list):
                self.logger.warning(f"[append_to_context_list] Existing value for {namespace}.{list_key} is not a list (type: {type(current_list)}). Replacing with new list. Actor: {actor}")
                current_list = []
            
            if parse_strings:
                item_to_append = self._parse_if_str_for_aggregation(item_to_append)

            if required_sub_keys and isinstance(item_to_append, dict):
                for r_key in required_sub_keys:
                    item_to_append.setdefault(r_key, "N/A")
            
            current_list.append(item_to_append)
            self.set_context(namespace, list_key, current_list, overwrite=True, actor=actor)
            self.logger.info(f"[append_to_context_list] Appended to {namespace}.{list_key}. New length: {len(current_list)}. Actor: {actor}")
            return True
        except Exception as e:
            self.logger.error(f"[append_to_context_list] Error appending to {namespace}.{list_key}: {e}. Actor: {actor}", exc_info=True)
            return False

    def copy_context_value(self, source_namespace: str, source_key: str, 
                           target_namespace: str, target_key: str, actor: str, 
                           remove_source: bool = False, overwrite_target: bool = True) -> bool:
        """Copies a value from a source context entry to a target context entry.
           Assumes self.key_exists and self.delete_context methods are available.
        """
        try:
            value_to_copy = self.get_context(source_namespace, source_key, actor=actor)
            # Check if key truly doesn't exist vs. value is None, relies on key_exists method
            if value_to_copy is None and not self.key_exists(source_namespace, source_key, actor=actor): 
                self.logger.warning(f"[copy_context_value] Source {source_namespace}.{source_key} not found. Actor: {actor}")
                return False

            self.set_context(target_namespace, target_key, value_to_copy, 
                             overwrite=overwrite_target, actor=actor)
            self.logger.info(f"[copy_context_value] Copied {source_namespace}.{source_key} to {target_namespace}.{target_key}. Actor: {actor}")

            if remove_source:
                self.delete_context(source_namespace, source_key, actor=actor) 
                self.logger.info(f"[copy_context_value] Removed source {source_namespace}.{source_key}. Actor: {actor}")
            return True
        except AttributeError as e:
            if 'key_exists' in str(e) or 'delete_context' in str(e):
                 self.logger.error(f"[copy_context_value] Missing required method (key_exists or delete_context): {e}. Actor: {actor}", exc_info=True)
                 raise NotImplementedError(f"ContextService is missing method: {e}. This is needed for copy_context_value with remove_source or accurate source checking.")
            self.logger.error(f"[copy_context_value] Error copying {source_namespace}.{source_key} to {target_namespace}.{target_key}: {e}. Actor: {actor}", exc_info=True)
            return False
        except Exception as e:
            self.logger.error(f"[copy_context_value] Error copying {source_namespace}.{source_key} to {target_namespace}.{target_key}: {e}. Actor: {actor}", exc_info=True)
            return False

    def load_file_into_context(self, filepath: str, file_type: str, namespace: str, context_key: str, actor: str, 
                               binary_mime_type: Optional[str] = None, 
                               binary_metadata_kwargs: Optional[dict] = None, 
                               overwrite: bool = True, schema_id: Optional[str] = None) -> bool:
        """Reads data from a specified file path and stores it in the context."""
        if not os.path.isabs(filepath):
            self.logger.error(f"[load_file_into_context] Filepath must be absolute: {filepath}. Actor: {actor}")
            return False
        if not os.path.exists(filepath) or not os.path.isfile(filepath):
            self.logger.error(f"[load_file_into_context] File not found or is not a file: {filepath}. Actor: {actor}")
            return False

        try:
            if file_type == 'json':
                with open(filepath, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                self.set_context(namespace, context_key, data, 
                                 overwrite=overwrite, actor=actor, schema_id=schema_id)
                self.logger.info(f"[load_file_into_context] Loaded JSON from {filepath} to {namespace}.{context_key}. Actor: {actor}")
            elif file_type == 'binary':
                if not self._binary_storage_backend:
                    self.logger.error(f"[load_file_into_context] Binary storage not configured. Cannot load binary file. Actor: {actor}")
                    return False
                if not binary_mime_type:
                    self.logger.error(f"[load_file_into_context] 'binary_mime_type' is required for file_type 'binary'. Actor: {actor}")
                    return False
                with open(filepath, 'rb') as f_stream:
                    self.save_binary_data(namespace, context_key, f_stream, 
                                          file_format=binary_mime_type, 
                                          actor=actor, 
                                          overwrite_metadata=overwrite, 
                                          **(binary_metadata_kwargs or {}))
                self.logger.info(f"[load_file_into_context] Loaded binary from {filepath} to {namespace}.{context_key} ({binary_mime_type}). Actor: {actor}")
            else:
                self.logger.error(f"[load_file_into_context] Unsupported file_type: {file_type}. Actor: {actor}")
                return False
            return True
        except Exception as e:
            self.logger.error(f"[load_file_into_context] Error loading {filepath} to {namespace}.{context_key}: {e}. Actor: {actor}", exc_info=True)
            return False

    def save_context_value_to_file(self, namespace: str, context_key: str, filepath: str, actor: str, 
                                   output_format: str = 'json', create_directories: bool = True) -> bool:
        """Retrieves a context value and saves it to a specified file path.
           Assumes self.key_exists method is available.
        """
        if not os.path.isabs(filepath):
            self.logger.error(f"[save_context_value_to_file] Filepath must be absolute: {filepath}. Actor: {actor}")
            return False

        try:
            value = self.get_context(namespace, context_key, actor=actor)
            if value is None and not self.key_exists(namespace, context_key, actor=actor):
                self.logger.warning(f"[save_context_value_to_file] Key {namespace}.{context_key} not found. Nothing to save. Actor: {actor}")
                return False

            if create_directories:
                dir_name = os.path.dirname(filepath)
                if dir_name: 
                    os.makedirs(dir_name, exist_ok=True)

            if output_format == 'json':
                with open(filepath, 'w', encoding='utf-8') as f:
                    json.dump(value, f, indent=2)
                self.logger.info(f"[save_context_value_to_file] Saved {namespace}.{context_key} as JSON to {filepath}. Actor: {actor}")
            elif output_format == 'binary':
                if not isinstance(value, dict) or value.get('__type__') != 'binary':
                    self.logger.error(f"[save_context_value_to_file] Value at {namespace}.{context_key} is not BinaryDataType. Cannot save as binary. Actor: {actor}")
                    return False
                
                if value.get("data_location") == "inline":
                    if 'data' not in value or 'encoding' not in value:
                        self.logger.error(f"[save_context_value_to_file] Inline BinaryDataType at {namespace}.{context_key} is missing 'data' or 'encoding'. Actor: {actor}")
                        return False
                    if value['encoding'] != 'base64':
                        self.logger.error(f"[save_context_value_to_file] Inline BinaryDataType at {namespace}.{context_key} has unsupported encoding '{value['encoding']}'. Only 'base64' supported for direct save. Actor: {actor}")
                        return False
                    decoded_data = base64.b64decode(value["data"])
                    with open(filepath, 'wb') as f:
                        f.write(decoded_data)
                elif value.get("data_location") == "referenced":
                    if not self._binary_storage_backend:
                        self.logger.error(f"[save_context_value_to_file] Binary storage not configured. Cannot load referenced binary data. Actor: {actor}")
                        return False
                    binary_stream = self.load_binary_data(namespace, context_key, actor=actor)
                    if binary_stream:
                        with open(filepath, 'wb') as f:
                            while True:
                                chunk = binary_stream.read(8192)
                                if not chunk:
                                    break
                                f.write(chunk)
                        binary_stream.close()
                    else:
                        self.logger.error(f"[save_context_value_to_file] Failed to load binary stream for {namespace}.{context_key}. Actor: {actor}")
                        return False
                else:
                    self.logger.error(f"[save_context_value_to_file] Unknown data_location '{value.get('data_location')}' for BinaryDataType at {namespace}.{context_key}. Actor: {actor}")
                    return False
                self.logger.info(f"[save_context_value_to_file] Saved {namespace}.{context_key} as binary to {filepath}. Actor: {actor}")
            else:
                self.logger.error(f"[save_context_value_to_file] Unsupported output_format: {output_format}. Actor: {actor}")
                return False
            return True
        except AttributeError as e:
            if 'key_exists' in str(e):
                 self.logger.error(f"[save_context_value_to_file] Missing required method (key_exists): {e}. Actor: {actor}", exc_info=True)
                 raise NotImplementedError(f"ContextService is missing method: {e}. This is needed for save_context_value_to_file for accurate source checking.")
            self.logger.error(f"[save_context_value_to_file] Error saving {namespace}.{context_key} to {filepath}: {e}. Actor: {actor}", exc_info=True)
            return False
        except Exception as e:
            self.logger.error(f"[save_context_value_to_file] Error saving {namespace}.{context_key} to {filepath}: {e}. Actor: {actor}", exc_info=True)
            return False

    def get_context_snapshot(self, namespace: str, actor: str) -> Dict[str, Any]:
        """Provides a dictionary representation of a namespace's content."""
        self.logger.info(f"[get_context_snapshot] Generating snapshot for namespace '{namespace}'. Actor: {actor}")
        with self._context_lock: # Corrected lock name
            if namespace in self._context_store:
                try:
                    # Attempt deep copy via JSON serialization for safety with shared structures
                    return json.loads(json.dumps(self._context_store[namespace]))
                except TypeError: 
                    # Fallback for data not fully JSON serializable (e.g., custom objects not handled by BinaryDataType)
                    import copy 
                    self.logger.warning(f"[get_context_snapshot] Data in namespace '{namespace}' not fully JSON serializable for deep copy via JSON. Using copy.deepcopy(). Actor: {actor}")
                    return copy.deepcopy(self._context_store[namespace]) # Potentially slower
            else:
                self.logger.warning(f"[get_context_snapshot] Namespace '{namespace}' not found. Actor: {actor}")
                return {}

    def _check_access(self, namespace: str, key: Optional[str] = None,
                      permission_type: str = "read", actor: str = "system") -> bool:
        """
        Internal method to check access permissions using the PermissionManager.

        Args:
            namespace: The namespace being accessed.
            key: The specific key within the namespace (optional).
            permission_type: The type of permission required (e.g., "read", "write", "delete", "snapshot", "restore").
            actor: The role of the entity requesting access.

        Returns:
            bool: True if access is granted, False otherwise.
        """
        resource = f"{namespace}.{key}" if key else namespace
        return self._permission_manager.check_permission(actor, resource, permission_type)

    def _resolve_context_path(self, context: Dict[str, Any], path: str, create_namespaces: bool = False) -> Tuple[Dict[str, Any], str]:
        """
        Resolves a dot-separated context path (e.g., 'namespace.key' or 'namespace.sub.key')
        and ensures the namespace/intermediate dictionaries exist if create_namespaces is True.

        Args:
            context: The base context dictionary (e.g., self._context_store or a specific namespace dict).
            path: The dot-separated path string.
            create_namespaces: If True, creates missing intermediate dictionaries along the path.

        Returns:
            A tuple containing:
                - The parent dictionary where the final key should be set/retrieved.
                - The final key itself.

        Raises:
            ValueError: If the path is invalid or a non-dict intermediate exists and create_namespaces is False.
            KeyError: If an intermediate key does not exist and create_namespaces is False.
        """
        if not path:
            raise ValueError("Context path cannot be empty.")

        parts = path.split('.')
        current_dict = context
        for i, part in enumerate(parts[:-1]):
            if part not in current_dict:
                if create_namespaces:
                    current_dict[part] = {}
                    self.audit_logger.log_operation(
                        operation="create",
                        entity_type="context_namespace",
                        entity_id=f"{path.rsplit('.', len(parts) - i - 1)[0]}",
                        actor="system", # Internal operation
                        new_value={},
                        metadata={"path_segment": part}
                    )
                else:
                    raise KeyError(f"Path segment '{part}' not found in context path '{path}'.")
            
            if not isinstance(current_dict[part], dict):
                raise TypeError(f"Intermediate path segment '{part}' is not a dictionary in context path '{path}'.")
            current_dict = current_dict[part]
        
        return current_dict, parts[-1]


    def set_value(self, context: Dict[str, Any], path: str, value: Any,
                  overwrite: bool = True, enforce_access: bool = True,
                  schema_id: Optional[str] = None, actor: str = "system") -> bool:
        """
        Sets a context value at a specific dot-separated path, with optional schema validation.
        This is a more generic version of set_context, allowing setting values within nested dictionaries.

        Args:
            context: The base context dictionary to modify (e.g., self._context_store or a specific namespace).
            path: The dot-separated path for the context key (e.g., 'namespace.key' or 'namespace.sub.key').
            value: The value to associate with the key.
            overwrite: If True, overwrites an existing key; otherwise, returns False if key exists.
            enforce_access: If True, performs an access control check before setting.
            schema_id: Optional ID of a registered schema to validate the value against.
            actor: The actor performing the operation.

        Returns:
            bool: True if the context was set successfully, False otherwise (e.g., no overwrite, access denied).

        Raises:
            ValueError: If path is empty or invalid.
            PermissionError: If access is denied and enforce_access is True.
            ContextValidationError: If schema validation fails.
            TypeError: If an intermediate path segment is not a dictionary.
        """
        if not path:
            raise ValueError("Context path cannot be empty.")
        
        # Extract namespace and key for permission checking
        namespace_parts = path.split('.')
        namespace = namespace_parts[0]
        key = '.'.join(namespace_parts[1:]) if len(namespace_parts) > 1 else None

        if enforce_access and not self._check_access(namespace, key, "write", actor):
            raise PermissionError(f"Access denied for '{actor}' to write context at path '{path}'.")

        # Perform validation if a schema_id is provided
        if schema_id:
            try:
                self._validator.validate_context(value, schema_id)
            except ContextValidationError as e:
                self.logger.error(f"Validation failed for context path '{path}' with schema '{schema_id}': {e}")
                raise # Re-raise the validation error

        with self._context_lock:
            try:
                parent_dict, final_key = self._resolve_context_path(context, path, create_namespaces=True)
            except (ValueError, KeyError, TypeError) as e:
                raise ValueError(f"Invalid context path '{path}': {e}") from e

            old_value = parent_dict.get(final_key)
            operation_type = "update" if final_key in parent_dict else "create"

            if not overwrite and final_key in parent_dict:
                self.logger.info(f"Context path '{path}' not overwritten (overwrite=False).")
                return False

            parent_dict[final_key] = value
            self.logger.debug(f"Context path '{path}' set.")

            self.audit_logger.log_operation(
                operation=operation_type,
                entity_type="context_value",
                entity_id=path,
                actor=actor,
                old_value=old_value,
                new_value=value,
                metadata={
                    "path": path,
                    "overwrite": overwrite,
                    "schema_id": schema_id
                }
            )
            self._update_performance_metrics(namespace, final_key, sys.getsizeof(value)) # Use namespace and final_key for metrics
            return True

    def set_context(self, namespace: str, key: str, value: Any,
                    overwrite: bool = True, enforce_access: bool = True,
                    schema_id: Optional[str] = None, actor: str = "system") -> bool:
        """
        Sets a context value within a specific namespace, with optional schema validation.

        Args:
            namespace: The namespace for the context key.
            key: The context key to set.
            value: The value to associate with the key.
            overwrite: If True, overwrites an existing key; otherwise, returns False if key exists.
            enforce_access: If True, performs an access control check before setting.
            schema_id: Optional ID of a registered schema to validate the value against.

        Returns:
            bool: True if the context was set successfully, False otherwise (e.g., no overwrite, access denied).

        Raises:
            ValueError: If namespace or key is empty.
            PermissionError: If access is denied and enforce_access is True.
            ContextValidationError: If schema validation fails.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")
        if not key:
            raise ValueError("Context key cannot be empty.")

        # Use the new set_value method for consistency
        return self.set_value(
            context=self._context_store,
            path=f"{namespace}.{key}",
            value=value,
            overwrite=overwrite,
            enforce_access=enforce_access,
            schema_id=schema_id,
            actor=actor
        )

        
        

    def get_context(self, namespace: str, key: Optional[str] = None,
                    enforce_access: bool = True, actor: str = "system") -> Any:
        """
        Retrieves a context value or an entire namespace's context with audit logging.

        Args:
            namespace: The namespace to retrieve context from.
            key: The specific key to retrieve. If None, returns the entire namespace's context.
            enforce_access: If True, performs an access control check before retrieving.
            actor: Who is performing the action (default: "system").

        Returns:
            Any: The requested context value, the namespace dictionary, or None if not found.

        Raises:
            ValueError: If namespace is empty.
            PermissionError: If access is denied and enforce_access is True.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")

        if enforce_access and not self._check_access(namespace, key, "read", actor):
            raise PermissionError(f"Access denied for '{actor}' to read context in namespace '{namespace}' for key '{key}'.")

        with self._context_lock:
            if namespace not in self._context_store:
                self.logger.debug(f"Namespace '{namespace}' not found.")
                exists = False
                value = None
            else:
                exists = True
                if key is None:
                    value = self._context_store[namespace].copy()
                else:
                    value = self._context_store[namespace].get(key)
                    exists = key in self._context_store[namespace]

            # Log the context access
            self.audit_logger.log_operation(
                operation="get",
                entity_type="context",
                entity_id=f"{namespace}.{key}" if key else namespace,
                actor=actor,
                metadata={
                    "value_type": type(value).__name__ if value else None,
                    "namespace": namespace,
                    "exists": exists,
                    "full_namespace": key is None
                }
            )
            return value

    def key_exists(self, namespace: str, key: str, actor: str = "system", enforce_access: bool = True) -> bool:
        """Checks if a specific key exists within a namespace.

        Args:
            namespace: The namespace to check within.
            key: The key to check for.
            actor: The actor performing the operation.
            enforce_access: If True, performs an access control check.

        Returns:
            bool: True if the key exists, False otherwise.

        Raises:
            ValueError: If namespace or key is empty.
            PermissionError: If access is denied and enforce_access is True.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")
        if not key:
            raise ValueError("Context key cannot be empty.")

        if enforce_access and not self._check_access(namespace, key, "read", actor): # Check read access
            # If we can't read it, we can't definitively say if it exists or not from user's perspective
            # However, for internal checks or if read access is denied, it effectively doesn't exist for this actor
            self.logger.warning(f"Access denied for '{actor}' to check existence of key '{key}' in namespace '{namespace}'. Assuming non-existent for this actor.")
            return False # Or raise PermissionError depending on desired strictness

        with self._context_lock:
            exists = namespace in self._context_store and key in self._context_store[namespace]
            self.logger.debug(f"Key '{key}' in namespace '{namespace}' exists: {exists}. Actor: {actor}")
            return exists



    def delete_context(self, namespace: str, key: Optional[str] = None,
                       enforce_access: bool = True, actor: str = "system") -> bool:
        """
        Deletes a specific context key or an entire namespace with audit logging.

        Args:
            namespace: The namespace from which to delete.
            key: The specific key to delete. If None, deletes the entire namespace.
            enforce_access: If True, performs an access control check before deleting.
            actor: Who is performing the action (default: "system").

        Returns:
            bool: True if deletion was successful, False if the key/namespace was not found.

        Raises:
            ValueError: If namespace is empty.
            PermissionError: If access is denied and enforce_access is True.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")

        # For delete, we check 'write' permission as it's a modification
        if enforce_access and not self._check_access(namespace, key, "write", actor):
            raise PermissionError(f"Access denied for '{actor}' to delete in namespace '{namespace}' for key '{key}'.")

        with self._context_lock:
            if namespace not in self._context_store:
                self.audit_logger.log_operation(
                    operation="delete",
                    entity_type="context",
                    entity_id=f"{namespace}.{key}" if key else namespace,
                    actor=actor,
                    metadata={
                        "namespace": namespace,
                        "exists": False
                    }
                )
                self.logger.debug(f"Deletion failed: Namespace '{namespace}' not found.")
                return False

            if key is None:
                deleted_count = len(self._context_store[namespace])
                del self._context_store[namespace]
                self.logger.info(f"Deleted entire namespace '{namespace}'.")
                self.audit_logger.log_operation(
                    operation="delete",
                    entity_type="context",
                    entity_id=namespace,
                    actor=actor,
                    metadata={
                        "namespace": namespace,
                        "count": deleted_count,
                        "full_namespace": True
                    }
                )
                self._remove_performance_metrics(namespace)
                return True
            elif key in self._context_store[namespace]:
                value = self._context_store[namespace][key]
                del self._context_store[namespace][key]
                self.logger.info(f"Deleted key '{key}' from namespace '{namespace}'.")
                self.audit_logger.log_operation(
                    operation="delete",
                    entity_type="context",
                    entity_id=f"{namespace}.{key}",
                    actor=actor,
                    metadata={
                        "value_type": type(value).__name__,
                        "namespace": namespace
                    }
                )
                self._remove_performance_metrics(namespace, key)
                return True
            else:
                self.audit_logger.log_operation(
                    operation="delete",
                    entity_type="context",
                    entity_id=f"{namespace}.{key}",
                    actor=actor,
                    metadata={
                        "namespace": namespace,
                        "exists": False
                    }
                )
                self.logger.debug(f"Deletion failed: Key '{key}' not found in namespace '{namespace}'.")
                return False

    def merge_context(self, namespace: str, new_context: Dict[str, Any], conflict_strategy: str = 'overwrite',
                       enforce_access: bool = True, schema_id: Optional[str] = None, actor: str = "system") -> Dict[str, Any]:
        """
        Merges new context into existing context within a specific namespace, with optional schema validation.

        Args:
            namespace: The namespace to merge context into.
            new_context: Dictionary of new context values to merge.
            conflict_strategy: How to handle conflicts ('overwrite', 'keep', 'merge').
                                - 'overwrite': New values replace existing ones.
                                - 'keep': Existing values are kept if a conflict occurs.
                                - 'merge': If both values are dictionaries, they are recursively merged.
                                            Otherwise, 'overwrite' strategy is used.
            enforce_access: If True, performs an access control check before merging.
            schema_id: Optional ID of a registered schema to validate the entire new_context against.

        Returns:
            Dict[str, Any]: A dictionary of conflicts that were handled, where keys are the conflicting
                            keys and values are the *original* values that were replaced or the *new*
                            values that were kept out (depending on strategy).

        Raises:
            ValueError: If namespace is empty or conflict_strategy is invalid.
            PermissionError: If access is denied and enforce_access is True.
            ContextValidationError: If schema validation fails for the new_context.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")
        if conflict_strategy not in ('overwrite', 'keep', 'merge'):
            raise ValueError(f"Invalid conflict strategy: {conflict_strategy}")

        if enforce_access and not self._check_access(namespace, permission_type="write", actor=actor):
            raise PermissionError(f"Access denied for '{actor}' to merge context in namespace '{namespace}'.")

        # Perform validation for the entire new_context if a schema_id is provided
        if schema_id:
            try:
                self._validator.validate_context(new_context, schema_id)
            except ContextValidationError as e:
                self.logger.error(f"Validation failed for merged context in namespace '{namespace}' with schema '{schema_id}': {e}")
                raise # Re-raise the validation error

        conflicts = {}
        with self._context_lock:
            if namespace not in self._context_store:
                self._context_store[namespace] = {}

            for key, value in new_context.items():
                current_value = self._context_store[namespace].get(key)

                if current_value is not None:  # Key exists in context
                    if conflict_strategy == 'overwrite':
                        conflicts[key] = current_value
                        self._context_store[namespace][key] = value
                    elif conflict_strategy == 'keep':
                        conflicts[key] = value  # The new value is the 'conflict' that was kept out
                    elif conflict_strategy == 'merge':
                        if isinstance(value, dict) and isinstance(current_value, dict):
                            merged_dict = current_value.copy()
                            merged_dict.update(value)
                            conflicts[key] = current_value  # The original value is the 'conflict' before merge
                            self._context_store[namespace][key] = merged_dict
                        else:
                            conflicts[key] = current_value
                            self._context_store[namespace][key] = value
                else:  # Key does not exist, simply set it
                    self._context_store[namespace][key] = value
                
                # Log merge operation for each key
                self.audit_logger.log_operation(
                    operation="merge",
                    entity_type="context",
                    entity_id=f"{namespace}.{key}",
                    actor=actor,
                    old_value=current_value,
                    new_value=self._context_store[namespace][key],
                    metadata={
                        "namespace": namespace,
                        "key": key,
                        "conflict_strategy": conflict_strategy
                    }
                )
                self._update_performance_metrics(namespace, key, sys.getsizeof(self._context_store[namespace][key]))

        self.logger.debug(f"Context merged into namespace '{namespace}' with strategy '{conflict_strategy}'.")
        return conflicts

    def snapshot_context(self, namespace: str, actor: str = None, enforce_access: bool = True) -> int:
        """
        Takes a snapshot of the current context state for a given namespace.

        Args:
            namespace: The namespace to snapshot.
            actor: The actor requesting the snapshot operation.
            enforce_access: If True, performs an access control check before snapshotting.

        Returns:
            int: The snapshot ID that can be used to restore this state, or -1 if snapshot failed.

        Raises:
            ValueError: If namespace is empty.
            PermissionError: If access is denied and enforce_access is True.
        """
        if not namespace:
            raise ValueError("Context namespace cannot be empty.")

        if enforce_access and not self._check_access(namespace, permission_type="snapshot", actor=actor):
            raise PermissionError(f"Access denied for '{actor}' to snapshot context in namespace '{namespace}'.")

        current_context_state = self.get_context(namespace, enforce_access=False) # Get without access check to capture state

        if current_context_state is None:
            self.logger.warning(f"Attempted to snapshot non-existent namespace: {namespace}")
            return -1 # Indicate failure or no snapshot taken

        with self._history_lock:
            snapshot_id = len(self._context_history) + 1
            self._context_history[snapshot_id] = {
                "namespace": namespace,
                "context_state": current_context_state
            }
            self.logger.info(f"Snapshot taken for namespace '{namespace}' with ID: {snapshot_id}")
            return snapshot_id

    def restore_context(self, snapshot_id: int, actor: str = None, enforce_access: bool = True) -> bool:
        """
        Restores context from a snapshot.

        Args:
            snapshot_id: The ID of the snapshot to restore.
            actor: The actor requesting the restore operation.
            enforce_access: If True, performs an access control check before restoring.

        Returns:
            bool: True if restore was successful, False otherwise.

        Raises:
            PermissionError: If access is denied and enforce_access is True.
        """
        with self._history_lock:
            if snapshot_id not in self._context_history:
                self.logger.warning(f"Snapshot ID {snapshot_id} not found in history.")
                return False

            snapshot_data = self._context_history[snapshot_id]
            namespace_to_restore = snapshot_data["namespace"]
            context_state_to_restore = snapshot_data["context_state"]

            if enforce_access and not self._check_access(namespace_to_restore, permission_type="restore", actor=actor):
                raise PermissionError(f"Access denied for '{actor}' to restore context in namespace '{namespace_to_restore}'.")

            with self._context_lock:
                # Clear existing context in the target namespace before restoring
                self.delete_context(namespace_to_restore, enforce_access=False) # Delete without access check as permission is for restore

                # Restore context by setting all values from the snapshot
                if namespace_to_restore not in self._context_store:
                    self._context_store[namespace_to_restore] = {}
                for key, value in context_state_to_restore.items():
                    self._context_store[namespace_to_restore][key] = value # Direct assignment, access already checked

            self.logger.info(f"Context for namespace '{namespace_to_restore}' restored from snapshot ID: {snapshot_id}")
            return True

    def list_namespaces(self, enforce_access: bool = True, actor: str = "system") -> List[str]:
        """
        Lists all available namespaces.

        Args:
            enforce_access: If True, performs an access control check for each namespace.
            actor: The role of the entity performing the action (default: "system").

        Returns:
            List[str]: A list of accessible namespace names.
        """
        accessible_namespaces = []
        with self._context_lock:
            for namespace in self._context_store.keys():
                if not enforce_access or self._check_access(namespace, permission_type="read", actor=actor):
                    accessible_namespaces.append(namespace)
        return accessible_namespaces

    def clear_all_context(self, enforce_access: bool = True, actor: str = "system") -> None:
        """
        Clears all context data from the store.

        Args:
            enforce_access: If True, performs an access control check before clearing.
            actor: The role of the entity performing the action (default: "system").

        Raises:
            PermissionError: If access is denied and enforce_access is True.
        """
        # For clearing all context, we can check for a broad 'delete' permission on a wildcard resource
        if enforce_access and not self._check_access("all_namespaces", permission_type="delete", actor=actor):
            raise PermissionError(f"Access denied for '{actor}' to clear all context.")

        with self._context_lock:
            self._context_store.clear()
            self._performance_metrics.clear()
            self.logger.info("All context data cleared.")
            self.audit_logger.log_operation(
                operation="clear_all",
                entity_type="context",
                entity_id="all",
                actor=actor,
                metadata={}
            )

    def __len__(self) -> int:
        """
        Returns the total number of keys across all namespaces.
        """
        with self._context_lock:
            return sum(len(ns_data) for ns_data in self._context_store.values())

    def __str__(self) -> str:
        """
        Returns a string representation of the ContextService state.
        """
        with self._context_lock:
            num_namespaces = len(self._context_store)
            total_keys = self.__len__()
            return f"ContextService(namespaces={num_namespaces}, total_keys={total_keys}, snapshots={len(self._context_history)})"

    def __repr__(self) -> str:
        """
        Returns a detailed string representation for debugging.
        """
        with self._context_lock:
            return f"ContextService(context_store={self._context_store}, context_history={self._context_history})"

    def register_context_schema(self, schema_id: str, schema: Dict[str, Any]) -> None:
        """
        Registers a JSON schema with the internal ContextValidator.

        This allows defining expected structures and types for context data,
        which can then be used for validation when setting or merging context.

        Args:
            schema_id: A unique identifier for the schema.
            schema: The JSON schema dictionary.

        Raises:
            ValueError: If schema_id is empty or schema is not a dictionary.
        """
        self._validator.register_schema(schema_id, schema)
        self.logger.info(f"Context schema '{schema_id}' registered with ContextService.")

    def validate_context_data(self, data: Any, schema_id: str) -> None:
        """
        Validates given data against a registered context schema.

        This method can be used by external components to explicitly validate
        data before attempting to set it in the context, or for internal checks.

        Args:
            data: The context data to validate.
            schema_id: The ID of the schema to validate against.

        Raises:
            ContextValidationError: If validation fails or the schema is not found.
            ValueError: If schema_id is empty.
        """
        self._validator.validate_context(data, schema_id)
        self.logger.debug(f"Data successfully validated against context schema '{schema_id}'.")

    def get_performance_metrics(self, namespace: Optional[str] = None, key: Optional[str] = None) -> Dict[str, Any]:
        """
        Retrieves performance metrics for a specific context key, namespace, or all metrics.

        Args:
            namespace: Optional. The namespace to retrieve metrics for.
            key: Optional. The specific key within the namespace to retrieve metrics for.

        Returns:
            Dict[str, Any]: A dictionary containing the requested performance metrics.
                            Returns an empty dictionary if no metrics are found.
        """
        with self._context_lock:
            if namespace is None:
                return self._performance_metrics.copy()
            elif key is None:
                return {k: v for k, v in self._performance_metrics.items() if k.startswith(f"{namespace}.")}
            else:
                metric_key = f"{namespace}.{key}"
                return self._performance_metrics.get(metric_key, {})

    def _update_performance_metrics(self, namespace: str, key: str, size_bytes: int) -> None:
        """
        Internal method to update performance metrics for a given context key.
        Tracks access time, size, and memory usage.
        """
        metric_key = f"{namespace}.{key}"
        current_time = time.time()

        if metric_key not in self._performance_metrics:
            self._performance_metrics[metric_key] = {
                "first_access_time": current_time,
                "last_access_time": current_time,
                "access_count": 0,
                "size_bytes": 0,
                "memory_usage_bytes": 0
            }
        
        metrics = self._performance_metrics[metric_key]
        metrics["last_access_time"] = current_time
        metrics["access_count"] += 1
        metrics["size_bytes"] = size_bytes
        metrics["memory_usage_bytes"] = size_bytes # For simplicity, assuming size_bytes approximates memory usage

        # Aggregate metrics for the namespace
        namespace_metric_key = f"{namespace}._namespace_metrics"
        if namespace_metric_key not in self._performance_metrics:
            self._performance_metrics[namespace_metric_key] = {
                "total_size_bytes": 0,
                "total_memory_usage_bytes": 0,
                "last_updated": current_time
            }
        
        ns_metrics = self._performance_metrics[namespace_metric_key]
        ns_metrics["total_size_bytes"] += size_bytes
        ns_metrics["total_memory_usage_bytes"] += size_bytes
        ns_metrics["last_updated"] = current_time

    def _remove_performance_metrics(self, namespace: str, key: Optional[str] = None) -> None:
        """
        Removes performance metrics for a namespace or a specific key within a namespace.

        Args:
            namespace: The namespace to remove metrics for.
            key: Optional. The specific key within the namespace to remove metrics for.
                 If None, removes all metrics for the namespace.
        """
        with self._context_lock:
            if key is None: # Deleting entire namespace
                keys_to_remove = [k for k in self._performance_metrics if k.startswith(f"{namespace}.")]
                for k in keys_to_remove:
                    del self._performance_metrics[k]
            else: # Deleting specific key
                metric_key = f"{namespace}.{key}"
                if metric_key in self._performance_metrics:
                    del self._performance_metrics[metric_key]

    def save_to_storage(self, namespace: Optional[str] = None) -> bool:
        """
        Save context to persistent storage.
        
        Args:
            namespace: Optional namespace to save. If None, saves all namespaces.
            
        Returns:
            bool: True if successful, False otherwise
        """
        if not self._persistence:
            self.logger.warning("Persistence is not enabled")
            return False
            
        try:
            with self._context_lock:
                if namespace:
                    if namespace not in self._context_store:
                        self.logger.warning(f"Namespace '{namespace}' not found in context")
                        return False
                    return self._persistence.save_context(
                        namespace, 
                        '_state', 
                        self._context_store[namespace]
                    )
                else:
                    success = True
                    for ns, data in self._context_store.items():
                        if not self._persistence.save_context(ns, '_state', data):
                            success = False
                    return success
        except Exception as e:
            self.logger.error(f"Failed to save context to storage: {e}", exc_info=True)
            return False
    
    def load_from_storage(self, namespace: str) -> bool:
        """
        Load context from persistent storage.
        
        Args:
            namespace: Namespace to load
            
        Returns:
            bool: True if successful, False otherwise
        """
        if not self._persistence:
            self.logger.warning("Persistence is not enabled")
            return False
            
        try:
            data = self._persistence.load_context(namespace, '_state')
            if data:
                with self._context_lock:
                    self._context_store[namespace] = data
                self.logger.info(f"Loaded context for namespace '{namespace}' from storage")
                return True
            self.logger.warning(f"No data found for namespace '{namespace}' in storage")
            return False
        except Exception as e:
            self.logger.error(f"Failed to load context from storage: {e}", exc_info=True)
            return False
            
    def delete_from_storage(self, namespace: Optional[str] = None) -> bool:
        """
        Delete context from persistent storage.
        
        Args:
            namespace: Optional namespace to delete. If None, deletes all namespaces.
            
        Returns:
            bool: True if successful, False otherwise
        """
        if not self._persistence:
            self.logger.warning("Persistence is not enabled")
            return False
            
        try:
            if namespace:
                return self._persistence.delete_context(namespace, '_state')
            else:
                # Delete all namespaces
                namespaces = self._persistence.load_context('*', None).keys()
                success = True
                for ns in namespaces:
                    if not self._persistence.delete_context(ns, '_state'):
                        success = False
                return success
        except Exception as e:
            self.logger.error(f"Failed to delete context from storage: {e}", exc_info=True)
            return False
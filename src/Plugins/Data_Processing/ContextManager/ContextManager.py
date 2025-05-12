"""Context management plugin for handling and coordinating pipeline context.

This plugin provides centralized context management capabilities including:
- Context state tracking
- Context merging and conflict resolution
- Context versioning
- Thread-safe context operations
"""

from typing import Any, Dict, Optional
import threading
from src.Plugins.BasePlugin import BasePlugin
from src.Plugins.PluginManager import PluginManager

class ContextManager(BasePlugin):
    """Centralized context management for pipeline operations.
    
    Attributes:
        _context: The current context state
        _context_lock: Thread lock for context operations
        _context_history: History of context states
    """
    
    def __init__(self, plugin_manager: PluginManager) -> None:
        """Initialize the ContextManager.
        
        Args:
            plugin_manager: Reference to the PluginManager for dependency injection
        """
        super().__init__()  # Call parent class __init__ without arguments
        self._context: Dict[str, Any] = {}
        self._context_lock = threading.Lock()
        self._context_history: Dict[int, Dict[str, Any]] = {}
        self.plugin_manager = plugin_manager  # Store plugin_manager reference
        
    def get_context(self, key: Optional[str] = None) -> Any:
        """Get the current context or a specific context value.
        
        Args:
            key: Optional key to get a specific context value. If None, returns entire context.
            
        Returns:
            The requested context value or the full context dict.
            
        TODO: Add tests for:
            - Getting full context
            - Getting specific key
            - Getting non-existent key
            - Thread safety
        """
        with self._context_lock:
            if key is None:
                return self._context.copy()
            return self._context.get(key)
            
    def set_context(self, key: str, value: Any, overwrite: bool = True) -> bool:
        """Set a context value.
        
        Args:
            key: The context key to set
            value: The value to set
            overwrite: Whether to overwrite existing values (default: True)
            
        Returns:
            True if the context was set, False if not set due to overwrite=False
            
        Raises:
            ValueError: If key is empty or None
            
        TODO: Add tests for:
            - Setting new context
            - Setting existing context with overwrite=True/False
            - Setting with invalid key
            - Thread safety
        """
        if not key:
            raise ValueError("Context key cannot be empty")
            
        with self._context_lock:
            if not overwrite and key in self._context:
                return False
                
            self._context[key] = value
            return True
            
    def merge_context(self, new_context: Dict[str, Any], conflict_strategy: str = 'overwrite') -> Dict[str, Any]:
        """Merge new context into existing context.
        
        Args:
            new_context: Dictionary of new context values
            conflict_strategy: How to handle conflicts ('overwrite', 'keep', 'merge')
            
        Returns:
            Dictionary of any conflicts that were handled
            
        Raises:
            ValueError: If conflict_strategy is invalid
            
        TODO: Add tests for:
            - Merge with overwrite strategy
            - Merge with keep strategy
            - Merge with merge strategy
            - Merge with empty context
            - Invalid strategy
            - Thread safety
        """
        if conflict_strategy not in ('overwrite', 'keep', 'merge'):
            raise ValueError(f"Invalid conflict strategy: {conflict_strategy}")
            
        conflicts = {}
        with self._context_lock:
            for key, value in new_context.items():
                if key in self._context:
                    if conflict_strategy == 'overwrite':
                        conflicts[key] = self._context[key]
                        self._context[key] = value
                    elif conflict_strategy == 'keep':
                        conflicts[key] = value
                    elif conflict_strategy == 'merge':
                        if isinstance(value, dict) and isinstance(self._context[key], dict):
                            conflicts[key] = self._context[key]
                            self._context[key].update(value)
                        else:
                            conflicts[key] = self._context[key]
                            self._context[key] = value
                else:
                    self._context[key] = value
                    
        return conflicts
        
    def snapshot_context(self) -> int:
        """Take a snapshot of the current context state.
        
        Returns:
            The snapshot ID that can be used to restore this state
            
        TODO: Add tests for:
            - Taking snapshots
            - Restoring snapshots
            - Snapshot ID uniqueness
        """
        with self._context_lock:
            snapshot_id = len(self._context_history) + 1
            self._context_history[snapshot_id] = self._context.copy()
            return snapshot_id
            
    def restore_context(self, snapshot_id: int) -> bool:
        """Restore context from a snapshot.
        
        Args:
            snapshot_id: The ID of the snapshot to restore
            
        Returns:
            True if restored successfully, False if snapshot doesn't exist
            
        TODO: Add tests for:
            - Restoring existing snapshot
            - Restoring non-existent snapshot
            - Thread safety
        """
        with self._context_lock:
            if snapshot_id not in self._context_history:
                return False
                
            self._context = self._context_history[snapshot_id].copy()
            return True
            
    def clear_context(self) -> None:
        """Clear all context values.
        
        TODO: Add tests for:
            - Clearing context
            - Thread safety
        """
        with self._context_lock:
            self._context.clear()
            
    def log_error(self, message: str) -> None:
        """Log an error message.
        
        Args:
            message: The error message to log
        """
        print(f"ERROR: {message}")  # Simple print for now, can be enhanced with proper logging

    def execute(self, *args, **kwargs) -> Any:
        """Execute the plugin's main functionality.
        
        This method is required by BasePlugin and provides the main entry point
        for the plugin's functionality when used in a pipeline.

        Args:
            *args: Positional arguments
            **kwargs: Keyword arguments
            
        Returns:
            The processed result
        """
        # Default implementation - can be overridden in pipeline configuration
        operation = kwargs.get('operation', 'get')
        key = kwargs.get('key')
        value = kwargs.get('value')
        
        try:
            if operation == 'get':
                return self.get_context(key)
            elif operation == 'set':
                if key is None or value is None:
                    raise ValueError("Both key and value required for set operation")
                return self.set_context(key, value, kwargs.get('overwrite', True))
            elif operation == 'merge':
                if not isinstance(value, dict):
                    raise ValueError("Value must be a dictionary for merge operation")
                return self.merge_context(value, kwargs.get('conflict_strategy', 'overwrite'))
            elif operation == 'clear':
                self.clear_context()
                return True
            else:
                raise ValueError(f"Unsupported operation: {operation}")
        except Exception as e:
            self.log_error(f"ContextManager operation failed: {str(e)}")
            raise

    def merge_context(self, new_context: Dict[str, Any], conflict_strategy: str = 'overwrite') -> Dict[str, Any]:
        """Merge new context into existing context.
        
        Args:
            new_context: Dictionary of new context values
            conflict_strategy: How to handle conflicts ('overwrite', 'keep', 'merge')
            
        Returns:
            Dictionary of any conflicts that were handled
            
        Raises:
            ValueError: If conflict_strategy is invalid
        """
        if conflict_strategy not in ('overwrite', 'keep', 'merge'):
            raise ValueError(f"Invalid conflict strategy: {conflict_strategy}")
            
        conflicts = {}
        with self._context_lock:
            for key, value in new_context.items():
                if key in self._context:
                    if conflict_strategy == 'overwrite':
                        conflicts[key] = self._context[key]
                        self._context[key] = value
                    elif conflict_strategy == 'keep':
                        conflicts[key] = value
                    elif conflict_strategy == 'merge':
                        if isinstance(value, dict) and isinstance(self._context[key], dict):
                            conflicts[key] = self._context[key].copy()  # Return original dict
                            self._context[key] = {**self._context[key], **value}  # Merge dicts
                        else:
                            conflicts[key] = self._context[key]
                            self._context[key] = value
                else:
                    self._context[key] = value
                    
        return conflicts
            
    def execute_pipeline_step(self, *args, **kwargs) -> Any:
        """Execute a pipeline step - delegates to the execute() method.
        
        Args:
            *args: Positional arguments
            **kwargs: Keyword arguments
            
        Returns:
            The processed result
        """
        return self.execute(*args, **kwargs)
        
    def __str__(self) -> str:
        """String representation of the ContextManager.
        
        Returns:
            A string describing the current context state
        """
        with self._context_lock:
            return f"ContextManager(context_keys={len(self._context)}, snapshots={len(self._context_history)})"
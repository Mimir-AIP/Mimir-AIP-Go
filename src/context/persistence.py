import logging
from typing import Any, Dict, Optional, Type, TypeVar, Generic

from src.context.storage import StorageBackend

logger = logging.getLogger(__name__)
T = TypeVar('T', bound=StorageBackend)

class PersistenceManager(Generic[T]):
    """Manages persistence of context data to various backends."""
    
    def __init__(self, backend_class: Type[T], **backend_kwargs):
        """
        Initialize the persistence manager.
        
        Args:
            backend_class: The storage backend class to use
            **backend_kwargs: Arguments to pass to the backend constructor
        """
        self.backend = backend_class(**backend_kwargs)
        logger.info(f"Initialized persistence with backend: {backend_class.__name__}")
    
    def save_context(self, namespace: str, key: str, data: Dict[str, Any]) -> bool:
        """Save context data to the storage backend."""
        try:
            return self.backend.save(namespace, key, data)
        except Exception as e:
            logger.error(f"Failed to save context {namespace}.{key}: {e}")
            return False
    
    def load_context(self, namespace: str, key: Optional[str] = None) -> Any:
        """Load context data from the storage backend."""
        try:
            return self.backend.load(namespace, key)
        except Exception as e:
            logger.error(f"Failed to load context {namespace}.{key if key else '*'}: {e}")
            return None if key is not None else {}
    
    def delete_context(self, namespace: str, key: Optional[str] = None) -> bool:
        """Delete context data from the storage backend."""
        try:
            return self.backend.delete(namespace, key)
        except Exception as e:
            logger.error(f"Failed to delete context {namespace}.{key if key else '*'}: {e}")
            return False

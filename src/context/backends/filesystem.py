import json
import os
from pathlib import Path
from typing import Any, Dict, Optional, Union

from src.context.storage import StorageBackend

def save_to_file(path: Path, data: Any) -> None:
    """Atomically save data to a file with proper error handling."""
    # Create parent directory if it doesn't exist
    path.parent.mkdir(parents=True, exist_ok=True)
    
    # Create a temporary file in the same directory
    temp_path = path.with_suffix('.tmp')
    
    try:
        # Write to temp file
        with open(temp_path, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
        
        # Atomically replace the target file
        os.replace(temp_path, path)
        
    except Exception as e:
        # Clean up temp file if it exists
        if temp_path.exists():
            try:
                temp_path.unlink()
            except OSError:
                pass
        raise RuntimeError(f"Failed to save file {path}: {e}")

class FilesystemBackend(StorageBackend):
    """Filesystem-based storage backend."""
    
    def __init__(self, base_path: Union[str, Path] = "context_data"):
        """
        Initialize the filesystem backend.
        
        Args:
            base_path: Base directory for storing context data
        """
        self.base_path = Path(base_path).resolve()
        self.base_path.mkdir(parents=True, exist_ok=True)
    
    def _get_path(self, namespace: str, key: str) -> Path:
        """Get the filesystem path for a key."""
        # Sanitize namespace and key to prevent directory traversal
        safe_ns = "".join(c for c in namespace if c.isalnum() or c in ('-', '_')).rstrip('.')
        safe_key = "".join(c for c in key if c.isalnum() or c in ('-', '_')).rstrip('.')
        ns_path = self.base_path / safe_ns
        ns_path.mkdir(exist_ok=True)
        return ns_path / f"{safe_key}.json"
    
    def save(self, namespace: str, key: str, data: Dict[str, Any]) -> bool:
        """Save data to filesystem storage."""
        if not namespace or not key:
            raise ValueError("Namespace and key must be non-empty strings")
            
        path = self._get_path(namespace, key)
        try:
            save_to_file(path, data)
            return True
        except Exception as e:
            raise RuntimeError(f"Failed to save {namespace}.{key}: {e}")
    
    def load(self, namespace: str, key: Optional[str] = None) -> Any:
        """Load data from filesystem storage."""
        if not namespace:
            raise ValueError("Namespace cannot be empty")
            
        if key is None:
            # Load all keys in namespace
            safe_ns = "".join(c for c in namespace if c.isalnum() or c in ('-', '_')).rstrip('.')
            ns_path = self.base_path / safe_ns
            if not ns_path.exists():
                return {}
            
            result = {}
            for file_path in ns_path.glob("*.json"):
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        result[file_path.stem] = json.load(f)
                except (json.JSONDecodeError, OSError):
                    continue
            return result
        else:
            # Load specific key
            path = self._get_path(namespace, key)
            if not path.exists():
                return None
            with open(path, 'r', encoding='utf-8') as f:
                return json.load(f)
    
    def delete(self, namespace: str, key: Optional[str] = None) -> bool:
        """Delete data from filesystem storage."""
        if not namespace:
            raise ValueError("Namespace cannot be empty")
            
        if key is None:
            # Delete entire namespace
            safe_ns = "".join(c for c in namespace if c.isalnum() or c in ('-', '_')).rstrip('.')
            ns_path = self.base_path / safe_ns
            if ns_path.exists():
                for file_path in ns_path.glob("*.json"):
                    file_path.unlink()
                ns_path.rmdir()
            return True
        else:
            # Delete specific key
            path = self._get_path(namespace, key)
            if path.exists():
                path.unlink()
            return True

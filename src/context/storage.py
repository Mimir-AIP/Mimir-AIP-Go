"""
Binary data storage backends for ContextService.

This module provides an abstract base class for binary data storage backends
and a filesystem implementation for storing large binary objects.
"""

import os
import shutil
import tempfile
from abc import ABC, abstractmethod
from typing import BinaryIO, Optional
from pathlib import Path

class StorageBackendError(Exception):
    """Custom exception for storage backend errors."""
    pass

class StorageBackend(ABC):
    """
    Abstract base class for binary data storage backends.

    This class defines the interface for storage backends that can be used
    by the ContextService to store and retrieve large binary objects.
    """

    @abstractmethod
    def save(self, namespace: str, key: str, data: BinaryIO) -> None:
        """
        Save binary data to the storage backend.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.
            data: A file-like object containing the binary data.
        """
        pass

    @abstractmethod
    def load(self, namespace: str, key: str) -> BinaryIO:
        """
        Load binary data from the storage backend.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.

        Returns:
            BinaryIO: A file-like object containing the binary data.
        """
        pass

    @abstractmethod
    def delete(self, namespace: str, key: str) -> None:
        """
        Delete binary data from the storage backend.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.
        """
        pass

    @abstractmethod
    def exists(self, namespace: str, key: str) -> bool:
        """
        Check if binary data exists in the storage backend.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.

        Returns:
            bool: True if the data exists, False otherwise.
        """
        pass

class FilesystemBackend(StorageBackend):
    """
    Filesystem implementation of the StorageBackend interface.

    This class provides a filesystem-based storage backend for binary data.
    It uses a directory structure to organize data by namespace and key.
    """

    def __init__(self, base_path: str):
        """
        Initialize the FilesystemBackend with a base directory.

        Args:
            base_path: The base directory for storing binary data.
        """
        self.base_path = Path(base_path)
        self.base_path.mkdir(parents=True, exist_ok=True)

    def _get_file_path(self, namespace: str, key: str) -> Path:
        """
        Get the file path for a given namespace and key.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.

        Returns:
            Path: The file path for the binary data.
        
        Note:
            The key is sanitized to prevent directory traversal attacks by taking only
            the basename component of the provided key.
        """
        # Sanitize both namespace and key to prevent directory traversal
        safe_namespace = Path(namespace).name
        safe_key = Path(key).name
        return self.base_path / safe_namespace / safe_key

    def save(self, namespace: str, key: str, data: BinaryIO) -> None:
        """
        Save binary data to the filesystem.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.
            data: A file-like object containing the binary data.
        """
        file_path = self._get_file_path(namespace, key)
        file_path.parent.mkdir(parents=True, exist_ok=True)

        # Use a temporary file for atomic writes
        temp_file = tempfile.NamedTemporaryFile(delete=False, dir=file_path.parent)
        try:
            # Read from the input data in chunks to handle large files
            while True:
                chunk = data.read(1024 * 1024)  # 1MB chunks
                if not chunk:
                    break
                temp_file.write(chunk)

            # Close the temporary file and move it to the target location
            temp_file.close()
            os.replace(temp_file.name, str(file_path))
        except Exception as e:
            # Clean up in case of error
            if os.path.exists(temp_file.name):
                os.unlink(temp_file.name)
            raise StorageBackendError(f"Failed to save binary data: {e}") from e

    def load(self, namespace: str, key: str) -> BinaryIO:
        """
        Load binary data from the filesystem.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.

        Returns:
            BinaryIO: A file-like object containing the binary data.
        """
        file_path = self._get_file_path(namespace, key)
        if not file_path.exists():
            raise StorageBackendError(f"Binary data not found: {namespace}/{key}")

        # Return a file-like object for reading
        return open(file_path, 'rb')

    def delete(self, namespace: str, key: str) -> None:
        """
        Delete binary data from the filesystem.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.
        """
        file_path = self._get_file_path(namespace, key)
        if file_path.exists():
            try:
                os.unlink(file_path)
            except OSError as e:
                raise StorageBackendError(f"Failed to delete binary data: {e}") from e

    def exists(self, namespace: str, key: str) -> bool:
        """
        Check if binary data exists in the filesystem.

        Args:
            namespace: The namespace for the context data.
            key: The key for the context data.

        Returns:
            bool: True if the data exists, False otherwise.
        """
        file_path = self._get_file_path(namespace, key)
        return file_path.exists()

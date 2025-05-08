"""
Base Database Plugin for Mimir-AIP.

Provides common functionality for database connections and queries.
All specific database plugins should inherit from this base class.
"""

import os
import sys
import logging
from typing import Dict, Any, Optional, Union, List
from abc import ABC, abstractmethod

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from Plugins.BasePlugin import BasePlugin
from .migrations import MigrationManager

class BaseDatabase(BasePlugin, ABC):
    """Base class for all database input plugins"""
    
    plugin_type = "Input"
    
    def __init__(self):
        """Initialize base database plugin"""
        self.logger = logging.getLogger(__name__)
        self.connection = None
        self.default_timeout = 30
        self.migration_manager = None
    
    @abstractmethod
    def connect(self, config: Dict[str, Any]) -> None:
        """Establish database connection using provided configuration
        
        Args:
            config: Dictionary containing connection parameters
        
        Raises:
            ValueError: If required connection parameters are missing
            ConnectionError: If connection fails
        """
        pass
    
    @abstractmethod
    def disconnect(self) -> None:
        """Close the database connection"""
        pass
    
    @abstractmethod
    def execute_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a database query and return results
        
        Args:
            query: SQL query string
            params: Optional query parameters for parameterized queries
            
        Returns:
            List of dictionaries containing query results
            
        Raises:
            ValueError: If query is invalid
            RuntimeError: If query execution fails
        """
        pass
    
    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a database input pipeline step
        
        The config should contain:
        - connection: Database connection parameters
        - query: SQL query to execute
        - params: (optional) Query parameters
        - mode: 'query', 'table', or 'migrate'
        - table_name: (required if mode='table') Name of table to dump
        - migrations_dir: (required if mode='migrate') Directory containing migrations
        - target_version: (optional for migrate) Version to migrate to
        - rollback_steps: (optional for migrate) Number of migrations to roll back
        
        Returns:
            Dictionary containing query results in step_config["output"]
        """
        config = step_config.get("config", {})
        
        try:
            # Connect if not already connected
            if not self.connection:
                self.connect(config.get("connection", {}))
            
            mode = config.get("mode", "query")
            results = []
            
            if mode == "query":
                query = config.get("query")
                if not query:
                    raise ValueError("Query is required when mode='query'")
                params = config.get("params")
                results = self.execute_query(query, params)
                
            elif mode == "table":
                table_name = config.get("table_name")
                if not table_name:
                    raise ValueError("table_name is required when mode='table'")
                results = self.execute_query(f"SELECT * FROM {table_name}")
                
            elif mode == "migrate":
                migrations_dir = config.get("migrations_dir")
                if not migrations_dir:
                    raise ValueError("migrations_dir is required when mode='migrate'")
                
                # Initialize migration manager if needed
                if not self.migration_manager:
                    self.migration_manager = MigrationManager(migrations_dir, self)
                
                # Handle migration operations
                if "target_version" in config:
                    success = self.migration_manager.migrate(config["target_version"])
                elif "rollback_steps" in config:
                    success = self.migration_manager.rollback(config["rollback_steps"])
                else:
                    success = self.migration_manager.migrate()
                    
                results = [{"success": success}]
                
            else:
                raise ValueError(f"Invalid mode: {mode}. Must be 'query', 'table', or 'migrate'")
            
            return {step_config["output"]: results}
            
        except Exception as e:
            self.logger.error(f"Database operation failed: {str(e)}")
            raise
        
        finally:
            # Clean up connection if configured to do so
            if config.get("close_connection", False):
                self.disconnect()
    
    def create_migration(self, name: str, description: str = "", migrations_dir: str = None) -> str:
        """Create a new migration file
        
        Args:
            name: Name of the migration
            description: Optional description
            migrations_dir: Directory for migration files
            
        Returns:
            str: Path to created migration file
            
        Raises:
            ValueError: If migrations_dir is not set
        """
        if not migrations_dir and not self.migration_manager:
            raise ValueError("migrations_dir must be provided if migration_manager not initialized")
            
        if not self.migration_manager:
            self.migration_manager = MigrationManager(migrations_dir, self)
            
        return self.migration_manager.create_migration(name, description)
    
    def __del__(self):
        """Ensure connection is closed when object is destroyed"""
        self.disconnect()
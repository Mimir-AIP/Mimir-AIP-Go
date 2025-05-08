"""
SQLite Plugin for Mimir-AIP.

Provides database connection and query functionality for SQLite databases.
Uses Python's built-in sqlite3 module.

Example usage:
    plugin = SQLite()
    result = plugin.execute_pipeline_step({
        "config": {
            "connection": {
                "database": "mydb.sqlite3",
                "timeout": 30
            },
            "mode": "query",
            "query": "SELECT * FROM users WHERE age > :min_age",
            "params": {"min_age": 18},
            "close_connection": True
        },
        "output": "query_results"
    }, {})
"""

import os
import sys
import sqlite3
from typing import Dict, Any, Optional, List
import logging

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from .BaseDatabase import BaseDatabase

class SQLite(BaseDatabase):
    """SQLite database plugin"""

    def connect(self, config: Dict[str, Any]) -> None:
        """Connect to SQLite database
        
        Args:
            config: Dictionary containing:
                - database: Path to SQLite database file
                - timeout: Connection timeout in seconds (optional)
                - isolation_level: Transaction isolation level (optional)
        
        Raises:
            ValueError: If required connection parameters are missing
            ConnectionError: If connection fails
        """
        if 'database' not in config:
            raise ValueError("database path is required in connection config")
            
        try:
            conn_params = {
                'database': config['database'],
                'timeout': config.get('timeout', self.default_timeout)
            }
            
            # Add isolation level if specified
            if 'isolation_level' in config:
                conn_params['isolation_level'] = config['isolation_level']
            
            self.connection = sqlite3.connect(**conn_params)
            # Configure connection to return dictionaries
            self.connection.row_factory = sqlite3.Row
            
            self.logger.info(f"Connected to SQLite database: {config['database']}")
            
        except sqlite3.Error as e:
            self.logger.error(f"Failed to connect to SQLite: {str(e)}")
            raise ConnectionError(f"SQLite connection failed: {str(e)}")

    def disconnect(self) -> None:
        """Close the SQLite connection"""
        if self.connection:
            try:
                self.connection.close()
                self.connection = None
                self.logger.info("Disconnected from SQLite database")
            except sqlite3.Error as e:
                self.logger.error(f"Error disconnecting from SQLite: {str(e)}")

    def execute_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a SQLite query
        
        Args:
            query: SQL query string
            params: Optional dictionary of query parameters
            
        Returns:
            List of dictionaries containing query results
            
        Raises:
            ValueError: If query is invalid
            RuntimeError: If query execution fails
        """
        if not self.connection:
            raise RuntimeError("Not connected to database")
            
        try:
            cursor = self.connection.cursor()
            cursor.execute(query, params or {})
            
            # For SELECT queries, fetch results
            if cursor.description is not None:
                results = cursor.fetchall()
                # Convert sqlite3.Row objects to dictionaries
                return [dict(row) for row in results]
                
            # For other queries (INSERT, UPDATE, etc.), return affected rows
            self.connection.commit()
            affected = cursor.rowcount
            cursor.close()
            return [{"affected_rows": affected}]
                
        except sqlite3.Error as e:
            self.logger.error(f"Query execution failed: {str(e)}")
            self.connection.rollback()
            raise RuntimeError(f"Query execution failed: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = SQLite()
    
    # Test configuration
    test_config = {
        "plugin": "SQLite",
        "config": {
            "connection": {
                "database": "test.db"
            },
            "mode": "query",
            "query": """
            CREATE TABLE IF NOT EXISTS test_table (
                id INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                age INTEGER
            );
            INSERT INTO test_table (name, age) VALUES (:name, :age);
            SELECT * FROM test_table;
            """,
            "params": {"name": "Test User", "age": 25},
            "close_connection": True
        },
        "output": "results"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Query returned {len(result['results'])} rows")
        for row in result['results']:  # Print all results
            print(row)
    except Exception as e:
        print(f"Test failed: {str(e)}")
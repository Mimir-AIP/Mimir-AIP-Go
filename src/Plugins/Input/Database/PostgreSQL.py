"""
PostgreSQL Plugin for Mimir-AIP.

Provides database connection and query functionality for PostgreSQL databases.
Requires psycopg2-binary package.

Example usage:
    plugin = PostgreSQL()
    result = plugin.execute_pipeline_step({
        "config": {
            "connection": {
                "host": "localhost",
                "port": 5432,
                "database": "mydb",
                "user": "myuser",
                "password": "mypassword"
            },
            "mode": "query",
            "query": "SELECT * FROM users WHERE age > %(min_age)s",
            "params": {"min_age": 18},
            "close_connection": True
        },
        "output": "query_results"
    }, {})
"""

import os
import sys
from typing import Dict, Any, Optional, List
import psycopg2
from psycopg2.extras import RealDictCursor
import logging

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from .BaseDatabase import BaseDatabase

class PostgreSQL(BaseDatabase):
    """PostgreSQL database plugin"""

    def connect(self, config: Dict[str, Any]) -> None:
        """Connect to PostgreSQL database
        
        Args:
            config: Dictionary containing:
                - host: Database host
                - port: Database port
                - database: Database name
                - user: Username
                - password: Password
                - sslmode: SSL mode (optional)
                - timeout: Connection timeout in seconds (optional)
        
        Raises:
            ValueError: If required connection parameters are missing
            ConnectionError: If connection fails
        """
        required = ['host', 'database', 'user', 'password']
        missing = [param for param in required if param not in config]
        if missing:
            raise ValueError(f"Missing required connection parameters: {', '.join(missing)}")
            
        try:
            conn_params = {
                'host': config['host'],
                'port': config.get('port', 5432),
                'dbname': config['database'],
                'user': config['user'],
                'password': config['password'],
                'sslmode': config.get('sslmode', 'prefer'),
                'connect_timeout': config.get('timeout', self.default_timeout)
            }
            
            self.connection = psycopg2.connect(**conn_params)
            self.logger.info(f"Connected to PostgreSQL database: {config['database']} on {config['host']}")
            
        except psycopg2.Error as e:
            self.logger.error(f"Failed to connect to PostgreSQL: {str(e)}")
            raise ConnectionError(f"PostgreSQL connection failed: {str(e)}")

    def disconnect(self) -> None:
        """Close the PostgreSQL connection"""
        if self.connection:
            try:
                self.connection.close()
                self.connection = None
                self.logger.info("Disconnected from PostgreSQL database")
            except psycopg2.Error as e:
                self.logger.error(f"Error disconnecting from PostgreSQL: {str(e)}")

    def execute_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a PostgreSQL query
        
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
            # Use RealDictCursor to get results as dictionaries
            with self.connection.cursor(cursor_factory=RealDictCursor) as cursor:
                cursor.execute(query, params)
                
                # For SELECT queries, fetch results
                if cursor.description is not None:
                    results = cursor.fetchall()
                    # Convert results to plain dictionaries
                    return [dict(row) for row in results]
                    
                # For other queries (INSERT, UPDATE, etc.), return affected rows
                self.connection.commit()
                return [{"affected_rows": cursor.rowcount}]
                
        except psycopg2.Error as e:
            self.logger.error(f"Query execution failed: {str(e)}")
            self.connection.rollback()
            raise RuntimeError(f"Query execution failed: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = PostgreSQL()
    
    # Test configuration
    test_config = {
        "plugin": "PostgreSQL",
        "config": {
            "connection": {
                "host": "localhost",
                "port": 5432,
                "database": "testdb",
                "user": "testuser",
                "password": "testpass"
            },
            "mode": "query",
            "query": "SELECT * FROM test_table LIMIT 5",
            "close_connection": True
        },
        "output": "results"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Query returned {len(result['results'])} rows")
        for row in result['results'][:3]:  # Print first 3 rows
            print(row)
    except Exception as e:
        print(f"Test failed: {str(e)}")
"""
MySQL/MariaDB Plugin for Mimir-AIP.

Provides database connection and query functionality for MySQL and MariaDB databases.
Requires mysql-connector-python package.

Example usage:
    plugin = MySQL()
    result = plugin.execute_pipeline_step({
        "config": {
            "connection": {
                "host": "localhost",
                "port": 3306,
                "database": "mydb",
                "user": "myuser",
                "password": "mypassword",
                "charset": "utf8mb4"
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
import mysql.connector
import logging

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from .BaseDatabase import BaseDatabase

class MySQL(BaseDatabase):
    """MySQL/MariaDB database plugin"""

    def connect(self, config: Dict[str, Any]) -> None:
        """Connect to MySQL/MariaDB database
        
        Args:
            config: Dictionary containing:
                - host: Database host
                - port: Database port
                - database: Database name
                - user: Username
                - password: Password
                - charset: Character set (optional, default utf8mb4)
                - ssl_ca: Path to SSL CA certificate (optional)
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
                'port': config.get('port', 3306),
                'database': config['database'],
                'user': config['user'],
                'password': config['password'],
                'charset': config.get('charset', 'utf8mb4'),
                'connection_timeout': config.get('timeout', self.default_timeout)
            }
            
            # Add SSL if configured
            if 'ssl_ca' in config:
                conn_params['ssl_ca'] = config['ssl_ca']
                conn_params['ssl_verify_cert'] = True
            
            self.connection = mysql.connector.connect(**conn_params)
            self.logger.info(f"Connected to MySQL database: {config['database']} on {config['host']}")
            
        except mysql.connector.Error as e:
            self.logger.error(f"Failed to connect to MySQL: {str(e)}")
            raise ConnectionError(f"MySQL connection failed: {str(e)}")

    def disconnect(self) -> None:
        """Close the MySQL connection"""
        if self.connection:
            try:
                self.connection.close()
                self.connection = None
                self.logger.info("Disconnected from MySQL database")
            except mysql.connector.Error as e:
                self.logger.error(f"Error disconnecting from MySQL: {str(e)}")

    def execute_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a MySQL query
        
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
            cursor = self.connection.cursor(dictionary=True)
            cursor.execute(query, params)
            
            # For SELECT queries, fetch results
            if cursor.description is not None:
                results = cursor.fetchall()
                cursor.close()
                return results
                
            # For other queries (INSERT, UPDATE, etc.), return affected rows
            self.connection.commit()
            affected = cursor.rowcount
            cursor.close()
            return [{"affected_rows": affected}]
                
        except mysql.connector.Error as e:
            self.logger.error(f"Query execution failed: {str(e)}")
            self.connection.rollback()
            raise RuntimeError(f"Query execution failed: {str(e)}")


if __name__ == "__main__":
    # Test the plugin
    plugin = MySQL()
    
    # Test configuration
    test_config = {
        "plugin": "MySQL",
        "config": {
            "connection": {
                "host": "localhost",
                "port": 3306,
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
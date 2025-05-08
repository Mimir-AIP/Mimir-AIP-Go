"""
MongoDB Plugin for Mimir-AIP.

Provides database connection and query functionality for MongoDB databases.
Requires pymongo package.

Example usage:
    plugin = MongoDB()
    result = plugin.execute_pipeline_step({
        "config": {
            "connection": {
                "host": "mongodb://localhost:27017",
                "database": "mydb",
                "username": "myuser",
                "password": "mypassword",
                "auth_source": "admin"
            },
            "mode": "query",
            "collection": "users",
            "query": {"age": {"$gt": 18}},
            "projection": {"_id": 0, "name": 1, "age": 1},
            "sort": [("age", -1)],
            "limit": 100,
            "close_connection": True
        },
        "output": "query_results"
    }, {})
"""

import os
import sys
from typing import Dict, Any, Optional, List, Union
from pymongo import MongoClient
from pymongo.errors import PyMongoError
import logging
import json

# Add src directory to Python path for imports
src_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
sys.path.insert(0, src_dir)

from .BaseDatabase import BaseDatabase

class MongoDB(BaseDatabase):
    """MongoDB database plugin"""

    def connect(self, config: Dict[str, Any]) -> None:
        """Connect to MongoDB database
        
        Args:
            config: Dictionary containing:
                - host: MongoDB connection string or host address
                - database: Database name
                - username: Username (optional)
                - password: Password (optional)
                - auth_source: Authentication database (optional, default: admin)
                - timeout: Connection timeout in ms (optional)
        
        Raises:
            ValueError: If required connection parameters are missing
            ConnectionError: If connection fails
        """
        required = ['host', 'database']
        missing = [param for param in required if param not in config]
        if missing:
            raise ValueError(f"Missing required connection parameters: {', '.join(missing)}")
            
        try:
            # Construct connection parameters
            conn_params = {
                'host': config['host'],
                'serverSelectionTimeoutMS': config.get('timeout', self.default_timeout * 1000)
            }
            
            # Add authentication if provided
            if 'username' in config and 'password' in config:
                conn_params.update({
                    'username': config['username'],
                    'password': config['password'],
                    'authSource': config.get('auth_source', 'admin')
                })
            
            # Create client and test connection
            self.client = MongoClient(**conn_params)
            self.client.admin.command('ping')  # Test connection
            
            # Store database reference
            self.connection = self.client[config['database']]
            self.logger.info(f"Connected to MongoDB database: {config['database']} on {config['host']}")
            
        except PyMongoError as e:
            self.logger.error(f"Failed to connect to MongoDB: {str(e)}")
            raise ConnectionError(f"MongoDB connection failed: {str(e)}")

    def disconnect(self) -> None:
        """Close the MongoDB connection"""
        if hasattr(self, 'client'):
            try:
                self.client.close()
                self.connection = None
                self.client = None
                self.logger.info("Disconnected from MongoDB database")
            except PyMongoError as e:
                self.logger.error(f"Error disconnecting from MongoDB: {str(e)}")

    def execute_query(self, query: str, params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """Execute a MongoDB query
        
        For MongoDB, the query parameter is actually a dictionary defining the operation.
        The params dictionary contains additional query parameters like collection, projection, etc.
        
        Args:
            query: MongoDB query/operation dictionary (will be parsed from string)
            params: Dictionary containing:
                - collection: Collection name
                - projection: Fields to return
                - sort: Sort specification
                - limit: Maximum number of documents
                - skip: Number of documents to skip
                
        Returns:
            List of dictionaries containing query results
            
        Raises:
            ValueError: If query is invalid
            RuntimeError: If query execution fails
        """
        if not self.connection:
            raise RuntimeError("Not connected to database")
            
        try:
            # Parse query if it's a string
            if isinstance(query, str):
                query = json.loads(query)
                
            # Get parameters
            params = params or {}
            collection_name = params.get('collection')
            if not collection_name:
                raise ValueError("Collection name is required in params")
                
            collection = self.connection[collection_name]
            
            # Build query options
            options = {}
            if 'projection' in params:
                options['projection'] = params['projection']
            if 'sort' in params:
                options['sort'] = params['sort']
            if 'limit' in params:
                options['limit'] = params['limit']
            if 'skip' in params:
                options['skip'] = params['skip']
            
            # Execute query
            cursor = collection.find(query, **options)
            
            # Convert cursor to list of dictionaries
            results = []
            for doc in cursor:
                # Convert ObjectId to string
                if '_id' in doc:
                    doc['_id'] = str(doc['_id'])
                results.append(doc)
                
            return results
                
        except PyMongoError as e:
            self.logger.error(f"Query execution failed: {str(e)}")
            raise RuntimeError(f"Query execution failed: {str(e)}")


    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a MongoDB pipeline step
        
        Extends the base implementation to handle MongoDB-specific operations:
        - Supports both query and aggregation modes
        - Handles MongoDB-specific query parameters
        - Allows collection operations
        
        Args:
            step_config: Step configuration containing:
                - mode: 'query', 'aggregate', or 'collection' 
                - collection: Collection name
                - query/pipeline: Query filter or aggregation pipeline
                - projection: Fields to return
                - sort: Sort specification
                - limit: Maximum documents
                - skip: Number of documents to skip
                
        Returns:
            Dictionary containing query results
        """
        config = step_config.get("config", {})
        
        try:
            # Connect if not already connected
            if not self.connection:
                self.connect(config.get("connection", {}))
            
            mode = config.get("mode", "query")
            collection = config.get("collection")
            if not collection:
                raise ValueError("collection is required in config")
            
            results = []
            
            if mode == "query":
                # Handle query mode
                query = config.get("query", {})
                params = {
                    "collection": collection,
                    "projection": config.get("projection"),
                    "sort": config.get("sort"),
                    "limit": config.get("limit"),
                    "skip": config.get("skip")
                }
                results = self.execute_query(query, params)
                
            elif mode == "aggregate":
                # Handle aggregation pipeline
                pipeline = config.get("pipeline", [])
                if not isinstance(pipeline, list):
                    raise ValueError("pipeline must be a list of aggregation stages")
                    
                try:
                    coll = self.connection[collection]
                    cursor = coll.aggregate(pipeline)
                    results = list(cursor)
                    # Convert ObjectIds to strings
                    for doc in results:
                        if '_id' in doc:
                            doc['_id'] = str(doc['_id'])
                except PyMongoError as e:
                    raise RuntimeError(f"Aggregation failed: {str(e)}")
                    
            elif mode == "collection":
                # Return collection info/stats
                try:
                    coll = self.connection[collection]
                    results = [{
                        "name": collection,
                        "count": coll.count_documents({}),
                        "stats": self.connection.command("collstats", collection)
                    }]
                except PyMongoError as e:
                    raise RuntimeError(f"Failed to get collection info: {str(e)}")
                    
            else:
                raise ValueError(f"Invalid mode: {mode}. Must be 'query', 'aggregate', or 'collection'")
            
            return {step_config["output"]: results}
            
        except Exception as e:
            self.logger.error(f"MongoDB operation failed: {str(e)}")
            raise
        
        finally:
            # Clean up connection if configured to do so
            if config.get("close_connection", False):
                self.disconnect()


if __name__ == "__main__":
    # Test the plugin
    plugin = MongoDB()
    
    # Test configuration
    test_config = {
        "plugin": "MongoDB",
        "config": {
            "connection": {
                "host": "mongodb://localhost:27017",
                "database": "testdb"
            },
            "mode": "query",
            "collection": "test_collection",
            "query": {"status": "active"},
            "projection": {"_id": 0, "name": 1, "status": 1},
            "sort": [("name", 1)],
            "limit": 5,
            "close_connection": True
        },
        "output": "results"
    }
    
    try:
        result = plugin.execute_pipeline_step(test_config, {})
        print(f"Query returned {len(result['results'])} documents")
        for doc in result['results']:  # Print all results
            print(doc)
    except Exception as e:
        print(f"Test failed: {str(e)}")
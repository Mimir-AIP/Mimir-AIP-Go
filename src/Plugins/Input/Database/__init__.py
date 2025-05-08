"""
Database Input Plugin Package for Mimir-AIP.

Provides plugins for connecting to and querying various types of databases:
- PostgreSQL
- MySQL/MariaDB
- MongoDB
- SQLite

Example usage:
    from Plugins.Input.Database import PostgreSQL, MySQL, MongoDB, SQLite
    
    # Use PostgreSQL plugin
    pg = PostgreSQL()
    result = pg.execute_pipeline_step({
        "config": {
            "connection": {
                "host": "localhost",
                "database": "mydb",
                "user": "user",
                "password": "pass"
            },
            "query": "SELECT * FROM mytable"
        },
        "output": "query_results"
    }, {})
"""

from .BaseDatabase import BaseDatabase
from .PostgreSQL import PostgreSQL
from .MySQL import MySQL
from .MongoDB import MongoDB
from .SQLite import SQLite

__all__ = ['BaseDatabase', 'PostgreSQL', 'MySQL', 'MongoDB', 'SQLite']
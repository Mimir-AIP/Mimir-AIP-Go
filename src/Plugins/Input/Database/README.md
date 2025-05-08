# Database Input Plugins for Mimir-AIP

This package provides database input plugins for Mimir-AIP, allowing you to connect to and query various types of databases.

## Supported Databases

- PostgreSQL
- MySQL/MariaDB 
- MongoDB (NoSQL)
- SQLite (File-based)

## Installation

The database plugins require additional Python packages. Install them using:

```bash
pip install -r requirements_db.txt
```

## Basic Usage

### SQL Databases (PostgreSQL, MySQL, SQLite)

```python
from Plugins.Input.Database import PostgreSQL

# Create plugin instance
db = PostgreSQL()

# Execute a query
result = db.execute_pipeline_step({
    "config": {
        "connection": {
            "host": "localhost",
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

# Access results
users = result["query_results"]
```

### MongoDB (NoSQL)

```python
from Plugins.Input.Database import MongoDB

# Create plugin instance
db = MongoDB()

# Execute a query
result = db.execute_pipeline_step({
    "config": {
        "connection": {
            "host": "mongodb://localhost:27017",
            "database": "mydb"
        },
        "mode": "query",
        "collection": "users",
        "query": {"age": {"$gt": 18}},
        "projection": {"_id": 0, "name": 1, "age": 1},
        "close_connection": True
    },
    "output": "query_results"
}, {})
```

## Using Migrations

The database plugins include a migration system to manage database schema changes. Migrations are defined in YAML files:

```yaml
version: 1
name: create_users
description: Create initial users table
up: |
  CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE
  );
down: |
  DROP TABLE IF EXISTS users;
```

### Running Migrations

```python
# Run all pending migrations
result = db.execute_pipeline_step({
    "config": {
        "connection": {
            "host": "localhost",
            "database": "mydb",
            "user": "myuser",
            "password": "mypassword"
        },
        "mode": "migrate",
        "migrations_dir": "migrations",
        "close_connection": True
    },
    "output": "migration_result"
}, {})

# Migrate to specific version
result = db.execute_pipeline_step({
    "config": {
        "mode": "migrate",
        "migrations_dir": "migrations",
        "target_version": 3
    },
    "output": "migration_result"
}, {})

# Rollback last migration
result = db.execute_pipeline_step({
    "config": {
        "mode": "migrate",
        "migrations_dir": "migrations",
        "rollback_steps": 1
    },
    "output": "migration_result"
}, {})
```

### Creating New Migrations

```python
db = PostgreSQL()
migration_path = db.create_migration(
    name="add_email_to_users",
    description="Add email column to users table",
    migrations_dir="migrations"
)
```

Then edit the created migration file to add your schema changes.

## Connection Parameters

### PostgreSQL
```python
{
    "host": "localhost",      # Required
    "port": 5432,            # Optional, default: 5432
    "database": "mydb",      # Required
    "user": "myuser",        # Required
    "password": "mypass",    # Required
    "sslmode": "prefer",     # Optional
    "timeout": 30            # Optional, connection timeout in seconds
}
```

### MySQL/MariaDB
```python
{
    "host": "localhost",      # Required
    "port": 3306,            # Optional, default: 3306
    "database": "mydb",      # Required
    "user": "myuser",        # Required
    "password": "mypass",    # Required
    "charset": "utf8mb4",    # Optional
    "ssl_ca": "/path/to/ca"  # Optional, path to SSL CA certificate
}
```

### MongoDB
```python
{
    "host": "mongodb://localhost:27017",  # Required
    "database": "mydb",                   # Required
    "username": "myuser",                 # Optional
    "password": "mypass",                 # Optional
    "auth_source": "admin",               # Optional
    "timeout": 30000                      # Optional, in milliseconds
}
```

### SQLite
```python
{
    "database": "mydb.sqlite3",     # Required, path to database file
    "timeout": 30,                  # Optional, connection timeout
    "isolation_level": None         # Optional, transaction isolation level
}
```

## Best Practices

1. Always use parameterized queries to prevent SQL injection.
2. Close connections when done using `close_connection: true`.
3. Use migrations for database schema changes.
4. Keep migration files small and focused.
5. Always include down migrations for rollback capability.
6. Use appropriate indexes for query optimization.
7. Handle database errors appropriately in your pipeline.
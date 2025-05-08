"""
Database migrations manager for Mimir-AIP database plugins.

Provides functionality to manage database schema versions and run migrations.
Migrations can be defined in YAML files with the following structure:

version: 1
name: initial_schema
description: Create initial database schema
up: |
  CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE
  );
down: |
  DROP TABLE users;
"""

import os
import yaml
import logging
from typing import Dict, Any, List, Optional
from datetime import datetime

class MigrationManager:
    """Manages database migrations across different database types"""
    
    def __init__(self, migrations_dir: str, db_plugin: Any):
        """Initialize migration manager
        
        Args:
            migrations_dir: Directory containing migration YAML files
            db_plugin: Database plugin instance to use for migrations
        """
        self.migrations_dir = migrations_dir
        self.db_plugin = db_plugin
        self.logger = logging.getLogger(__name__)
        
        # Ensure migrations directory exists
        os.makedirs(migrations_dir, exist_ok=True)
        
    def _get_migrations_table_sql(self) -> str:
        """Get SQL to create migrations tracking table based on database type"""
        db_type = self.db_plugin.__class__.__name__.lower()
        
        if db_type == 'postgresql':
            return """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                description TEXT
            );
            """
        elif db_type == 'mysql':
            return """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version INTEGER PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                description TEXT
            );
            """
        else:  # SQLite and others
            return """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                description TEXT
            );
            """
            
    def init_migrations(self) -> None:
        """Initialize migrations system by creating the tracking table"""
        try:
            self.db_plugin.execute_query(self._get_migrations_table_sql())
            self.logger.info("Initialized migrations tracking table")
        except Exception as e:
            self.logger.error(f"Failed to initialize migrations: {str(e)}")
            raise
            
    def get_applied_versions(self) -> List[int]:
        """Get list of already applied migration versions"""
        try:
            results = self.db_plugin.execute_query(
                "SELECT version FROM schema_migrations ORDER BY version"
            )
            return [row['version'] for row in results]
        except Exception as e:
            self.logger.error(f"Failed to get applied migrations: {str(e)}")
            return []
            
    def load_migration(self, filename: str) -> Optional[Dict[str, Any]]:
        """Load a migration from a YAML file"""
        try:
            path = os.path.join(self.migrations_dir, filename)
            with open(path, 'r') as f:
                migration = yaml.safe_load(f)
                
            required = ['version', 'name', 'up']
            if not all(key in migration for key in required):
                self.logger.error(f"Migration {filename} missing required fields")
                return None
                
            return migration
        except Exception as e:
            self.logger.error(f"Failed to load migration {filename}: {str(e)}")
            return None
            
    def load_migrations(self) -> List[Dict[str, Any]]:
        """Load all migration files in version order"""
        migrations = []
        
        try:
            for filename in sorted(os.listdir(self.migrations_dir)):
                if not filename.endswith('.yml'):
                    continue
                    
                migration = self.load_migration(filename)
                if migration:
                    migrations.append(migration)
                    
            return sorted(migrations, key=lambda m: m['version'])
            
        except Exception as e:
            self.logger.error(f"Failed to load migrations: {str(e)}")
            return []
            
    def apply_migration(self, migration: Dict[str, Any], record: bool = True) -> bool:
        """Apply a single migration
        
        Args:
            migration: Migration dictionary with version, name, and up SQL
            record: Whether to record this migration in schema_migrations table
            
        Returns:
            bool: True if migration was applied successfully
        """
        try:
            # Execute migration SQL
            self.db_plugin.execute_query(migration['up'])
            
            # Record migration if requested
            if record:
                self.db_plugin.execute_query(
                    """
                    INSERT INTO schema_migrations 
                    (version, name, description) 
                    VALUES (:version, :name, :description)
                    """,
                    {
                        'version': migration['version'],
                        'name': migration['name'],
                        'description': migration.get('description', '')
                    }
                )
                
            self.logger.info(f"Applied migration {migration['version']}: {migration['name']}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to apply migration {migration['version']}: {str(e)}")
            return False
            
    def revert_migration(self, migration: Dict[str, Any]) -> bool:
        """Revert a single migration
        
        Args:
            migration: Migration dictionary with version, name, and down SQL
            
        Returns:
            bool: True if migration was reverted successfully
        """
        if 'down' not in migration:
            self.logger.error(f"Migration {migration['version']} has no down migration")
            return False
            
        try:
            # Execute down migration
            self.db_plugin.execute_query(migration['down'])
            
            # Remove migration record
            self.db_plugin.execute_query(
                "DELETE FROM schema_migrations WHERE version = :version",
                {'version': migration['version']}
            )
            
            self.logger.info(f"Reverted migration {migration['version']}: {migration['name']}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to revert migration {migration['version']}: {str(e)}")
            return False
            
    def migrate(self, target_version: Optional[int] = None) -> bool:
        """Run all pending migrations or up to target_version
        
        Args:
            target_version: Optional version to migrate to. If None, runs all pending
            
        Returns:
            bool: True if all migrations were successful
        """
        try:
            # Initialize if needed
            self.init_migrations()
            
            # Get current state
            applied = set(self.get_applied_versions())
            migrations = self.load_migrations()
            
            if not migrations:
                self.logger.info("No migrations found")
                return True
                
            if target_version is None:
                target_version = migrations[-1]['version']
                
            # Apply missing migrations up to target
            success = True
            for migration in migrations:
                version = migration['version']
                
                # Skip if already applied or beyond target
                if version in applied or version > target_version:
                    continue
                    
                if not self.apply_migration(migration):
                    success = False
                    break
                    
            return success
            
        except Exception as e:
            self.logger.error(f"Migration failed: {str(e)}")
            return False
            
    def rollback(self, steps: int = 1) -> bool:
        """Rollback the last n migrations
        
        Args:
            steps: Number of migrations to roll back
            
        Returns:
            bool: True if all rollbacks were successful
        """
        try:
            # Get current state
            migrations = {m['version']: m for m in self.load_migrations()}
            applied = self.get_applied_versions()
            
            # Nothing to do if no migrations applied
            if not applied:
                self.logger.info("No migrations to roll back")
                return True
                
            # Rollback requested number of migrations
            success = True
            for version in reversed(applied[-steps:]):
                if version not in migrations:
                    self.logger.error(f"Missing migration file for version {version}")
                    success = False
                    break
                    
                if not self.revert_migration(migrations[version]):
                    success = False
                    break
                    
            return success
            
        except Exception as e:
            self.logger.error(f"Rollback failed: {str(e)}")
            return False
            
    def create_migration(self, name: str, description: str = "") -> str:
        """Create a new migration file
        
        Args:
            name: Name of the migration (will be used in filename)
            description: Optional description of what the migration does
            
        Returns:
            str: Path to created migration file
        """
        try:
            # Get next version number
            applied = self.get_applied_versions()
            next_version = (max(applied) + 1) if applied else 1
            
            # Create migration file
            timestamp = datetime.now().strftime('%Y%m%d%H%M%S')
            filename = f"{timestamp}_{next_version}_{name}.yml"
            path = os.path.join(self.migrations_dir, filename)
            
            migration = {
                'version': next_version,
                'name': name,
                'description': description,
                'up': '-- Add your UP migration SQL here',
                'down': '-- Add your DOWN migration SQL here (optional)'
            }
            
            with open(path, 'w') as f:
                yaml.dump(migration, f, sort_keys=False, indent=2)
                
            self.logger.info(f"Created migration file: {filename}")
            return path
            
        except Exception as e:
            self.logger.error(f"Failed to create migration: {str(e)}")
            raise
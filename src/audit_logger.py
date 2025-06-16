"""
Audit logging system for tracking changes and access to pipeline data.

Features:
- Change history tracking
- Source/actor identification
- Timestamped operations
- Configurable output formats
"""

import logging
from datetime import datetime
from typing import Dict, Any
import threading

class AuditLogger:
    """Core audit logging service for pipeline operations"""
    
    def __init__(self, log_file: str = "audit.log"):
        """
        Initialize audit logger with output file.
        
        Args:
            log_file: Path to audit log file
        """
        self.logger = logging.getLogger('mimir.audit')
        self.logger.setLevel(logging.INFO)
        
        # Create file handler
        handler = logging.FileHandler(log_file)
        handler.setFormatter(logging.Formatter(
            '%(asctime)s | %(levelname)s | %(message)s'
        ))
        self.logger.addHandler(handler)
        
        # Ensure thread-safe operation
        self.lock = threading.RLock()
    
    def log_operation(
        self,
        operation: str,
        entity_type: str,
        entity_id: str,
        actor: str = "system",
        old_value: Optional[Any] = None,
        new_value: Optional[Any] = None,
        metadata: Dict[str, Any] = None
    ) -> None:
        """
        Log an auditable operation with full context.
        
        Args:
            operation: CRUD operation (create/read/update/delete)
            entity_type: Type of entity being modified
            entity_id: Unique identifier for the entity
            actor: Who performed the action
            metadata: Additional context about the operation
        """
        with self.lock:
            log_entry = {
                "timestamp": datetime.utcnow().isoformat(),
                "operation": operation,
                "entity_type": entity_type,
                "entity_id": entity_id,
                "actor": actor,
                "metadata": metadata or {}
            }
            if operation == "update":
                log_entry["old_value"] = old_value
                log_entry["new_value"] = new_value
            self.logger.info(str(log_entry))
    
    def get_log_reader(self):
        """Get an interface for reading audit logs"""
        return AuditLogReader(self.logger.handlers[0].baseFilename)

class AuditLogReader:
    """Helper class for reading and querying audit logs"""
    
    def __init__(self, log_file: str):
        self.log_file = log_file
    
    def query(self, filters: Dict[str, Any] = None):
        """
        Query audit logs with optional filters
        """
        # Implementation would go here
        pass
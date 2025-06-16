import logging
import re
from typing import Dict, Any, List, Optional
from pydantic import BaseModel, Field

logger = logging.getLogger(__name__)

class AccessPolicy(BaseModel):
    """
    Defines an access control policy for internal system/pipeline roles.
    """
    role: str = Field(..., description="The internal role to which this policy applies (e.g., 'system', 'pipeline_executor', 'admin_tool').")
    resource_pattern: str = Field(..., description="Regex pattern for the context key/path (e.g., 'data.*', 'pipeline_status.current_step').")
    actions: List[str] = Field(..., description="List of allowed actions (e.g., 'read', 'write', 'delete', 'snapshot', 'restore').")
    effect: str = Field("allow", description="Effect of the policy: 'allow' or 'deny'.")

class PermissionManager:
    """
    Manages and enforces access control policies for internal system/pipeline roles.
    """
    def __init__(self, policies: List[Dict[str, Any]] = None, enabled: bool = True):
        """
        Initializes the PermissionManager with a list of access policies.

        Args:
            policies (List[Dict[str, Any]]): A list of policy dictionaries.
            enabled (bool): If False, all access checks will pass (for backwards compatibility).
        """
        self.enabled = enabled
        self.policies: List[AccessPolicy] = []
        if policies:
            for policy_data in policies:
                try:
                    self.policies.append(AccessPolicy(**policy_data))
                except Exception as e:
                    logger.error(f"Failed to load access policy: {policy_data}. Error: {e}")
        logger.info(f"PermissionManager initialized. Enabled: {self.enabled}, Loaded policies: {len(self.policies)}")

    def check_permission(self, role: str, resource: str, action: str) -> bool:
        """
        Checks if the given internal role has permission to perform the action on the resource.

        Args:
            role (str): The internal role of the entity requesting access.
            resource (str): The context key or path being accessed (e.g., 'namespace.key').
            action (str): The action being performed ('read', 'write', 'delete', 'snapshot', 'restore').

        Returns:
            bool: True if permission is granted, False otherwise.
        """
        if not self.enabled:
            logger.debug(f"PermissionManager is disabled. Allowing access for role '{role}' on resource '{resource}' with action '{action}'.")
            return True

        # Deny-over-allow logic: if any deny policy matches, access is denied.
        # Otherwise, if any allow policy matches, access is granted.
        # If no policies match, access is denied by default.

        # Check deny policies first
        for policy in self.policies:
            if policy.effect == "deny" and policy.role == role and action in policy.actions:
                if re.fullmatch(policy.resource_pattern, resource):
                    logger.warning(f"Access denied for role '{role}' on resource '{resource}' with action '{action}' by deny policy: {policy}")
                    return False
        
        # Check allow policies
        for policy in self.policies:
            if policy.effect == "allow" and policy.role == role and action in policy.actions:
                if re.fullmatch(policy.resource_pattern, resource):
                    logger.debug(f"Access granted for role '{role}' on resource '{resource}' with action '{action}' by policy: {policy}")
                    return True
        
        logger.warning(f"Access denied: No matching allow policy found for role '{role}' on resource '{resource}' with action '{action}'.")
        return False
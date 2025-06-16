from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Union

@dataclass
class ConfigNode:
    """Represents the configuration for a plugin step."""
    data: Dict[str, Any] = field(default_factory=dict)

@dataclass
class StepNode:
    """Represents a single step within a pipeline."""
    name: str
    type: str = "plugin"  # plugin, conditional, or jump
    plugin: Optional[str] = None
    config: Optional[ConfigNode] = None
    input: Optional[str] = None
    output: Optional[str] = None
    label: Optional[str] = None
    iterate: Optional[Union[str, Dict[str, Union[str, Optional[str]]]]] = None
    use_plugin_manager: Optional[bool] = None
    steps: List["StepNode"] = field(default_factory=list)  # Nested steps for iteration or conditional branches
    condition: Optional[Dict[str, Any]] = None
    path: Optional[str] = None
    value: Optional[Any] = None
    overwrite: Optional[bool] = None
    create_if_missing: Optional[bool] = None
    source: Optional[str] = None
    destination: Optional[str] = None

@dataclass
class PipelineNode:
    """Represents a single pipeline definition."""
    name: str
    description: str
    steps: List[StepNode]
    version: Optional[str] = None
    enabled: bool = False
    execution_mode: str = "single"
    plugin_manager_required: Optional[bool] = None
    schedule: Optional[str] = None

@dataclass
class RootNode:
    """Represents the root of the pipeline configuration, containing multiple pipelines."""
    pipelines: List[PipelineNode]
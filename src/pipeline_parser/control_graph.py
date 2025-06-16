import logging
from collections import defaultdict, deque
from typing import Dict, List, Set, Tuple, Any, Optional

from .ast_nodes import RootNode, PipelineNode, StepNode

logger = logging.getLogger(__name__)

class ControlGraph:
    """
    Represents the control flow graph of a pipeline, built from the AST.
    Provides functionality for node/edge representation and cycle detection.
    """

    def __init__(self, ast: RootNode):
        """
        Initializes the ControlGraph from a Pipeline AST.

        Args:
            ast (RootNode): The root node of the pipeline's Abstract Syntax Tree.
        """
        self.ast = ast
        self.graph: Dict[str, List[str]] = defaultdict(list) # Adjacency list: {node_name: [neighbor_names]}
        self.nodes: Dict[str, StepNode] = {} # {node_name: StepNode instance}
        self.errors: List[str] = []
        self._build_graph()

    def _build_graph(self):
        """
        Builds the graph representation from the AST.
        Each step in the pipeline becomes a node. Edges represent sequential flow.
        Handles nested steps within 'iterate' blocks.
        """
        self.graph = defaultdict(list)
        self.nodes = {}
        self.errors = []

        for pipeline_node in self.ast.pipelines:
            # Prefix step names with pipeline name to ensure uniqueness across pipelines
            pipeline_prefix = f"{pipeline_node.name}::"
            
            # Build graph for top-level steps
            self._add_steps_to_graph(pipeline_node.steps, pipeline_prefix)

    def _add_steps_to_graph(self, steps: List[StepNode], prefix: str):
        """
        Recursively adds steps and their nested steps to the graph.
        """
        previous_step_name = None
        for i, step in enumerate(steps):
            current_step_name = f"{prefix}{step.name}"
            if current_step_name in self.nodes:
                self.errors.append(f"Duplicate step name found: '{step.name}' in pipeline '{prefix.strip('::')}'. Step names must be unique within a pipeline.")
                continue
            self.nodes[current_step_name] = step

            if previous_step_name:
                self.graph[previous_step_name].append(current_step_name)
            
            # Handle nested steps for 'iterate' blocks
            if step.iterate and step.steps:
                # The iteration block itself is a node, and its internal steps follow
                # conceptually, the last internal step flows back to the iteration block
                # or to the next step after the iteration block.
                # For simplicity now, we'll just connect the outer step to its first inner step
                # and the last inner step to the next outer step (if any).
                # A more complex model would represent the loop explicitly.
                if step.steps:
                    first_nested_step_name = f"{current_step_name}::{step.steps[0].name}"
                    self.graph[current_step_name].append(first_nested_step_name)
                    self._add_steps_to_graph(step.steps, f"{current_step_name}::")
                    # The last step of the inner loop should conceptually connect to the step after the outer loop
                    # This is complex without explicit 'next' pointers or a more sophisticated AST.
                    # For now, we assume linear flow after iteration completes.
            
            previous_step_name = current_step_name

    def detect_cycles(self) -> List[List[str]]:
        """
        Detects cycles in the control flow graph using DFS.

        Returns:
            List[List[str]]: A list of detected cycles, where each cycle is a list of node names.
        """
        visited: Set[str] = set()
        recursion_stack: Set[str] = set()
        cycles: List[List[str]] = []
        path: List[str] = []

        def dfs(node: str):
            visited.add(node)
            recursion_stack.add(node)
            path.append(node)

            for neighbor in self.graph.get(node, []):
                if neighbor not in visited:
                    dfs(neighbor)
                elif neighbor in recursion_stack:
                    # Cycle detected
                    cycle_start_index = path.index(neighbor)
                    cycles.append(path[cycle_start_index:])
            
            path.pop()
            recursion_stack.remove(node)

        for node_name in self.nodes.keys():
            if node_name not in visited:
                dfs(node_name)
        
        return cycles

    def get_graph_representation(self) -> Dict[str, List[str]]:
        """
        Returns the adjacency list representation of the graph.

        Returns:
            Dict[str, List[str]]: The graph as an adjacency list.
        """
        return self.graph

    def get_node(self, name: str) -> Optional[StepNode]:
        """
        Returns a StepNode instance by its full name (including pipeline prefix).

        Args:
            name (str): The full name of the step node.

        Returns:
            Optional[StepNode]: The StepNode instance, or None if not found.
        """
        return self.nodes.get(name)

    def get_errors(self) -> List[str]:
        """
        Returns a list of errors encountered during graph building.

        Returns:
            List[str]: A list of error messages.
        """
        return self.errors

    def visualize(self) -> str:
        """
        Generates a DOT language string for graph visualization.
        This can be used with tools like Graphviz.

        Returns:
            str: A DOT language string representing the graph.
        """
        dot_string = "digraph PipelineControlFlow {\n"
        dot_string += "  rankdir=LR;\n" # Left to Right
        dot_string += "  node [shape=box];\n"

        # Add nodes
        for node_name in self.nodes.keys():
            dot_string += f'  "{node_name}" [label="{node_name.split("::")[-1]}"];\n'

        # Add edges
        for node, neighbors in self.graph.items():
            for neighbor in neighbors:
                dot_string += f'  "{node}" -> "{neighbor}";\n'
        
        dot_string += "}\n"
        return dot_string
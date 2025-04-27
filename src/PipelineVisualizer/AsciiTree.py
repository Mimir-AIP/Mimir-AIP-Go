"""
ASCII Tree Visualizer for Mimir-AIP Pipeline Runner

- Renders the pipeline as a tree with status icons for each step/iteration.
- Designed for CLI integration and extensibility (supports iteration, sub-steps, future branching).
- Usage: import and call PipelineAsciiTreeVisualizer.render(pipeline_tree) at desired points in the runner.
"""
from typing import List, Dict, Optional
import sys

# Status icons (customizable)
ICONS = {
    'completed': '[✓]',
    'running':   '[→]',
    'pending':   '[ ]',
    'failed':    '[✗]'
}

class PipelineAsciiTreeVisualizer:
    """Provides methods to build and render ASCII tree visualizations of pipeline steps."""
    @staticmethod
    def render(tree: Dict, highlight_path: Optional[List[int]] = None, file=sys.stdout):
        """
        Print the pipeline tree with ASCII branches, updating statuses.
        Args:
            tree (dict): Root node of the pipeline (with children/substeps/iterations).
            highlight_path (list of int): Path to the currently running node (for highlighting).
            file: Output stream (default: sys.stdout)
        """
        # Internal recursive rendering helper
        def _render_node(node, prefix='', is_last=True, path=None):
            path = path or []
            # Status icon
            status = node.get('status', 'pending')
            icon = ICONS.get(status, '[?]')
            # Highlight running node
            highlight = (highlight_path == path) if highlight_path else False
            name = node.get('name', 'unnamed')
            line = f"{prefix}{icon} {name}"
            if highlight:
                line += "  <== running"
            print(line, file=file)
            # Children (substeps or iterations)
            children = node.get('children', [])
            if children:
                for i, child in enumerate(children):
                    is_child_last = (i == len(children) - 1)
                    branch = '    ' if is_last else '|   '
                    connector = '└-- ' if is_child_last else '|-- '
                    _render_node(child, prefix + connector, is_child_last, path + [i])
        _render_node(tree)

    @staticmethod
    def build_tree_from_pipeline(pipeline: Dict, statuses: Optional[Dict[str, str]] = None, runtime_info: Optional[Dict] = None) -> Dict:
        """
        Build a tree structure from pipeline definition and step statuses.
        Args:
            pipeline (dict): Pipeline definition (from YAML).
            statuses (dict): Mapping of step names to status (optional).
            runtime_info (dict): Optional runtime info for dynamic iteration steps (e.g. iteration counts, statuses, labels).
        Returns:
            dict: Tree structure for rendering.
        """
        # Internal recursive node construction helper
        def _build_node(step):
            name = step.get('name', 'unnamed')
            status = (statuses or {}).get(name, 'pending')
            node = {'name': name, 'status': status}
            # Iterative steps: children are iterations, else substeps
            if step.get('iterate'):
                iter_key = name
                iter_count = 0
                iter_statuses = []
                iter_labels = []
                if runtime_info and 'iterations' in runtime_info and iter_key in runtime_info['iterations']:
                    iter_count = runtime_info['iterations'][iter_key].get('count', 0)
                    iter_statuses = runtime_info['iterations'][iter_key].get('statuses', [])
                    iter_labels = runtime_info['iterations'][iter_key].get('labels', [])
                if iter_count > 0:
                    node['children'] = [
                        {'name': iter_labels[i] if i < len(iter_labels) and iter_labels[i] else f"Iteration {i+1}",
                         'status': iter_statuses[i] if i < len(iter_statuses) else 'pending'}
                        for i in range(iter_count)
                    ]
                else:
                    # fallback for demo if no runtime_info
                    node['children'] = [
                        {'name': f"Iteration {i+1}", 'status': 'pending'} for i in range(3)
                    ]
            elif 'steps' in step:
                node['children'] = [_build_node(sub) for sub in step['steps']]
            else:
                node['children'] = []
            return node
        return _build_node(pipeline)

# Example usage (for testing):
if __name__ == "__main__":
    # Example pipeline definition
    pipeline = {
        'name': 'Root',
        'steps': [
            {'name': 'Load Data'},
            {'name': 'Preprocess', 'iterate': 'context["batches"]', 'steps': [
                {'name': 'Clean'},
                {'name': 'Validate'}
            ]},
            {'name': 'Run Model'},
            {'name': 'Save Results'}
        ]
    }
    statuses = {'Load Data': 'completed', 'Preprocess': 'running', 'Run Model': 'pending', 'Save Results': 'pending'}
    runtime_info = {
        'iterations': {
            'Preprocess': {
                'count': 5,
                'statuses': ['completed', 'running', 'pending', 'pending', 'pending'],
                'labels': ['Batch 1', 'Batch 2', 'Batch 3', 'Batch 4', 'Batch 5']
            }
        }
    }
    tree = PipelineAsciiTreeVisualizer.build_tree_from_pipeline(pipeline, statuses, runtime_info)
    PipelineAsciiTreeVisualizer.render(tree, highlight_path=[1])
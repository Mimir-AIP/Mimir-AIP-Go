"""
ASCII Tree Visualizer for Mimir-AIP Pipeline Runner

- Renders the pipeline as a tree with status icons for each step/iteration.
- Designed for CLI integration and extensibility (supports iteration, sub-steps, future branching).
- Usage: import and call PipelineAsciiTreeVisualizer.render(pipeline_tree) at desired points in the runner.
"""
from typing import List, Dict, Optional
import sys

# Status icons and visualization elements
ICONS = {
    'completed': '[+]',  # Using brackets for visual clarity
    'success':   '[+]',
    'running':   '[~]',  # Tilde indicates ongoing process
    'pending':   '[ ]',  # Empty brackets for pending
    'failed':    '[X]',  # Capital X for failure
    'warning':   '[!]',  # Exclamation for warnings
    'skipped':   '[-]'   # Dash for skipped
}

TREE_CONNECTORS = {
    'vertical':    '| ',   # Standard pipe for vertical lines
    'branch':      '+- ',  # Plus and dash for branches
    'end_branch':  '\\- ', # Backslash for final branch
    'horizontal':  '--',   # Double dash for horizontal lines
    'space':       '  '    # Space for indentation
}

class PipelineAsciiTreeVisualizer:
    """Provides methods to build and render ASCII tree visualizations of pipeline steps."""
    
    # Configuration defaults
    compact_mode = False
    show_timings = True
    max_error_length = 20
    
    def __init__(self, show_timings=None):
        # Instance-level show_timings overrides class-level default
        self.show_timings = show_timings if show_timings is not None else self.show_timings

    def generate_tree(self, nodes):
        """Generate ASCII tree representation of pipeline nodes"""
        if not nodes:
            raise ValueError("Empty node structure provided")
            
        return '\n'.join(self._build_tree(nodes, 'root'))

    def _build_tree(self, nodes, node_id, prefix='', is_last=True, is_root=True):
        """Recursively build tree lines"""
        node = nodes.get(node_id)
        if not node:
            raise KeyError(f"Missing node: {node_id}")

        lines = []
        current_line = []
        
        # Build connectors
        if not is_root:
            current_line.append(prefix + ('\\- ' if is_last else '+- '))
            
        # Add status icon and name
        status = node.get('status', 'pending')
        # Map success to completed for backwards compatibility
        if status == 'success':
            status = 'completed'
        icon = ICONS.get(status, '?')
        name = node.get('name', 'unnamed')
        current_line.append(f"{icon} {name}")
        
        # Add timing information if enabled and available
        if self.show_timings:
            start_time = node.get('start_time')
            end_time = node.get('end_time')
            
            if start_time is not None and end_time is not None:
                try:
                    duration = end_time - start_time
                    current_line.append(f" ({duration:.1f}s)")
                except (TypeError, ValueError):
                    # Skip timing display if calculation fails
                    pass
            elif start_time is not None:
                current_line.append(" (running)")
                
        lines.append(''.join(current_line))
        
        # Process children if any
        if 'children' in node:
            for i, child_id in enumerate(node['children']):
                if isinstance(child_id, str) and child_id in nodes:
                    new_prefix = prefix
                    if not is_root:
                        new_prefix += '    ' if is_last else '|   '
                    child_lines = self._build_tree(
                        nodes,
                        child_id,
                        new_prefix,
                        i == len(node['children']) - 1,
                        False
                    )
                    lines.extend(child_lines)
                elif isinstance(child_id, dict):
                    # Handle inline child nodes (for backward compatibility)
                    child_node = child_id
                    new_prefix = prefix
                    if not is_root:
                        new_prefix += '    ' if is_last else '│   '
                    child_lines = self._build_tree_from_node(
                        child_node,
                        new_prefix,
                        i == len(node['children']) - 1
                    )
                    lines.extend(child_lines)
        
        return lines

    def _build_tree_from_node(self, node, prefix='', is_last=True):
        """Helper method to build tree lines from a node dictionary"""
        lines = []
        current_line = []
        
        # Build connectors
        current_line.append(prefix + ('\\- ' if is_last else '+- '))
            
        # Add status icon and name
        status = node.get('status', 'pending')
        if status == 'success':
            status = 'completed'
        icon = ICONS.get(status, '?')
        name = node.get('name', 'unnamed')
        current_line.append(f"{icon} {name}")
        
        # Add timing information if enabled and available
        if self.show_timings:
            start_time = node.get('start_time')
            end_time = node.get('end_time')
            
            if start_time is not None and end_time is not None:
                try:
                    duration = end_time - start_time
                    current_line.append(f" ({duration:.1f}s)")
                except (TypeError, ValueError):
                    # Skip timing display if calculation fails
                    pass
            elif start_time is not None:
                current_line.append(" (running)")
                
        lines.append(''.join(current_line))
        
        # Process children if any
        children = node.get('children', [])
        for i, child in enumerate(children):
            new_prefix = prefix + ('    ' if is_last else '|   ')
            child_lines = self._build_tree_from_node(
                child,
                new_prefix,
                i == len(children) - 1
            )
            lines.extend(child_lines)
            
        return lines
    
    @classmethod
    def configure(cls, compact_mode=None, show_timings=None, max_error_length=None):
        """Configure visualization options"""
        if compact_mode is not None:
            cls.compact_mode = compact_mode
        if show_timings is not None:
            cls.show_timings = show_timings
        if max_error_length is not None:
            cls.max_error_length = max_error_length
    @staticmethod
    def render(tree: Dict, highlight_path: Optional[List[int]] = None,
              runtime_info: Optional[Dict] = None, file=sys.stdout):
        """
        Print the pipeline tree with ASCII branches, updating statuses.
        Args:
            tree (dict): Root node of the pipeline (with children/substeps/iterations).
            highlight_path (list of int): Path to the currently running node (for highlighting).
            runtime_info (dict): Additional runtime information including execution mode.
            file: Output stream (default: sys.stdout)
        """
        # Show execution mode if provided
        if runtime_info and runtime_info.get('execution_mode'):
            mode = runtime_info['execution_mode']
            run_count = runtime_info.get('run_count', 0)
            if mode == 'continuous':
                print(f"[Continuous Mode] Run #{run_count}", file=file)
            elif mode == 'scheduled':
                print(f"[Scheduled Mode] Next run: {runtime_info.get('next_run', 'N/A')}", file=file)
            else:
                print(f"[Single Execution Mode]", file=file)

        # Internal recursive rendering helper
        def _render_node(node, prefix='', is_last=True, path=None):
            path = path or []
            status = node.get('status', 'pending')
            icon = ICONS.get(status, '?')
            name = node.get('name', 'unnamed')
            
            # Build main line
            line = f"{prefix}{icon} {name}"
            
            # Add timing info if available
            if PipelineAsciiTreeVisualizer.show_timings:
                start_time = node.get('start_time')
                end_time = node.get('end_time')
                
                if start_time is not None and end_time is not None:
                    try:
                        duration = end_time - start_time
                        line += f" ({duration:.1f}s)"
                    except (TypeError, ValueError):
                        # Skip timing display if calculation fails
                        pass
                elif start_time is not None:
                    line += " (running)"
                
            # Highlight running node
            if highlight_path and highlight_path == path:
                line += " ◀"
            
            # Show iteration progress
            if runtime_info and 'iterations' in runtime_info and name in runtime_info['iterations']:
                iters = runtime_info['iterations'][name]
                completed = sum(1 for s in iters['statuses'] if s == 'completed')
                total = iters['count']
                line += f" [{completed}/{total}]"
                if total > 0:
                    progress = int((completed / total) * 10)
                    line += f" {'■' * progress}{'□' * (10 - progress)}"
            
            # Show error info if available
            if status == 'failed' and 'error' in node and node['error'] is not None:
                error_text = str(node['error'])[:PipelineAsciiTreeVisualizer.max_error_length]
                line += f" ({error_text}...)"
            
            # Force ASCII output with explicit encoding
            try:
                print(line.encode('utf-8').decode('ascii', 'replace'), file=file)
            except (UnicodeEncodeError, TypeError):
                # Final fallback to simple ASCII
                print(line.encode('ascii', 'replace').decode('ascii'), file=file)
            
            # Children rendering with improved connectors
            children = node.get('children', [])
            if children:
                for i, child in enumerate(children):
                    is_child_last = (i == len(children) - 1)
                    branch = '    ' if is_last else "|   "
                    connector = TREE_CONNECTORS['end_branch'] if is_child_last else TREE_CONNECTORS['branch']
                    _render_node(child, prefix + branch + connector, is_child_last, path + [i])
        _render_node(tree)

    @staticmethod
    def build_tree_from_pipeline(pipeline: Dict, statuses: Optional[Dict[str, str]] = None,
                               runtime_info: Optional[Dict] = None, timing_info: Optional[Dict] = None,
                               error_info: Optional[Dict] = None) -> Dict:
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
            node = {
                'name': name,
                'status': status,
                'start_time': (timing_info or {}).get(name, {}).get('start'),
                'end_time': (timing_info or {}).get(name, {}).get('end'),
                'error': (error_info or {}).get(name)
            }
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
import pytest
from src.PipelineVisualizer.AsciiTree import PipelineAsciiTreeVisualizer

def test_ascii_tree_visualization():
    """Test basic ASCII tree structure with timing information"""
    # Sample pipeline data
    nodes = {
        'root': {
            'name': 'Main Pipeline',
            'status': 'success',
            'children': ['process_data'],
            'start_time': 1620000000.0,
            'end_time': 1620000005.5
        },
        'process_data': {
            'name': 'Data Processing',
            'status': 'running',
            'start_time': 1620000001.0
        }
    }
    
    visualizer = PipelineAsciiTreeVisualizer(show_timings=True)
    result = visualizer.generate_tree(nodes)
    
    # Verify basic structure
    assert 'Main Pipeline' in result
    assert 'Data Processing' in result
    
    # Verify timing displays
    assert '(5.5s)' in result  # Completed duration
    assert '(running)' in result  # Active node
    
    # Verify icons
    assert '✔' in result  # Success icon
    assert '⌛' in result  # Running icon

def test_timing_display_conditions():
    """Test timing display toggling and validation"""
    nodes = {'root': {'name': 'Test Node', 'status': 'success'}}
    
    # Test with timings disabled
    visualizer = PipelineAsciiTreeVisualizer(show_timings=False)
    result = visualizer.generate_tree(nodes)
    assert '(' not in result
    
    # Test with incomplete timing data
    visualizer.show_timings = True
    result = visualizer.generate_tree(nodes)
    assert '(' not in result

def test_error_handling():
    """Test error cases and edge conditions"""
    visualizer = PipelineAsciiTreeVisualizer()
    
    # Test empty input
    with pytest.raises(ValueError):
        visualizer.generate_tree({})
        
    # Test invalid node structure
    with pytest.raises(KeyError):
        visualizer.generate_tree({'root': {}})
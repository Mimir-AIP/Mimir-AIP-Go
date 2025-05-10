"""
Unit tests for VisualGraphs output plugin.
"""
import unittest
import os
import base64
import tempfile
import pandas as pd
from src.Plugins.Output.VisualGraphs.VisualGraphs import VisualGraphs

class TestVisualGraphs(unittest.TestCase):
    """Test cases for VisualGraphs plugin"""
    
    def setUp(self):
        self.plugin = VisualGraphs()
        self.test_data = pd.DataFrame({
            'category': ['A', 'B', 'C'],
            'value': [10, 20, 30]
        })
        self.temp_dir = tempfile.mkdtemp()
        
    def test_bar_chart_file_output(self):
        """Test bar chart generation with file output"""
        config = {
            'chart_type': 'bar',
            'data': self.test_data,
            'output': {
                'path': os.path.join(self.temp_dir, 'bar.png')
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertTrue(os.path.exists(result['graph_file']))
        
    def test_line_chart_file_output(self):
        """Test line chart generation with file output"""
        config = {
            'chart_type': 'line',
            'data': self.test_data,
            'output': {
                'path': os.path.join(self.temp_dir, 'line.png')
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertTrue(os.path.exists(result['graph_file']))
        
    def test_pie_chart_file_output(self):
        """Test pie chart generation with file output"""
        config = {
            'chart_type': 'pie',
            'data': self.test_data,
            'output': {
                'path': os.path.join(self.temp_dir, 'pie.png')
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertTrue(os.path.exists(result['graph_file']))
        
    def test_scatter_chart_file_output(self):
        """Test scatter plot generation with file output"""
        config = {
            'chart_type': 'scatter',
            'data': self.test_data,
            'output': {
                'path': os.path.join(self.temp_dir, 'scatter.png')
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertTrue(os.path.exists(result['graph_file']))
        
    def test_base64_output(self):
        """Test Base64 string output"""
        config = {
            'chart_type': 'bar',
            'data': self.test_data,
            'output': {
                'format': 'base64'
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertIn('graph_base64', result)
        # Verify it's valid base64
        base64.b64decode(result['graph_base64'])
        
    def test_styling_options(self):
        """Test styling options are applied"""
        config = {
            'chart_type': 'bar',
            'data': self.test_data,
            'output': {
                'path': os.path.join(self.temp_dir, 'styled.png')
            },
            'style': {
                'title_style': {'fontsize': 14, 'color': 'red'},
                'axis_style': {'fontsize': 12}
            }
        }
        result = self.plugin.execute_pipeline_step(config, {})
        self.assertTrue(os.path.exists(result['graph_file']))
        
    def test_invalid_chart_type(self):
        """Test error handling for invalid chart type"""
        config = {
            'chart_type': 'invalid',
            'data': self.test_data,
            'output': {'path': 'test.png'}
        }
        with self.assertRaises(ValueError):
            self.plugin.execute_pipeline_step(config, {})
            
    def test_missing_data(self):
        """Test error handling for missing data"""
        config = {
            'chart_type': 'bar',
            'output': {'path': 'test.png'}
        }
        with self.assertRaises(ValueError):
            self.plugin.execute_pipeline_step(config, {})

    def tearDown(self):
        # Clean up temp files
        for f in os.listdir(self.temp_dir):
            os.remove(os.path.join(self.temp_dir, f))
        os.rmdir(self.temp_dir)

if __name__ == '__main__':
    unittest.main()
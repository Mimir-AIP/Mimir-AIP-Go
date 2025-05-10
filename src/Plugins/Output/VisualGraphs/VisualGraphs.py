"""
VisualGraphs output plugin.

Generates various chart types (bar, line, pie, scatter) as images or Base64 strings.
"""
import base64
import io
from typing import Dict, Any
import matplotlib.pyplot as plt
import pandas as pd
from Plugins.BasePlugin import BasePlugin

class VisualGraphs(BasePlugin):
    """Plugin for generating visual graphs from data"""
    
    def execute_pipeline_step(self, step_config: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Generate visual graphs based on configuration
        
        Args:
            step_config: Configuration for this step including:
                - chart_type: Type of chart to generate (bar, line, pie, scatter)
                - data: Data to visualize (dict, list or DataFrame)
                - output: Output configuration (file path or 'base64')
                - style: Optional styling options
            context: Pipeline context containing variables
            
        Returns:
            Updated context with any new variables
        """
        try:
            # Get data from context if not directly provided
            data = step_config.get('data') or context.get(step_config['data_key'])
            
            # Validate data
            if not data:
                raise ValueError("No data provided for visualization")
                
            # Convert to DataFrame if needed
            if not isinstance(data, pd.DataFrame):
                data = pd.DataFrame(data)
                
            # Generate chart based on type
            chart_type = step_config.get('chart_type', 'bar')
            
            # Create figure and axis
            fig, ax = plt.subplots(figsize=step_config.get('figsize', (10, 6)))
            
            # Generate different chart types
            if chart_type == 'bar':
                self._generate_bar_chart(ax, data, step_config)
            elif chart_type == 'line':
                self._generate_line_chart(ax, data, step_config)
            elif chart_type == 'pie':
                self._generate_pie_chart(ax, data, step_config)
            elif chart_type == 'scatter':
                self._generate_scatter_chart(ax, data, step_config)
            else:
                raise ValueError(f"Unsupported chart type: {chart_type}")
                
            # Apply styling if provided
            if 'style' in step_config:
                self._apply_styling(ax, step_config['style'])
                
            # Handle output
            output = step_config.get('output', {})
            if output.get('format') == 'base64':
                return self._output_base64(fig, context)
            else:
                return self._output_file(fig, output, context)
                
        except Exception as e:
            self.log_error(f"Error generating graph: {str(e)}")
            raise

    def _generate_bar_chart(self, ax, data, config):
        """Generate bar chart"""
        x = config.get('x') or data.columns[0]
        y = config.get('y') or data.columns[1]
        ax.bar(data[x], data[y], **config.get('bar_kwargs', {}))
        ax.set_title(config.get('title', 'Bar Chart'))
        ax.set_xlabel(config.get('xlabel', x))
        ax.set_ylabel(config.get('ylabel', y))

    def _generate_line_chart(self, ax, data, config):
        """Generate line chart"""
        x = config.get('x') or data.columns[0]
        y = config.get('y') or data.columns[1]
        ax.plot(data[x], data[y], **config.get('line_kwargs', {}))
        ax.set_title(config.get('title', 'Line Chart'))
        ax.set_xlabel(config.get('xlabel', x))
        ax.set_ylabel(config.get('ylabel', y))

    def _generate_pie_chart(self, ax, data, config):
        """Generate pie chart"""
        labels = config.get('labels') or data.columns[0]
        values = config.get('values') or data.columns[1]
        ax.pie(data[values], labels=data[labels], **config.get('pie_kwargs', {}))
        ax.set_title(config.get('title', 'Pie Chart'))

    def _generate_scatter_chart(self, ax, data, config):
        """Generate scatter plot"""
        x = config.get('x') or data.columns[0]
        y = config.get('y') or data.columns[1]
        ax.scatter(data[x], data[y], **config.get('scatter_kwargs', {}))
        ax.set_title(config.get('title', 'Scatter Plot'))
        ax.set_xlabel(config.get('xlabel', x))
        ax.set_ylabel(config.get('ylabel', y))

    def _apply_styling(self, ax, style_config):
        """Apply styling to chart"""
        if 'title_style' in style_config:
            ax.title.set(**style_config['title_style'])
        if 'axis_style' in style_config:
            ax.xaxis.label.set(**style_config['axis_style'])
            ax.yaxis.label.set(**style_config['axis_style'])
        if 'grid' in style_config:
            ax.grid(**style_config['grid'])

    def _output_base64(self, fig, context):
        """Output chart as Base64 string"""
        buf = io.BytesIO()
        fig.savefig(buf, format='png')
        plt.close(fig)
        base64_str = base64.b64encode(buf.getvalue()).decode('utf-8')
        context['graph_base64'] = base64_str
        return context

    def _output_file(self, fig, output_config, context):
        """Output chart to file"""
        file_path = output_config.get('path', 'output.png')
        fig.savefig(file_path, **output_config.get('save_kwargs', {}))
        plt.close(fig)
        context['graph_file'] = file_path
        return context
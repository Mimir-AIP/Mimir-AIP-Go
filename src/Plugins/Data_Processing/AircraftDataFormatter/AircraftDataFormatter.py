"""
Plugin for formatting aircraft data as a human-readable HTML table for reports.
"""
import html
from Plugins.BasePlugin import BasePlugin

class AircraftDataFormatter(BasePlugin):
    """Formats aircraft data (list of dicts) into an HTML table for report embedding."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """
        Args:
            step_config (dict): expects step_config['config']['input_key'] (str, context key for aircraft data)
                                      step_config['config']['output_key'] (str, context key for HTML output)
            context (dict): pipeline context
        Returns:
            dict: {output_key: html_table_str}
        """
        config = step_config["config"]
        input_key = config.get("input_key", "aircraft_data")
        output_key = config.get("output_key", "aircraft_data_html")
        aircraft_data = context.get(input_key, [])
        if not aircraft_data:
            html_table = "<p>No aircraft data available.</p>"
        else:
            # Choose columns to display
            columns = ["flight", "r", "t", "desc", "ownOp", "alt_geom", "gs", "lat", "lon"]
            header = "".join(f"<th>{html.escape(col)}</th>" for col in columns)
            rows = ""
            for entry in aircraft_data:
                row = "".join(f"<td>{html.escape(str(entry.get(col, '')))}</td>" for col in columns)
                rows += f"<tr>{row}</tr>"
            html_table = f"<table><thead><tr>{header}</tr></thead><tbody>{rows}</tbody></table>"
        return {output_key: html_table}

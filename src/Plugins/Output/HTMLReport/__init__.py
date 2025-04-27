"""
Plugin for generating HTML reports

Example usage:
    plugin = HTMLReport()
    result = plugin.execute_pipeline_step({
        "config": {
            "output_file": "report.html"
        },
        "output": "report"
    }, {})
"""
from .HTMLReport import HTMLReport

__all__ = ['HTMLReport']
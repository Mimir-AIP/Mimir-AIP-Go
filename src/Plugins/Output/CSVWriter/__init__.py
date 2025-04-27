"""Plugin for writing pipeline output data to CSV files.

Example usage:
    plugin = CSVWriter()
    result = plugin.execute_pipeline_step({
        "config": {
            "data_key": "records",
            "output_dir": "csv",
            "filename": "output.csv",
            "include_header": True,
            "delimiter": ","
        },
        "output": "csv_path"
    }, context)
"""
from .CSVWriter import CSVWriter

__all__ = ['CSVWriter']

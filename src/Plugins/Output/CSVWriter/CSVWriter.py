"""
Plugin for writing pipeline data to CSV files.

Example usage:
    plugin = CSVWriter()
    result = plugin.execute_pipeline_step({
        "config": {
            "data_key": "records",        # Context key with list of dicts or lists
            "output_dir": "csv",          # Optional: overrides default directory
            "filename": "output.csv",    # Optional: CSV filename
            "include_header": True,        # Optional: include header row for dict data
            "delimiter": ","             # Optional: CSV delimiter
        },
        "output": "csv_path"             # Key to return in result mapping
    }, context)
"""
import os
import csv
import logging
from Plugins.BasePlugin import BasePlugin


class CSVWriter(BasePlugin):
    """
    Plugin for writing pipeline output data to CSV files.
    """
    plugin_type = "Output"

    def __init__(self, output_directory="csv_outputs"):
        """
        Initialize the CSVWriter plugin.

        Args:
            output_directory (str): Default directory to save CSV files.
        """
        self.output_directory = output_directory
        os.makedirs(self.output_directory, exist_ok=True)

    def execute_pipeline_step(self, step_config, context):
        """
        Execute the CSVWriter pipeline step.

        Args:
            step_config (dict): Full step dict including 'config'.
            context (dict): Pipeline context variables.

        Returns:
            dict: Mapping from step_config['output'] to CSV file path.
        """
        config = step_config.get("config", {})
        logger = logging.getLogger(__name__)

        # Extract data: either from context by key or inline in config
        if "data_key" in config:
            data = context.get(config["data_key"])
            if data is None:
                logger.error(f"[CSVWriter] Context key '{config['data_key']}' not found or None.")
                raise KeyError(f"Context key '{config['data_key']}' not found.")
        elif "data" in config:
            data = config["data"]
        else:
            logger.error("[CSVWriter] No 'data_key' or 'data' specified in config.")
            raise ValueError("CSVWriter requires 'data_key' or 'data' in config.")

        # Validate data type
        if not isinstance(data, (list, tuple)):
            logger.error(f"[CSVWriter] Data is not a list or tuple: {type(data)}")
            raise TypeError("CSVWriter data must be a list or tuple of dicts or lists.")

        # Prepare output directory
        output_dir = config.get("output_dir", self.output_directory)
        os.makedirs(output_dir, exist_ok=True)

        filename = config.get("filename", "output.csv")
        file_path = os.path.join(output_dir, filename)

        delimiter = config.get("delimiter", ",")
        include_header = config.get("include_header", True)

        try:
            with open(file_path, mode="w", newline="", encoding="utf-8") as csvfile:
                # Write dict rows with header
                if data and isinstance(data[0], dict):
                    fieldnames = list(data[0].keys())
                    writer = csv.DictWriter(csvfile, fieldnames=fieldnames, delimiter=delimiter)
                    if include_header:
                        writer.writeheader()
                    for row in data:
                        writer.writerow(row)
                else:
                    # Write list/tuple rows
                    writer = csv.writer(csvfile, delimiter=delimiter)
                    for row in data:
                        writer.writerow(row)
            logger.info(f"[CSVWriter] Wrote CSV file at: {file_path}")
            return {step_config["output"]: file_path}
        except Exception as e:
            logger.error(f"[CSVWriter] Error writing CSV file: {e}")
            raise

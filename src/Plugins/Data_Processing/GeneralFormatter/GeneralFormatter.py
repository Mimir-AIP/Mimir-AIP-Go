"""
GeneralFormatter module.

Formats context data (lists of dicts, dicts, or strings) into HTML or JSON for reporting.

Config Options:
    input_key (str): Context key to format.
    output_key (str): Context key to store formatted result.
    format (str): One of 'html_list', 'table', 'json', 'raw'. Defaults to 'html_list'.
    title_key (str, optional): Key for item title/headline.
    link_key (str, optional): Key for item link.
    body_key (str, optional): Key for item body/content.
    max_items (int, optional): Limit number of items.
"""
from Plugins.BasePlugin import BasePlugin
import html
import json

class GeneralFormatter(BasePlugin):
    """Plugin to format context data into HTML or JSON for reporting."""
    plugin_type = "Data_Processing"

    def execute_pipeline_step(self, step_config, context):
        """Format context data into HTML or other formats for reporting.

        Args:
            step_config (dict): Pipeline step configuration.
            context (dict): Pipeline context.
        Returns:
            dict: Updated context with formatted output.
        """
        cfg = step_config.get("config", {})
        input_key = cfg.get("input_key")
        output_key = cfg.get("output_key")
        fmt = cfg.get("format", "html_list")
        title_key = cfg.get("title_key")
        link_key = cfg.get("link_key")
        body_key = cfg.get("body_key")
        max_items = cfg.get("max_items")

        data = context.get(input_key)
        if data is None:
            context[output_key] = "<div style='color:red'>No data to format.</div>"
            return context

        # Format as HTML list
        if fmt == "html_list":
            html_out = self._format_html_list(data, title_key, link_key, body_key, max_items)
        elif fmt == "table":
            html_out = self._format_html_table(data, max_items)
        elif fmt == "json":
            html_out = f"<pre>{html.escape(json.dumps(data, indent=2))}</pre>"
        else:
            html_out = html.escape(str(data))

        context[output_key] = html_out
        return context

    def _format_html_list(self, data, title_key, link_key, body_key, max_items):
        """Format list of items into an HTML <ul> list.

        Args:
            data (list|dict|str): Data to format.
            title_key (str, optional): Key for item title.
            link_key (str, optional): Key for item link.
            body_key (str, optional): Key for item body.
            max_items (int, optional): Maximum number of items.

        Returns:
            str: HTML string of the list.
        """
        if isinstance(data, dict):
            data = [data]
        if not isinstance(data, list):
            return f"<pre>{html.escape(str(data))}</pre>"
        if max_items:
            try:
                data = data[:int(max_items)]
            except Exception:
                pass
        items = []
        for item in data:
            if not isinstance(item, dict):
                items.append(f"<li>{html.escape(str(item))}</li>")
                continue
            title = html.escape(str(item.get(title_key, ''))) if title_key else ''
            link = item.get(link_key) if link_key else None
            body = html.escape(str(item.get(body_key, ''))) if body_key else ''
            if link and title:
                items.append(f"<li><a href='{html.escape(link)}' target='_blank'>{title}</a>{('<br>'+body) if body else ''}</li>")
            elif title:
                items.append(f"<li>{title}{('<br>'+body) if body else ''}</li>")
            elif body:
                items.append(f"<li>{body}</li>")
            else:
                items.append(f"<li>{html.escape(str(item))}</li>")
        return f"<ul>\n{''.join(items)}\n</ul>"

    def _format_html_table(self, data, max_items):
        """Format list of dicts into an HTML <table>.

        Args:
            data (list|dict): Data to format.
            max_items (int, optional): Maximum number of items.

        Returns:
            str: HTML string of the table.
        """
        if isinstance(data, dict):
            data = [data]
        if not isinstance(data, list) or not data:
            return f"<pre>{html.escape(str(data))}</pre>"
        if max_items:
            try:
                data = data[:int(max_items)]
            except Exception:
                pass
        # Use keys from first item
        keys = list(data[0].keys())
        rows = ["<tr>" + "".join(f"<th>{html.escape(str(k))}</th>" for k in keys) + "</tr>"]
        for item in data:
            row = "<tr>" + "".join(f"<td>{html.escape(str(item.get(k,'')))}</td>" for k in keys) + "</tr>"
            rows.append(row)
        return f"<table border='1'>\n{''.join(rows)}\n</table>"
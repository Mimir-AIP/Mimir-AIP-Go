"""
Plugin for generating HTML reports with sections and styling

Example usage:
    plugin = HTMLReport()
    result = plugin.execute_pipeline_step({
        "config": {
            "title": "Report Title",
            "sections": [
                {
                    "heading": "Section 1",
                    "text": "Some content",
                    "javascript": "console.log('Hello');"
                }
            ],
            "output_dir": "reports",
            "filename": "report.html",
            "css": "..."  # Optional, custom CSS string to override default styling
        },
        "output": "report_path"
    }, {})
"""

import os
import logging
import re
from Plugins.BasePlugin import BasePlugin


# Utility: Ensure a base64 string has exactly one 'data:image/jpeg;base64,' prefix
def ensure_single_base64_prefix(b64_string: str) -> str:
    """
    Ensure the input string has exactly one 'data:image/jpeg;base64,' prefix.
    Removes all existing such prefixes, then prepends one.
    Args:
        b64_string (str): The base64 image string, possibly with or without prefix.
    Returns:
        str: String with exactly one prefix.
    """
    prefix = "data:image/jpeg;base64,"
    if not isinstance(b64_string, str):
        return b64_string
    # Remove all leading prefixes
    while b64_string.startswith(prefix):
        b64_string = b64_string[len(prefix):]
    return prefix + b64_string


class HTMLReport(BasePlugin):
    """
    Plugin for generating HTML reports with sections and styling
    """

    plugin_type = "Output"

    def __init__(self, output_directory="reports"):
        self.output_directory = output_directory
        # Create the output directory if it doesn't exist
        os.makedirs(self.output_directory, exist_ok=True)

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "HTMLReport",
            "config": {
                "title": "Report Title",
                "sections": [
                    {
                        "heading": "Section 1",
                        "text": "Some content",
                        "javascript": "console.log('Hello');"
                    }
                ],
                "output_dir": "reports",  # Optional
                "filename": "report.html"  # Optional
                "css": "..."  # Optional, custom CSS string to override default styling
            },
            "output": "report_path"
        }
        If 'css' is omitted, the default theme is used for backward compatibility.
        """
        config = step_config["config"]
        logger = logging.getLogger(__name__)
        
        # Log all context variables ending with 'image_path' and check file existence
        for k, v in context.items():
            if (k.endswith('image_path') or k == 'traffic_image_path') and isinstance(v, str):
                exists = os.path.exists(v)
                logger.info(f"[HTMLReport] Context variable '{k}': {v} (exists: {exists})")
                if not exists:
                    logger.warning(f"[HTMLReport] Image file referenced by '{k}' does not exist: {v}")
        
        # Update output directory if specified
        if "output_dir" in config:
            self.output_directory = config["output_dir"]
            os.makedirs(self.output_directory, exist_ok=True)
        
        # Evaluate sections expression
        try:
            logger.info(f"[HTMLReport] Output directory: {self.output_directory}")
            logger.info(f"[HTMLReport] Step config: {step_config}")
            logger.info(f"[HTMLReport] Context keys: {list(context.keys())}")
            # Diagnostic: log samples of image base64
            boxed_b64 = context.get('boxed_image_base64', None)
            img_b64 = context.get('image_base64', None)
            logger.info(f"[HTMLReport] boxed_image_base64 present: {boxed_b64 is not None}, sample: {boxed_b64[:40] if boxed_b64 else 'None'}")
            logger.info(f"[HTMLReport] image_base64 present: {img_b64 is not None}, sample: {img_b64[:40] if img_b64 else 'None'}")
            logger.info(f"[HTMLReport] Evaluating sections from config: {config.get('sections')}")
            sections = eval(config["sections"], {"context": context, **context}) if isinstance(config["sections"], str) else config["sections"]
            logger.info(f"[HTMLReport] Number of sections to write: {len(sections)}")
            logger.info(f"[HTMLReport] Sample section: {sections[0] if sections else 'None'}")

            # Robust placeholder substitution for each section
            for section in sections:
                # Check all text fields for placeholders
                for key in list(section.keys()):
                    value = section[key]
                    if isinstance(value, str):
                        matches = re.findall(r'\{([a-zA-Z0-9_]+)\}', value)
                        for match in matches:
                            replacement = context.get(match, None)
                            if replacement is None or (isinstance(replacement, (list, dict)) and not replacement):
                                # Hide section if critical data is missing or empty
                                if key == 'text':
                                    section[key] = ''
                                else:
                                    section[key] = section[key].replace(f"{{{match}}}", "")
                                logger.warning(f"[HTMLReport] Placeholder {{{match}}} not found or empty in context; hiding section or substituting blank.")
                            else:
                                section[key] = section[key].replace(f"{{{match}}}", str(replacement))
            # Remove sections with empty 'text' or all empty fields
            sections = [s for s in sections if any(str(v).strip() for k, v in s.items() if k != 'javascript')]
        except Exception as e:
            logger.error(f"Error evaluating sections: {e}")
            raise
        
        # Handle custom CSS (theme)
        css = config.get("css", None)
        
        # Generate report
        try:
            filename = config.get("filename", "report.html")
            logger.info(f"[HTMLReport] Writing report to: {os.path.join(self.output_directory, filename)}")
            report_path = self.generate_report(
                title=config["title"],
                sections=sections,
                filename=filename,
                css=css
            )
            logger.info(f"[HTMLReport] Generated report at: {report_path}")
            return {step_config["output"]: report_path}
        except Exception as e:
            logger.error(f"Error generating report: {e}")
            raise

    def generate_report(self, title, sections, filename="report.html", css=None):
        """
        Generate an HTML report with multiple text and JavaScript sections, supporting custom theming.

        :param title: Title of the HTML document
        :param sections: List of sections, where each section is a dictionary with:
                     - "heading": Heading for the section
                     - "text": Text content for the section (HTML allowed, supports {var} interpolation)
                     - "javascript": JavaScript code for the section
        :param filename: Name of the output HTML file
        :param css: Optional custom CSS string to override the default styling
        :return: Absolute path to the generated HTML file
        """
        import logging
        logger = logging.getLogger(__name__)
        # Generate HTML content for all text and JavaScript sections
        section_html = ""
        # Variable interpolation: use context if available
        import inspect
        # Find the context from the caller's stack if passed
        frame = inspect.currentframe()
        context = {}
        try:
            outer = frame.f_back.f_back
            if 'context' in outer.f_locals:
                context = outer.f_locals['context']
        except Exception:
            pass
        for section in sections:
            # Interpolate variables in text using context
            text = section.get('text', '')
            if context:
                try:
                    # Use image path variables if present for direct linking
                    safe_context = context.copy()
                    for k, v in safe_context.items():
                        # Prefer *_image_path or traffic_image_path for direct linking
                        if (k.endswith('image_path') or k == 'traffic_image_path') and isinstance(v, str):
                            # Make the path relative to the HTML report if needed
                            import os
                            report_dir = os.path.dirname(os.path.abspath(os.path.join(self.output_directory, filename)))
                            rel_path = os.path.relpath(v, report_dir)
                            safe_context[k] = rel_path
                    text = text.format(**safe_context)
                except Exception as e:
                    logger.warning(f"[HTMLReport] Error formatting section text with context: {e}")
                    pass  # fallback to raw text if formatting fails
            # No base64 sanitization needed for direct file linking
            section_html += f"""
            <div class="section">
                <h2>{section.get('heading', '')}</h2>
                <div class="content">
                    {text}
                </div>
                {f'<script>{section["javascript"]}</script>' if section.get('javascript') else ''}
            </div>
            """
        # Default CSS (Mimir-AIP GitHub Pages inspired)
        default_css = '''
html {
  font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Segoe UI Symbol";
  -ms-text-size-adjust: 100%;
  -webkit-text-size-adjust: 100%;
}
body {
  margin: 0;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.5;
  color: #24292e;
  background-color: #fff;
  max-width: 900px;
  margin-left: auto;
  margin-right: auto;
  padding: 32px 16px 32px 16px;
}
h1 {
  font-size: 2em;
  font-weight: 600;
  color: #2c3e50;
  text-align: center;
  margin-top: 0.67em;
  margin-bottom: 0.67em;
  padding-bottom: 10px;
  border-bottom: 2px solid #eee;
}
h2 {
  font-size: 1.5em;
  font-weight: 600;
  color: #34495e;
  margin-top: 30px;
  margin-bottom: 0.5em;
}
p {
  margin-top: 0;
  margin-bottom: 10px;
}
a {
  color: #0366d6;
  text-decoration: none;
}
a:hover {
  text-decoration: underline;
}
.section {
  background: #fff;
  padding: 24px 20px;
  margin: 24px 0;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  box-shadow: 0 1px 5px rgba(27,31,35,0.07);
}
.content {
  margin: 15px 0;
  padding: 10px;
  background: #f6f8fa;
  border-radius: 5px;
  overflow-x: auto;
}
footer {
  text-align: center;
  margin-top: 32px;
  font-size: 0.95em;
  color: #888;
}
code, pre {
  font-family: "SFMono-Regular",Consolas,"Liberation Mono",Menlo,Courier,monospace;
  font-size: 13px;
  background: #f6f8fa;
  border-radius: 4px;
  padding: 2px 4px;
}
table {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
}
th, td {
  border: 1px solid #e1e4e8;
  padding: 8px 12px;
  text-align: left;
}
th {
  background: #f6f8fa;
}
blockquote {
  margin: 0 0 16px 0;
  padding-left: 1em;
  color: #6a737d;
  border-left: 4px solid #dfe2e5;
  background: #f6f8fa;
}
hr {
  border: 0;
  border-bottom: 1px solid #e1e4e8;
  margin: 24px 0;
}
'''
        html_template = f"""
<!DOCTYPE html>
<html>
<head>
    <title>{title}</title>
    <style>
    {css or default_css}
    </style>
</head>
<body>
    <h1>{title}</h1>
    {section_html}
    <footer>
        Generated by HTMLReport
    </footer>
</body>
</html>
"""
        # Write to file
        report_path = os.path.join(self.output_directory, filename)
        try:
            with open(report_path, "w", encoding="utf-8") as file:
                file.write(html_template)
            logger.info(f"[HTMLReport] Successfully wrote HTML to {report_path}")
        except Exception as e:
            logger.error(f"[HTMLReport] ERROR writing HTML to {report_path}: {e}")
            raise
        # Check existence immediately after writing
        if os.path.exists(report_path):
            logger.info(f"[HTMLReport] File confirmed present after write: {report_path}")
        else:
            logger.error(f"[HTMLReport] File missing after write: {report_path}")
        print(f"Report generated: {report_path}")
        return report_path


if __name__ == "__main__":
    # Test the plugin
    plugin = HTMLReport()
    
    # Test configuration
    test_config = {
        "plugin": "HTMLReport",
        "config": {
            "title": "Test Report",
            "sections": [
                {
                    "heading": "Section 1",
                    "text": "<p>This is a test section with some <b>formatted</b> content.</p>",
                    "javascript": "console.log('Section 1 loaded');"
                },
                {
                    "heading": "Section 2",
                    "text": "<p>Another section with a list:</p><ul><li>Item 1</li><li>Item 2</li></ul>"
                }
            ],
            "output_dir": "test_reports",
            "filename": "test_report.html"
        },
        "output": "report_path"
    }
    
    # Generate test report
    result = plugin.execute_pipeline_step(test_config, {})
    print(f"Test report generated at: {result['report_path']}")
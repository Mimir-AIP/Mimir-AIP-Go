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
from Plugins.BasePlugin import BasePlugin


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
        
        # Update output directory if specified
        if "output_dir" in config:
            self.output_directory = config["output_dir"]
            os.makedirs(self.output_directory, exist_ok=True)
        
        # Evaluate sections expression
        try:
            logger.info(f"[HTMLReport] Output directory: {self.output_directory}")
            logger.info(f"[HTMLReport] Step config: {step_config}")
            logger.info(f"[HTMLReport] Context keys: {list(context.keys())}")
            logger.info(f"[HTMLReport] Evaluating sections from config: {config.get('sections')}")
            sections = eval(config["sections"], {"context": context, **context}) if isinstance(config["sections"], str) else config["sections"]
            logger.info(f"[HTMLReport] Number of sections to write: {len(sections)}")
            logger.info(f"[HTMLReport] Sample section: {sections[0] if sections else 'None'}")

            # Robust placeholder substitution for each section
            import re
            placeholder_pattern = re.compile(r'\{([a-zA-Z0-9_]+)\}')
            for section in sections:
                # Check all text fields for placeholders
                for key in list(section.keys()):
                    value = section[key]
                    if isinstance(value, str):
                        matches = placeholder_pattern.findall(value)
                        for match in matches:
                            replacement = context.get(match, "No data available")
                            if replacement == "No data available":
                                logger.warning(f"[HTMLReport] Placeholder {{{match}}} not found in context; substituting default.")
                            # Replace all occurrences of the placeholder
                            section[key] = section[key].replace(f"{{{match}}}", str(replacement))
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
        import os
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
                    text = text.format(**context)
                except Exception:
                    pass  # fallback to raw text if formatting fails
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
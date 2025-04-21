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
            "filename": "report.html"
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
            },
            "output": "report_path"
        }
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
        except Exception as e:
            logger.error(f"Error evaluating sections: {e}")
            raise
        
        # Generate report
        try:
            filename = config.get("filename", "report.html")
            logger.info(f"[HTMLReport] Writing report to: {os.path.join(self.output_directory, filename)}")
            report_path = self.generate_report(
                title=config["title"],
                sections=sections,
                filename=filename
            )
            logger.info(f"[HTMLReport] Generated report at: {report_path}")
            return {step_config["output"]: report_path}
        except Exception as e:
            logger.error(f"Error generating report: {e}")
            raise

    def generate_report(self, title, sections, filename="report.html"):
        """
        Generate an HTML report with multiple text and JavaScript sections.

        :param title: Title of the HTML document
        :param sections: List of sections, where each section is a dictionary with:
                     - "heading": Heading for the section
                     - "text": Text content for the section (HTML allowed)
                     - "javascript": JavaScript code for the section
        :param filename: Name of the output HTML file
        """
        import os
        import logging
        logger = logging.getLogger(__name__)
        # Generate HTML content for all text and JavaScript sections
        section_html = ""
        for section in sections:
            section_html += f"""
            <div class="section">
                <h2>{section.get('heading', '')}</h2>
                <div class="content">
                    {section.get('text', '')}
                </div>
                {f'<script>{section["javascript"]}</script>' if section.get('javascript') else ''}
            </div>
            """

        # HTML template with styling
        html_template = f"""
<!DOCTYPE html>
<html>
<head>
    <title>{title}</title>
    <style>
        body {{
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }}
        h1 {{
            color: #2c3e50;
            text-align: center;
            padding-bottom: 10px;
            border-bottom: 2px solid #eee;
        }}
        h2 {{
            color: #34495e;
            margin-top: 30px;
        }}
        .section {{
            background: #fff;
            padding: 20px;
            margin: 20px 0;
            border: 1px solid #ddd;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .content {{
            margin: 15px 0;
            padding: 10px;
            background: #f9f9f9;
            border-radius: 5px;
            overflow-x: auto;
        }}
        footer {{
            text-align: center;
            margin-top: 20px;
            font-size: 0.9em;
            color: #888;
        }}
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
        # Full path for the output file
        report_path = os.path.abspath(os.path.join(self.output_directory, filename))
        logger.info(f"[HTMLReport] Absolute report path: {report_path}")

        # Write the HTML content to the file
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
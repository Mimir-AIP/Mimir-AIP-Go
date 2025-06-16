"""
Test reporting module for comprehensive test results analysis and visualization.

This module provides functionality to:
- Parse pytest and Jest test results
- Collect coverage data
- Track resource usage
- Generate performance metrics
- Create detailed HTML reports with visualizations

Classes:
    TestReporter: Main class for managing test reporting
    CoverageCollector: Handles coverage data collection and analysis
    MetricsAggregator: Aggregates and processes performance metrics
    ReportGenerator: Generates HTML reports with visualizations
"""

import os
import json
import logging
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Any
import psutil
import pandas as pd
import matplotlib.pyplot as plt
from jinja2 import Template

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class TestResult:
    """Data class for storing individual test results."""
    name: str
    result: str  # success, failure, error, skipped
    duration: float
    message: Optional[str] = None
    traceback: Optional[str] = None

@dataclass
class CoverageData:
    """Data class for storing coverage information."""
    statements: int
    covered: int
    percentage: float
    missing_lines: List[int]

class CoverageCollector:
    """Collects and processes coverage data from test runs."""
    
    def __init__(self, coverage_dir: str):
        """Initialize the coverage collector.
        
        Args:
            coverage_dir: Directory containing coverage data files
        """
        self.coverage_dir = Path(coverage_dir)
        self.python_coverage: Dict[str, CoverageData] = {}
        self.js_coverage: Dict[str, CoverageData] = {}

    def collect_python_coverage(self) -> None:
        """Parse Python coverage data from .coverage files."""
        try:
            import coverage
            cov = coverage.Coverage()
            cov.load()
            
            for filename in cov.data.measured_files():
                analysis = cov.analysis(filename)
                self.python_coverage[filename] = CoverageData(
                    statements=len(analysis[1]),
                    covered=len(analysis[1]) - len(analysis[2]),
                    percentage=cov.report(filename, show_missing=False),
                    missing_lines=analysis[2]
                )
        except Exception as e:
            logger.error(f"Error collecting Python coverage: {e}")

    def collect_js_coverage(self) -> None:
        """Parse JavaScript coverage data from lcov files."""
        try:
            lcov_file = self.coverage_dir / "js" / "lcov.info"
            if lcov_file.exists():
                current_file = None
                statements = 0
                covered = 0
                missing_lines = []

                with open(lcov_file) as f:
                    for line in f:
                        if line.startswith("SF:"):
                            current_file = line[3:].strip()
                            statements = covered = 0
                            missing_lines = []
                        elif line.startswith("DA:"):
                            _, execution = line[3:].strip().split(",")
                            statements += 1
                            if execution == "0":
                                missing_lines.append(int(line[3:].split(",")[0]))
                            else:
                                covered += 1
                        elif line.startswith("end_of_record"):
                            if current_file:
                                self.js_coverage[current_file] = CoverageData(
                                    statements=statements,
                                    covered=covered,
                                    percentage=(covered / statements * 100) if statements > 0 else 0,
                                    missing_lines=missing_lines
                                )
        except Exception as e:
            logger.error(f"Error collecting JavaScript coverage: {e}")

class MetricsAggregator:
    """Aggregates and processes test performance metrics."""

    def __init__(self):
        """Initialize the metrics aggregator."""
        self.start_time = datetime.now()
        self.metrics: Dict[str, Any] = {
            'timing': {},
            'memory': [],
            'cpu': []
        }
        self.process = psutil.Process()

    def start_monitoring(self) -> None:
        """Start monitoring system resources."""
        self.start_time = datetime.now()
        self._record_metrics()

    def _record_metrics(self) -> None:
        """Record current system metrics."""
        try:
            self.metrics['memory'].append(self.process.memory_info().rss / 1024 / 1024)  # MB
            self.metrics['cpu'].append(self.process.cpu_percent())
        except Exception as e:
            logger.error(f"Error recording metrics: {e}")

    def add_timing(self, name: str, duration: float) -> None:
        """Add timing data for a specific test or operation.
        
        Args:
            name: Name of the test or operation
            duration: Duration in seconds
        """
        self.metrics['timing'][name] = duration

    def get_summary(self) -> Dict[str, Any]:
        """Generate summary of collected metrics.
        
        Returns:
            Dict containing summarized metrics
        """
        return {
            'total_duration': (datetime.now() - self.start_time).total_seconds(),
            'max_memory': max(self.metrics['memory']) if self.metrics['memory'] else 0,
            'avg_memory': sum(self.metrics['memory']) / len(self.metrics['memory']) if self.metrics['memory'] else 0,
            'max_cpu': max(self.metrics['cpu']) if self.metrics['cpu'] else 0,
            'avg_cpu': sum(self.metrics['cpu']) / len(self.metrics['cpu']) if self.metrics['cpu'] else 0,
            'timing_data': self.metrics['timing']
        }

class ReportGenerator:
    """Generates HTML test reports with visualizations."""

    def __init__(self, output_dir: str):
        """Initialize the report generator.
        
        Args:
            output_dir: Directory for output files
        """
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)

    def create_performance_graphs(self, metrics: Dict[str, Any]) -> Dict[str, str]:
        """Create performance visualization graphs.
        
        Args:
            metrics: Dictionary of collected metrics
            
        Returns:
            Dict mapping graph types to their filenames
        """
        graphs = {}
        
        try:
            # Timing graph
            plt.figure(figsize=(10, 6))
            pd.Series(metrics['timing_data']).plot(kind='bar')
            plt.title('Test Execution Times')
            plt.xlabel('Test Name')
            plt.ylabel('Duration (s)')
            plt.xticks(rotation=45)
            timing_graph = self.output_dir / 'timing.png'
            plt.savefig(timing_graph, bbox_inches='tight')
            plt.close()
            graphs['timing'] = str(timing_graph)

            # Resource usage graph
            fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(10, 8))
            
            times = range(len(metrics['memory']))
            ax1.plot(times, metrics['memory'])
            ax1.set_title('Memory Usage')
            ax1.set_ylabel('Memory (MB)')
            
            ax2.plot(times, metrics['cpu'])
            ax2.set_title('CPU Usage')
            ax2.set_ylabel('CPU %')
            
            resource_graph = self.output_dir / 'resources.png'
            plt.savefig(resource_graph, bbox_inches='tight')
            plt.close()
            graphs['resources'] = str(resource_graph)

        except Exception as e:
            logger.error(f"Error generating performance graphs: {e}")

        return graphs

    def generate_html_report(self, 
                           test_results: Dict[str, List[TestResult]],
                           coverage_data: Dict[str, Dict[str, CoverageData]],
                           metrics: Dict[str, Any],
                           graphs: Dict[str, str]) -> str:
        """Generate HTML report with all test results and metrics.
        
        Args:
            test_results: Dictionary of test results by type
            coverage_data: Dictionary of coverage data by language
            metrics: Dictionary of performance metrics
            graphs: Dictionary of generated graph filenames
            
        Returns:
            Path to generated HTML report
        """
        template = Template('''
<!DOCTYPE html>
<html>
<head>
    <title>Test Results Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 2em; }
        .card { border: 1px solid #ddd; padding: 1em; margin: 1em 0; border-radius: 4px; }
        .success { color: green; }
        .failure { color: red; }
        .warning { color: orange; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f5f5f5; }
        .graph { max-width: 100%; height: auto; margin: 1em 0; }
    </style>
</head>
<body>
    <h1>Test Results Report</h1>
    <p>Generated on: {{ timestamp }}</p>

    <div class="card">
        <h2>Summary</h2>
        <table>
            <tr>
                <th>Metric</th>
                <th>Value</th>
            </tr>
            <tr>
                <td>Total Duration</td>
                <td>{{ "%.2f"|format(metrics.total_duration) }}s</td>
            </tr>
            <tr>
                <td>Max Memory Usage</td>
                <td>{{ "%.2f"|format(metrics.max_memory) }} MB</td>
            </tr>
            <tr>
                <td>Average CPU Usage</td>
                <td>{{ "%.2f"|format(metrics.avg_cpu) }}%</td>
            </tr>
        </table>
    </div>

    <div class="card">
        <h2>Test Results</h2>
        {% for test_type, results in test_results.items() %}
        <h3>{{ test_type }}</h3>
        <table>
            <tr>
                <th>Test</th>
                <th>Result</th>
                <th>Duration</th>
                <th>Message</th>
            </tr>
            {% for result in results %}
            <tr>
                <td>{{ result.name }}</td>
                <td class="{{ result.result }}">{{ result.result }}</td>
                <td>{{ "%.3f"|format(result.duration) }}s</td>
                <td>{{ result.message if result.message else "" }}</td>
            </tr>
            {% endfor %}
        </table>
        {% endfor %}
    </div>

    <div class="card">
        <h2>Coverage</h2>
        {% for lang, files in coverage_data.items() %}
        <h3>{{ lang }} Coverage</h3>
        <table>
            <tr>
                <th>File</th>
                <th>Statements</th>
                <th>Covered</th>
                <th>Coverage %</th>
            </tr>
            {% for file, data in files.items() %}
            <tr>
                <td>{{ file }}</td>
                <td>{{ data.statements }}</td>
                <td>{{ data.covered }}</td>
                <td>{{ "%.2f"|format(data.percentage) }}%</td>
            </tr>
            {% endfor %}
        </table>
        {% endfor %}
    </div>

    <div class="card">
        <h2>Performance Metrics</h2>
        <div>
            <h3>Execution Times</h3>
            <img src="{{ graphs.timing }}" alt="Test Execution Times" class="graph">
        </div>
        <div>
            <h3>Resource Usage</h3>
            <img src="{{ graphs.resources }}" alt="Resource Usage" class="graph">
        </div>
    </div>
</body>
</html>
''')

        report_path = self.output_dir / 'test_report.html'
        try:
            with open(report_path, 'w') as f:
                f.write(template.render(
                    timestamp=datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
                    test_results=test_results,
                    coverage_data=coverage_data,
                    metrics=metrics,
                    graphs=graphs
                ))
        except Exception as e:
            logger.error(f"Error generating HTML report: {e}")
            raise

        return str(report_path)

class TestReporter:
    """Main class for managing test reporting and analysis."""

    def __init__(self, report_dir: str = "./test_reports"):
        """Initialize the test reporter.
        
        Args:
            report_dir: Base directory for reports and artifacts
        """
        self.report_dir = Path(report_dir)
        self.report_dir.mkdir(parents=True, exist_ok=True)
        
        self.coverage_collector = CoverageCollector(self.report_dir / "coverage")
        self.metrics_aggregator = MetricsAggregator()
        self.report_generator = ReportGenerator(self.report_dir)
        
        self.test_results: Dict[str, List[TestResult]] = {
            'unit': [],
            'integration': [],
            'performance': []
        }

    def start_test_run(self) -> None:
        """Start a new test run, initializing metrics collection."""
        self.metrics_aggregator.start_monitoring()

    def parse_pytest_results(self, xml_path: str) -> None:
        """Parse pytest XML results file.
        
        Args:
            xml_path: Path to pytest XML results file
        """
        try:
            tree = ET.parse(xml_path)
            root = tree.getroot()
            
            for testcase in root.findall(".//testcase"):
                result = "success"
                message = None
                traceback = None
                
                failure = testcase.find("failure")
                error = testcase.find("error")
                skipped = testcase.find("skipped")
                
                if failure is not None:
                    result = "failure"
                    message = failure.get("message")
                    traceback = failure.text
                elif error is not None:
                    result = "error"
                    message = error.get("message")
                    traceback = error.text
                elif skipped is not None:
                    result = "skipped"
                    message = skipped.get("message")
                
                test_type = "unit" if "unit" in xml_path else "integration" if "integration" in xml_path else "performance"
                
                self.test_results[test_type].append(TestResult(
                    name=f"{testcase.get('classname')}.{testcase.get('name')}",
                    result=result,
                    duration=float(testcase.get('time', 0)),
                    message=message,
                    traceback=traceback
                ))
                
                if result != "skipped":
                    self.metrics_aggregator.add_timing(
                        f"{testcase.get('classname')}.{testcase.get('name')}",
                        float(testcase.get('time', 0))
                    )
        except Exception as e:
            logger.error(f"Error parsing pytest results: {e}")
            raise

    def parse_jest_results(self, xml_path: str) -> None:
        """Parse Jest XML results file.
        
        Args:
            xml_path: Path to Jest XML results file
        """
        try:
            tree = ET.parse(xml_path)
            root = tree.getroot()
            
            for testcase in root.findall(".//testcase"):
                result = "success"
                message = None
                traceback = None
                
                failure = testcase.find("failure")
                error = testcase.find("error")
                skipped = testcase.find("skipped")
                
                if failure is not None:
                    result = "failure"
                    message = failure.get("message")
                    traceback = failure.text
                elif error is not None:
                    result = "error"
                    message = error.get("message")
                    traceback = error.text
                elif skipped is not None:
                    result = "skipped"
                    message = skipped.get("message")
                
                test_type = "unit" if "unit" in xml_path else "integration" if "integration" in xml_path else "performance"
                
                self.test_results[test_type].append(TestResult(
                    name=f"{testcase.get('classname')}.{testcase.get('name')}",
                    result=result,
                    duration=float(testcase.get('time', 0)),
                    message=message,
                    traceback=traceback
                ))
                
                if result != "skipped":
                    self.metrics_aggregator.add_timing(
                        f"{testcase.get('classname')}.{testcase.get('name')}",
                        float(testcase.get('time', 0))
                    )
        except Exception as e:
            logger.error(f"Error parsing Jest results: {e}")
            raise

    def collect_coverage(self) -> None:
        """Collect coverage data from both Python and JavaScript tests."""
        self.coverage_collector.collect_python_coverage()
        self.coverage_collector.collect_js_coverage()

    def generate_report(self) -> str:
        """Generate final HTML report with all test results and metrics.
        
        Returns:
            Path to generated HTML report
        """
        try:
            # Create performance graphs
            graphs = self.report_generator.create_performance_graphs(
                self.metrics_aggregator.metrics
            )
            
            # Generate HTML report
            report_path = self.report_generator.generate_html_report(
                test_results=self.test_results,
                coverage_data={
                    'Python': self.coverage_collector.python_coverage,
                    'JavaScript': self.coverage_collector.js_coverage
                },
                metrics=self.metrics_aggregator.get_summary(),
                graphs=graphs
            )
            
            # Generate JSON data for CI/CD
            json_data = {
                'test_results': {
                    test_type: [vars(result) for result in results]
                    for test_type, results in self.test_results.items()
                },
                'coverage': {
                    'python': {
                        file: vars(data)
                        for file, data in self.coverage_collector.python_coverage.items()
                    },
                    'javascript': {
                        file: vars(data)
                        for file, data in self.coverage_collector.js_coverage.items()
                    }
                },
                'metrics': self.metrics_aggregator.get_summary()
            }
            
            json_path = self.report_dir / 'test_results.json'
            with open(json_path, 'w') as f:
                json.dump(json_data, f, indent=2)
            
            return report_path
            
        except Exception as e:
            logger.error(f"Error generating report: {e}")
            raise
"""Test Metrics Generation Script

This script collects and analyzes test metrics including:
- Test execution times
- Coverage statistics
- Test counts by category
- Pass/fail rates
- Resource usage
"""

import json
import os
import re
import time
import logging
from pathlib import Path
from typing import Dict, List, Optional
from dataclasses import dataclass
import xml.etree.ElementTree as ET
import psutil

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class TestSuiteMetrics:
    """Metrics for a test suite."""
    name: str
    test_count: int
    passed: int
    failed: int
    skipped: int
    execution_time: float
    coverage: float
    memory_usage: float
    cpu_usage: float

@dataclass
class BenchmarkResult:
    """Results from performance benchmarks."""
    name: str
    mean_time: float
    min_time: float
    max_time: float
    std_dev: float
    iterations: int

class TestMetricsCollector:
    """Collects and analyzes test metrics."""

    def __init__(self, project_root: Path):
        """Initialize metrics collector.
        
        Args:
            project_root: Root directory of the project
        """
        self.project_root = project_root
        self.reports_dir = project_root / "reports"
        self.reports_dir.mkdir(exist_ok=True)
        self.test_suites: Dict[str, TestSuiteMetrics] = {}
        self.benchmarks: List[BenchmarkResult] = []

    def collect_junit_metrics(self) -> None:
        """Collect metrics from JUnit XML reports."""
        junit_path = self.reports_dir / "junit" / "test-results.xml"
        if not junit_path.exists():
            logger.warning("No JUnit report found")
            return

        tree = ET.parse(junit_path)
        root = tree.getroot()
        
        for suite in root.findall(".//testsuite"):
            name = suite.get("name", "unknown")
            metrics = TestSuiteMetrics(
                name=name,
                test_count=int(suite.get("tests", 0)),
                passed=int(suite.get("tests", 0)) - int(suite.get("failures", 0)) - int(suite.get("errors", 0)),
                failed=int(suite.get("failures", 0)) + int(suite.get("errors", 0)),
                skipped=int(suite.get("skipped", 0)),
                execution_time=float(suite.get("time", 0)),
                coverage=0.0,  # Will be updated from coverage data
                memory_usage=0.0,  # Will be updated from resource metrics
                cpu_usage=0.0  # Will be updated from resource metrics
            )
            self.test_suites[name] = metrics

    def collect_coverage_metrics(self) -> None:
        """Collect metrics from coverage reports."""
        coverage_path = self.project_root / "coverage.json"
        if not coverage_path.exists():
            logger.warning("No coverage report found")
            return
            
        with open(coverage_path) as f:
            coverage_data = json.load(f)
            
        # Update coverage in test suites
        for suite_name, suite in self.test_suites.items():
            if suite_name in coverage_data:
                suite.coverage = coverage_data[suite_name].get("line_rate", 0) * 100

    def collect_benchmark_results(self) -> None:
        """Collect performance benchmark results."""
        benchmark_path = self.reports_dir / "benchmarks.json"
        if not benchmark_path.exists():
            logger.warning("No benchmark results found")
            return
            
        with open(benchmark_path) as f:
            benchmark_data = json.load(f)
            
        for bench in benchmark_data["benchmarks"]:
            result = BenchmarkResult(
                name=bench["name"],
                mean_time=bench["stats"]["mean"],
                min_time=bench["stats"]["min"],
                max_time=bench["stats"]["max"],
                std_dev=bench["stats"]["stddev"],
                iterations=bench["stats"]["iterations"]
            )
            self.benchmarks.append(result)

    def collect_resource_metrics(self) -> None:
        """Collect system resource usage metrics."""
        process = psutil.Process()
        
        # Memory usage
        memory_info = process.memory_info()
        memory_usage = memory_info.rss / 1024 / 1024  # Convert to MB
        
        # CPU usage
        cpu_usage = process.cpu_percent()
        
        # Update resource metrics in test suites
        for suite in self.test_suites.values():
            suite.memory_usage = memory_usage
            suite.cpu_usage = cpu_usage

    def generate_metrics_report(self) -> str:
        """Generate comprehensive metrics report.
        
        Returns:
            Formatted report string
        """
        report = ["Test Metrics Report", "=" * 20, ""]
        
        # Test Suite Summary
        report.append("Test Suite Summary:")
        total_tests = sum(s.test_count for s in self.test_suites.values())
        total_passed = sum(s.passed for s in self.test_suites.values())
        total_failed = sum(s.failed for s in self.test_suites.values())
        total_skipped = sum(s.skipped for s in self.test_suites.values())
        
        report.append(f"Total Tests: {total_tests}")
        report.append(f"Passed: {total_passed}")
        report.append(f"Failed: {total_failed}")
        report.append(f"Skipped: {total_skipped}")
        report.append("")
        
        # Coverage Summary
        report.append("Coverage Summary:")
        for suite_name, suite in self.test_suites.items():
            report.append(f"{suite_name}: {suite.coverage:.1f}%")
        report.append("")
        
        # Performance Summary
        report.append("Performance Summary:")
        for bench in self.benchmarks:
            report.append(f"\n{bench.name}:")
            report.append(f"  Mean Time: {bench.mean_time:.3f}s")
            report.append(f"  Min/Max: {bench.min_time:.3f}s / {bench.max_time:.3f}s")
            report.append(f"  Std Dev: {bench.std_dev:.3f}s")
            report.append(f"  Iterations: {bench.iterations}")
            
        # Resource Usage
        report.append("\nResource Usage:")
        for suite_name, suite in self.test_suites.items():
            report.append(f"\n{suite_name}:")
            report.append(f"  Memory: {suite.memory_usage:.1f} MB")
            report.append(f"  CPU: {suite.cpu_usage:.1f}%")
            
        return "\n".join(report)

    def save_metrics(self) -> None:
        """Save metrics to JSON file."""
        metrics = {
            "test_suites": {
                name: vars(suite)
                for name, suite in self.test_suites.items()
            },
            "benchmarks": [vars(bench) for bench in self.benchmarks]
        }
        
        metrics_file = self.reports_dir / "test_metrics.json"
        with open(metrics_file, "w") as f:
            json.dump(metrics, f, indent=2)
        logger.info(f"Metrics saved to {metrics_file}")

def main():
    """Main entry point."""
    project_root = Path(__file__).parent.parent
    collector = TestMetricsCollector(project_root)
    
    # Collect all metrics
    collector.collect_junit_metrics()
    collector.collect_coverage_metrics()
    collector.collect_benchmark_results()
    collector.collect_resource_metrics()
    
    # Generate and print report
    report = collector.generate_metrics_report()
    print(report)
    
    # Save metrics to file
    collector.save_metrics()

if __name__ == "__main__":
    main()
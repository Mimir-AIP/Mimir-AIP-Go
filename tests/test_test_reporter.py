"""Test module for test_reporter.py

This module contains tests for:
- Test result parsing from XML files
- Coverage data collection
- Performance metrics aggregation
- Report generation with visualizations

Test Categories:
- Unit tests for core functionality
- Integration tests with test runners

Prerequisites:
- Sample test result XML files in fixtures
- Coverage data files
"""

import os
import pytest
import tempfile
import xml.etree.ElementTree as ET
from datetime import datetime
from pathlib import Path
from unittest.mock import patch, MagicMock

from src.test_reporter import (
    TestReporter,
    CoverageCollector,
    MetricsAggregator,
    ReportGenerator,
    TestResult,
    CoverageData
)

@pytest.fixture
def sample_pytest_xml():
    """Create a sample pytest results XML file."""
    xml_content = '''<?xml version="1.0" encoding="utf-8"?>
<testsuites>
  <testsuite errors="0" failures="1" hostname="localhost" name="pytest" skipped="1" tests="3" time="1.234">
    <testcase classname="test_module" name="test_success" time="0.123"/>
    <testcase classname="test_module" name="test_failure" time="0.234">
      <failure message="assertion failed">Test failed</failure>
    </testcase>
    <testcase classname="test_module" name="test_skipped" time="0.001">
      <skipped message="skipped test">Skipped</skipped>
    </testcase>
  </testsuite>
</testsuites>
'''
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.xml') as f:
        f.write(xml_content)
    yield f.name
    os.unlink(f.name)

@pytest.fixture
def sample_jest_xml():
    """Create a sample Jest results XML file."""
    xml_content = '''<?xml version="1.0" encoding="utf-8"?>
<testsuites>
  <testsuite name="Jest Tests" tests="3" failures="1" errors="0" skipped="1">
    <testcase classname="ComponentTest" name="renders correctly" time="0.145"/>
    <testcase classname="ComponentTest" name="handles click" time="0.234">
      <failure message="Failed">Test failed</failure>
    </testcase>
    <testcase classname="ComponentTest" name="skipped test" time="0.001">
      <skipped message="Not implemented">Skipped</skipped>
    </testcase>
  </testsuite>
</testsuites>
'''
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.xml') as f:
        f.write(xml_content)
    yield f.name
    os.unlink(f.name)

@pytest.fixture
def temp_report_dir():
    """Create a temporary directory for test reports."""
    with tempfile.TemporaryDirectory() as temp_dir:
        yield temp_dir

class TestTestReporter:
    """Test suite for TestReporter class."""

    def test_initialization(self, temp_report_dir):
        """Test TestReporter initialization."""
        reporter = TestReporter(temp_report_dir)
        assert isinstance(reporter.coverage_collector, CoverageCollector)
        assert isinstance(reporter.metrics_aggregator, MetricsAggregator)
        assert isinstance(reporter.report_generator, ReportGenerator)

    def test_parse_pytest_results(self, temp_report_dir, sample_pytest_xml):
        """Test parsing pytest XML results."""
        reporter = TestReporter(temp_report_dir)
        reporter.parse_pytest_results(sample_pytest_xml)
        
        # Verify test results are parsed correctly
        assert len(reporter.test_results['unit']) == 3
        results = reporter.test_results['unit']
        
        assert any(r.name == "test_module.test_success" and r.result == "success" for r in results)
        assert any(r.name == "test_module.test_failure" and r.result == "failure" for r in results)
        assert any(r.name == "test_module.test_skipped" and r.result == "skipped" for r in results)

    def test_parse_jest_results(self, temp_report_dir, sample_jest_xml):
        """Test parsing Jest XML results."""
        reporter = TestReporter(temp_report_dir)
        reporter.parse_jest_results(sample_jest_xml)
        
        # Verify test results are parsed correctly
        assert len(reporter.test_results['unit']) == 3
        results = reporter.test_results['unit']
        
        assert any(r.name == "ComponentTest.renders correctly" and r.result == "success" for r in results)
        assert any(r.name == "ComponentTest.handles click" and r.result == "failure" for r in results)
        assert any(r.name == "ComponentTest.skipped test" and r.result == "skipped" for r in results)

    @patch('src.test_reporter.CoverageCollector')
    def test_collect_coverage(self, mock_collector, temp_report_dir):
        """Test coverage data collection."""
        reporter = TestReporter(temp_report_dir)
        reporter.collect_coverage()
        
        assert mock_collector().collect_python_coverage.called
        assert mock_collector().collect_js_coverage.called

    @patch('src.test_reporter.ReportGenerator')
    def test_generate_report(self, mock_generator, temp_report_dir):
        """Test report generation."""
        reporter = TestReporter(temp_report_dir)
        reporter.generate_report()
        
        assert mock_generator().create_performance_graphs.called
        assert mock_generator().generate_html_report.called

class TestCoverageCollector:
    """Test suite for CoverageCollector class."""

    def test_initialization(self, temp_report_dir):
        """Test CoverageCollector initialization."""
        collector = CoverageCollector(temp_report_dir)
        assert collector.coverage_dir == Path(temp_report_dir)
        assert isinstance(collector.python_coverage, dict)
        assert isinstance(collector.js_coverage, dict)

    @patch('coverage.Coverage')
    def test_collect_python_coverage(self, mock_coverage, temp_report_dir):
        """Test Python coverage collection."""
        collector = CoverageCollector(temp_report_dir)
        
        # Mock coverage data
        mock_coverage().data.measured_files.return_value = ['file1.py']
        mock_coverage().analysis.return_value = ([], [1, 2, 3], [2])  # statements, excluded, missing
        mock_coverage().report.return_value = 66.67  # coverage percentage
        
        collector.collect_python_coverage()
        
        assert 'file1.py' in collector.python_coverage
        coverage_data = collector.python_coverage['file1.py']
        assert coverage_data.statements == 3
        assert coverage_data.covered == 2
        assert coverage_data.percentage == 66.67

class TestMetricsAggregator:
    """Test suite for MetricsAggregator class."""

    def test_initialization(self):
        """Test MetricsAggregator initialization."""
        aggregator = MetricsAggregator()
        assert isinstance(aggregator.metrics, dict)
        assert 'timing' in aggregator.metrics
        assert 'memory' in aggregator.metrics
        assert 'cpu' in aggregator.metrics

    def test_add_timing(self):
        """Test adding timing data."""
        aggregator = MetricsAggregator()
        aggregator.add_timing('test_case', 1.234)
        assert aggregator.metrics['timing']['test_case'] == 1.234

    def test_get_summary(self):
        """Test metrics summary generation."""
        aggregator = MetricsAggregator()
        aggregator.start_monitoring()
        aggregator.add_timing('test1', 1.0)
        aggregator._record_metrics()  # Add some sample metrics
        
        summary = aggregator.get_summary()
        assert 'total_duration' in summary
        assert 'max_memory' in summary
        assert 'avg_memory' in summary
        assert 'max_cpu' in summary
        assert 'avg_cpu' in summary
        assert 'timing_data' in summary

class TestReportGenerator:
    """Test suite for ReportGenerator class."""

    def test_initialization(self, temp_report_dir):
        """Test ReportGenerator initialization."""
        generator = ReportGenerator(temp_report_dir)
        assert generator.output_dir == Path(temp_report_dir)
        assert generator.output_dir.exists()

    def test_create_performance_graphs(self, temp_report_dir):
        """Test performance graph creation."""
        generator = ReportGenerator(temp_report_dir)
        metrics = {
            'timing_data': {'test1': 1.0, 'test2': 2.0},
            'memory': [100, 200, 300],
            'cpu': [10, 20, 30]
        }
        
        graphs = generator.create_performance_graphs(metrics)
        assert 'timing' in graphs
        assert 'resources' in graphs
        assert Path(graphs['timing']).exists()
        assert Path(graphs['resources']).exists()

    def test_generate_html_report(self, temp_report_dir):
        """Test HTML report generation."""
        generator = ReportGenerator(temp_report_dir)
        
        test_results = {
            'unit': [TestResult('test1', 'success', 1.0)],
            'integration': [TestResult('test2', 'failure', 2.0, 'Failed')]
        }
        
        coverage_data = {
            'Python': {'file1.py': CoverageData(10, 8, 80.0, [3, 5])},
            'JavaScript': {'file1.js': CoverageData(20, 15, 75.0, [1, 2, 3])}
        }
        
        metrics = {
            'total_duration': 10.0,
            'max_memory': 500,
            'avg_cpu': 25.0,
            'timing_data': {'test1': 1.0, 'test2': 2.0}
        }
        
        graphs = {
            'timing': str(Path(temp_report_dir) / 'timing.png'),
            'resources': str(Path(temp_report_dir) / 'resources.png')
        }
        
        report_path = generator.generate_html_report(
            test_results=test_results,
            coverage_data=coverage_data,
            metrics=metrics,
            graphs=graphs
        )
        
        assert Path(report_path).exists()
        with open(report_path) as f:
            content = f.read()
            assert 'Test Results Report' in content
            assert 'test1' in content
            assert 'test2' in content
            assert 'Success' in content
            assert 'Failure' in content
            assert 'file1.py' in content
            assert 'file1.js' in content

if __name__ == '__main__':
    pytest.main(['-v', __file__])
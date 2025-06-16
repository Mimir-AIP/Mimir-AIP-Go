#!/bin/bash

# Comprehensive test runner for Python and JavaScript tests
# Follows project test standards for organization and reporting

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'  # No Color

# Default values
PARALLEL_JOBS=4
TIMEOUT=300  # 5 minutes
TEST_PATTERN="test_*"
COVERAGE_THRESHOLD=90
PERFORMANCE_THRESHOLD_OPS=1000
PERFORMANCE_THRESHOLD_MEM=512  # MB
TEST_TYPES="unit integration performance"

# Temporary directories
TEMP_DIR="./temp_test_data"
COVERAGE_DIR="./coverage"
REPORT_DIR="./test_reports"

# Log file
LOG_FILE="$REPORT_DIR/test_run.log"

# Help message
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -t, --test-type     Specify test type(s) to run (unit,integration,performance)"
    echo "  -p, --pattern       Test file pattern to match (default: test_*)"
    echo "  -j, --jobs          Number of parallel jobs (default: 4)"
    echo "  --timeout           Test timeout in seconds (default: 300)"
    echo "  --no-coverage       Skip coverage reporting"
    echo "  --no-cleanup        Keep temporary files after testing"
    echo "  -h, --help          Show this help message"
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--test-type)
            TEST_TYPES="$2"
            shift 2
            ;;
        -p|--pattern)
            TEST_PATTERN="$2"
            shift 2
            ;;
        -j|--jobs)
            PARALLEL_JOBS="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --no-coverage)
            SKIP_COVERAGE=1
            shift
            ;;
        --no-cleanup)
            SKIP_CLEANUP=1
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Logging function
log() {
    local level=$1
    shift
    local message=$@
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

# Error handling
error_handler() {
    local exit_code=$?
    log "ERROR" "An error occurred on line $1"
    cleanup
    exit $exit_code
}

trap 'error_handler ${LINENO}' ERR

# Setup function
setup() {
    log "INFO" "Setting up test environment..."
    
    # Create required directories
    mkdir -p "$TEMP_DIR" "$COVERAGE_DIR" "$REPORT_DIR"
    
    # Set up Python virtual environment if needed
    if [[ ! -d "venv" ]]; then
        log "INFO" "Creating Python virtual environment..."
        python -m venv venv
        source venv/Scripts/activate
        pip install -r requirements.txt
        pip install pytest pytest-cov pytest-xdist pytest-benchmark pytest-timeout
    else
        source venv/Scripts/activate
    fi
    
    # Install Node.js dependencies if needed
    if [[ -f "package.json" ]]; then
        log "INFO" "Installing Node.js dependencies..."
        npm install
        npm install -g jest jest-junit
    fi
    
    # Set environment variables
    export PYTHONPATH="$PYTHONPATH:$(pwd)/src"
    export TEST_TEMP_DIR="$TEMP_DIR"
    export TEST_MODE=1
}

# Run Python tests
run_python_tests() {
    local test_type=$1
    local test_dir="tests/$test_type"
    local report_file="$REPORT_DIR/${test_type}_python.xml"
    
    log "INFO" "Running Python $test_type tests..."
    
    PYTEST_ARGS=(
        -v
        --junitxml="$report_file"
        --timeout="$TIMEOUT"
        -n "$PARALLEL_JOBS"
    )
    
    # Add coverage for non-performance tests
    if [[ "$test_type" != "performance" && -z "$SKIP_COVERAGE" ]]; then
        PYTEST_ARGS+=(
            --cov=src
            --cov-report=term-missing
            --cov-report="html:$COVERAGE_DIR/python"
            --cov-fail-under="$COVERAGE_THRESHOLD"
        )
    fi
    
    # Add test directory
    PYTEST_ARGS+=("$test_dir")
    
    python -m pytest "${PYTEST_ARGS[@]}" || return 1
}

# Run JavaScript tests
run_js_tests() {
    local test_type=$1
    local test_dir="tests/$test_type"
    local report_file="$REPORT_DIR/${test_type}_js.xml"
    
    log "INFO" "Running JavaScript $test_type tests..."
    
    JEST_ARGS=(
        --verbose
        --runInBand
        --ci
        --reporters=default
        --reporters=jest-junit
        --testTimeout="$TIMEOUT"000
    )
    
    # Add coverage for non-performance tests
    if [[ "$test_type" != "performance" && -z "$SKIP_COVERAGE" ]]; then
        JEST_ARGS+=(
            --coverage
            --coverageDirectory="$COVERAGE_DIR/js"
            --coverageThreshold="{\"global\":{\"statements\":$COVERAGE_THRESHOLD}}"
        )
    fi
    
    # Add test directory
    JEST_ARGS+=("$test_dir")
    
    JEST_JUNIT_OUTPUT_DIR="$REPORT_DIR" \
    JEST_JUNIT_OUTPUT_NAME="${test_type}_js.xml" \
    jest "${JEST_ARGS[@]}" || return 1
}

# Generate HTML report
generate_report() {
    log "INFO" "Generating test report..."
    
    # Combine test results
    python - <<EOF
import json
import xml.etree.ElementTree as ET
from pathlib import Path
import datetime

def parse_junit_xml(file_path):
    tree = ET.parse(file_path)
    root = tree.getroot()
    return {
        'tests': int(root.attrib.get('tests', 0)),
        'failures': int(root.attrib.get('failures', 0)),
        'errors': int(root.attrib.get('errors', 0)),
        'skipped': int(root.attrib.get('skipped', 0)),
        'time': float(root.attrib.get('time', 0))
    }

def generate_html_report():
    report_dir = Path("$REPORT_DIR")
    coverage_dir = Path("$COVERAGE_DIR")
    
    # Collect test results
    results = {
        'python': {'unit': {}, 'integration': {}, 'performance': {}},
        'js': {'unit': {}, 'integration': {}, 'performance': {}}
    }
    
    # Parse all XML files
    for xml_file in report_dir.glob('*.xml'):
        name = xml_file.stem
        test_type, lang = name.split('_')
        results[lang][test_type] = parse_junit_xml(xml_file)
    
    # Generate HTML
    with open(report_dir / 'index.html', 'w') as f:
        f.write(f'''
<!DOCTYPE html>
<html>
<head>
    <title>Test Results</title>
    <style>
        body {{ font-family: Arial, sans-serif; margin: 2em; }}
        .card {{ border: 1px solid #ddd; padding: 1em; margin: 1em 0; border-radius: 4px; }}
        .success {{ color: green; }}
        .failure {{ color: red; }}
        .warning {{ color: orange; }}
    </style>
</head>
<body>
    <h1>Test Results</h1>
    <p>Generated on: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
    
    <div class="card">
        <h2>Summary</h2>
        <p>Total tests: {sum(r['tests'] for lang in results.values() for r in lang.values())}</p>
        <p>Total failures: {sum(r['failures'] for lang in results.values() for r in lang.values())}</p>
        <p>Total errors: {sum(r['errors'] for lang in results.values() for r in lang.values())}</p>
    </div>
''')
        
        # Add detailed results
        for lang in results:
            f.write(f'<div class="card"><h2>{lang.upper()} Results</h2>')
            for test_type, data in results[lang].items():
                if data:
                    status = 'success' if data['failures'] + data['errors'] == 0 else 'failure'
                    f.write(f'''
                        <h3>{test_type.title()}</h3>
                        <p class="{status}">
                            Tests: {data['tests']}<br>
                            Failures: {data['failures']}<br>
                            Errors: {data['errors']}<br>
                            Skipped: {data['skipped']}<br>
                            Time: {data['time']:.2f}s
                        </p>
                    ''')
            f.write('</div>')
        
        # Add coverage information
        f.write('''
            <div class="card">
                <h2>Coverage Reports</h2>
                <p><a href="../coverage/python/index.html">Python Coverage Report</a></p>
                <p><a href="../coverage/js/lcov-report/index.html">JavaScript Coverage Report</a></p>
            </div>
        ''')
        
        f.write('</body></html>')

generate_html_report()
EOF
}

# Cleanup function
cleanup() {
    if [[ -z "$SKIP_CLEANUP" ]]; then
        log "INFO" "Cleaning up..."
        rm -rf "$TEMP_DIR"
    fi
}

# Main execution
main() {
    log "INFO" "Starting test execution..."
    
    # Setup environment
    setup
    
    # Track overall success
    local success=true
    
    # Run tests for each specified type
    for test_type in ${TEST_TYPES//,/ }; do
        if [[ ! -d "tests/$test_type" ]]; then
            log "WARNING" "Test directory tests/$test_type does not exist, skipping..."
            continue
        fi
        
        # Run Python tests if they exist
        if ls "tests/$test_type"/*.py >/dev/null 2>&1; then
            if ! run_python_tests "$test_type"; then
                success=false
                log "ERROR" "Python $test_type tests failed"
            fi
        fi
        
        # Run JavaScript tests if they exist
        if ls "tests/$test_type"/*.js >/dev/null 2>&1; then
            if ! run_js_tests "$test_type"; then
                success=false
                log "ERROR" "JavaScript $test_type tests failed"
            fi
        fi
    done
    
    # Generate combined report
    generate_report
    
    # Cleanup
    cleanup
    
    # Final status
    if [[ "$success" == true ]]; then
        log "INFO" "All tests completed successfully"
        exit 0
    else
        log "ERROR" "Some tests failed"
        exit 1
    fi
}

# Create log file directory
mkdir -p "$(dirname "$LOG_FILE")"

# Start execution
main 2>&1 | tee -a "$LOG_FILE"
"""Test Standards Validation Script

This script validates that all test files comply with the project's testing standards:
- Naming conventions
- Directory structure
- Documentation requirements
- Test coverage requirements
"""

import os
import re
import ast
import logging
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class TestValidationResult:
    """Stores validation results for a test file."""
    file_path: Path
    has_docstring: bool
    test_count: int
    coverage: float
    issues: List[str]

class TestStandardsValidator:
    """Validates test files against project standards."""

    def __init__(self, project_root: Path):
        """Initialize validator.
        
        Args:
            project_root: Root directory of the project
        """
        self.project_root = project_root
        self.test_dir = project_root / "tests"
        self.required_dirs = ["unit", "integration", "performance"]
        self.results: Dict[Path, TestValidationResult] = {}

    def validate_directory_structure(self) -> List[str]:
        """Validate test directory structure.
        
        Returns:
            List of validation issues found
        """
        issues = []
        for required_dir in self.required_dirs:
            dir_path = self.test_dir / required_dir
            if not dir_path.exists():
                issues.append(f"Missing required test directory: {required_dir}")
        return issues

    def validate_file_naming(self, test_file: Path) -> List[str]:
        """Validate test file naming conventions.
        
        Args:
            test_file: Path to test file
            
        Returns:
            List of validation issues found
        """
        issues = []
        if not test_file.name.startswith("test_"):
            issues.append(f"Test file {test_file} doesn't follow 'test_' prefix convention")
        if not test_file.name.endswith(".py"):
            issues.append(f"Test file {test_file} must have .py extension")
        return issues

    def validate_test_documentation(self, content: str) -> Tuple[bool, List[str]]:
        """Validate test documentation requirements.
        
        Args:
            content: Content of test file
            
        Returns:
            Tuple of (has_docstring, list of issues)
        """
        issues = []
        module = ast.parse(content)
        
        # Check module docstring
        has_docstring = ast.get_docstring(module) is not None
        if not has_docstring:
            issues.append("Missing module docstring")
            
        # Check class and function docstrings
        for node in ast.walk(module):
            if isinstance(node, (ast.ClassDef, ast.FunctionDef)):
                if not ast.get_docstring(node):
                    issues.append(f"Missing docstring for {node.name}")
                    
        return has_docstring, issues

    def count_tests(self, content: str) -> int:
        """Count number of test methods in file.
        
        Args:
            content: Content of test file
            
        Returns:
            Number of test methods found
        """
        module = ast.parse(content)
        test_count = 0
        
        for node in ast.walk(module):
            if isinstance(node, ast.FunctionDef):
                if node.name.startswith("test_"):
                    test_count += 1
                    
        return test_count

    def validate_test_file(self, test_file: Path) -> TestValidationResult:
        """Validate a single test file.
        
        Args:
            test_file: Path to test file
            
        Returns:
            TestValidationResult with validation details
        """
        issues = []
        
        # Check naming
        issues.extend(self.validate_file_naming(test_file))
        
        # Read and validate content
        content = test_file.read_text()
        has_docstring, doc_issues = self.validate_test_documentation(content)
        issues.extend(doc_issues)
        
        # Count tests
        test_count = self.count_tests(content)
        if test_count == 0:
            issues.append("No test methods found")
            
        # Get coverage from .coverage file if exists
        coverage = self.get_file_coverage(test_file)
        
        return TestValidationResult(
            file_path=test_file,
            has_docstring=has_docstring,
            test_count=test_count,
            coverage=coverage,
            issues=issues
        )

    def get_file_coverage(self, test_file: Path) -> float:
        """Get coverage percentage for a test file.
        
        Args:
            test_file: Path to test file
            
        Returns:
            Coverage percentage (0-100)
        """
        # TODO: Implement coverage parsing from .coverage file
        return 0.0

    def validate_all(self) -> Dict[Path, TestValidationResult]:
        """Validate all test files in the project.
        
        Returns:
            Dictionary mapping file paths to validation results
        """
        # Check directory structure
        structure_issues = self.validate_directory_structure()
        if structure_issues:
            for issue in structure_issues:
                logger.error(issue)
                
        # Validate each test file
        for test_file in self.test_dir.rglob("test_*.py"):
            result = self.validate_test_file(test_file)
            self.results[test_file] = result
            
            # Log issues
            if result.issues:
                for issue in result.issues:
                    logger.error(f"{test_file}: {issue}")
                    
        return self.results

    def generate_report(self) -> str:
        """Generate validation report.
        
        Returns:
            Formatted report string
        """
        report = ["Test Standards Validation Report", "=" * 30, ""]
        
        # Directory structure
        report.append("Directory Structure:")
        structure_issues = self.validate_directory_structure()
        if structure_issues:
            for issue in structure_issues:
                report.append(f"  ❌ {issue}")
        else:
            report.append("  ✓ All required directories present")
        report.append("")
        
        # File results
        report.append("Test Files:")
        for file_path, result in self.results.items():
            report.append(f"\n{file_path}:")
            report.append(f"  Tests: {result.test_count}")
            report.append(f"  Coverage: {result.coverage:.1f}%")
            report.append(f"  Docstring: {'✓' if result.has_docstring else '❌'}")
            if result.issues:
                report.append("  Issues:")
                for issue in result.issues:
                    report.append(f"    - {issue}")
                    
        return "\n".join(report)

def main():
    """Main entry point."""
    project_root = Path(__file__).parent.parent
    validator = TestStandardsValidator(project_root)
    validator.validate_all()
    report = validator.generate_report()
    print(report)

if __name__ == "__main__":
    main()
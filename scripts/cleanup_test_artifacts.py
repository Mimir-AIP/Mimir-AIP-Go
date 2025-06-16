"""Test Artifacts Cleanup Script

This script handles cleanup of test artifacts including:
- Temporary test files
- Coverage reports
- Test reports
- Cache directories
- Log files
"""

import os
import shutil
import logging
from pathlib import Path
from typing import List, Set
import time

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class TestArtifactsCleaner:
    """Handles cleanup of test artifacts."""

    def __init__(self, project_root: Path):
        """Initialize cleaner.
        
        Args:
            project_root: Root directory of the project
        """
        self.project_root = project_root
        self.cleaned_paths: Set[Path] = set()
        
        # Patterns for files to clean
        self.temp_patterns = ["*.tmp", "*.temp", "*.pyc", "__pycache__"]
        self.report_patterns = ["*.coverage", ".coverage.*", "coverage.*", "test-results.*"]
        self.log_patterns = ["*.log"]

    def get_cleanup_paths(self) -> List[Path]:
        """Get list of paths to clean.
        
        Returns:
            List of paths that should be cleaned
        """
        cleanup_paths = []
        
        # Standard cleanup directories
        cleanup_dirs = [
            self.project_root / ".pytest_cache",
            self.project_root / "__pycache__",
            self.project_root / ".coverage_cache",
            self.project_root / "htmlcov",
            self.project_root / "reports",
            self.project_root / ".tox"
        ]
        
        # Add paths if they exist
        for dir_path in cleanup_dirs:
            if dir_path.exists():
                cleanup_paths.append(dir_path)
                
        # Find temporary files
        patterns = self.temp_patterns + self.report_patterns + self.log_patterns
        for pattern in patterns:
            cleanup_paths.extend(self.project_root.rglob(pattern))
            
        return cleanup_paths

    def is_safe_to_delete(self, path: Path) -> bool:
        """Check if it's safe to delete a path.
        
        Args:
            path: Path to check
            
        Returns:
            True if path can be safely deleted
        """
        # Never delete source code
        if path.suffix in {".py", ".js", ".ts", ".jsx", ".tsx"}:
            return False
            
        # Never delete git directory
        if ".git" in path.parts:
            return False
            
        # Never delete actual test files
        if path.name.startswith("test_") and path.suffix == ".py":
            return False
            
        # Never delete configuration files
        if path.name in {
            "pytest.ini",
            "tox.ini",
            ".coveragerc",
            "jest.config.js",
            "package.json"
        }:
            return False
            
        return True

    def remove_path(self, path: Path) -> None:
        """Safely remove a file or directory.
        
        Args:
            path: Path to remove
        """
        try:
            if path.is_file():
                path.unlink()
                logger.info(f"Removed file: {path}")
            elif path.is_dir():
                shutil.rmtree(path)
                logger.info(f"Removed directory: {path}")
            self.cleaned_paths.add(path)
        except Exception as e:
            logger.error(f"Failed to remove {path}: {str(e)}")

    def cleanup_old_artifacts(self, max_age_days: int = 7) -> None:
        """Clean up artifacts older than specified age.
        
        Args:
            max_age_days: Maximum age in days for artifacts to keep
        """
        now = time.time()
        max_age_secs = max_age_days * 24 * 60 * 60
        
        for path in self.get_cleanup_paths():
            try:
                if not path.exists():
                    continue
                    
                # Check file age
                mtime = path.stat().st_mtime
                age = now - mtime
                
                if age > max_age_secs and self.is_safe_to_delete(path):
                    self.remove_path(path)
            except Exception as e:
                logger.error(f"Error processing {path}: {str(e)}")

    def cleanup_all_artifacts(self) -> None:
        """Clean up all test artifacts regardless of age."""
        for path in self.get_cleanup_paths():
            if path.exists() and self.is_safe_to_delete(path):
                self.remove_path(path)

    def generate_report(self) -> str:
        """Generate cleanup report.
        
        Returns:
            Formatted report string
        """
        report = ["Test Artifacts Cleanup Report", "=" * 30, ""]
        
        if not self.cleaned_paths:
            report.append("No artifacts were cleaned")
            return "\n".join(report)
            
        # Group by type
        files = [p for p in self.cleaned_paths if p.is_file()]
        dirs = [p for p in self.cleaned_paths if p.is_dir()]
        
        # Report directories
        if dirs:
            report.append("Cleaned Directories:")
            for dir_path in sorted(dirs):
                report.append(f"  - {dir_path}")
            report.append("")
            
        # Report files by type
        if files:
            by_suffix = {}
            for file_path in files:
                suffix = file_path.suffix or "no extension"
                by_suffix.setdefault(suffix, []).append(file_path)
                
            report.append("Cleaned Files:")
            for suffix, paths in sorted(by_suffix.items()):
                report.append(f"\n{suffix}:")
                for path in sorted(paths):
                    report.append(f"  - {path}")
                    
        # Summary
        report.append(f"\nTotal Cleaned: {len(self.cleaned_paths)}")
        report.append(f"Files: {len(files)}")
        report.append(f"Directories: {len(dirs)}")
        
        return "\n".join(report)

def main():
    """Main entry point."""
    project_root = Path(__file__).parent.parent
    cleaner = TestArtifactsCleaner(project_root)
    
    # Clean artifacts older than 7 days
    cleaner.cleanup_old_artifacts()
    
    # Generate and print report
    report = cleaner.generate_report()
    print(report)

if __name__ == "__main__":
    main()
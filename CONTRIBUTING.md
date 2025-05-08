# Contributing to Mimir-AIP

First off, thank you for considering contributing to Mimir-AIP! It's people like you that make Mimir-AIP such a great tool.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

* Use a clear and descriptive title
* Describe the exact steps to reproduce the problem
* Provide specific examples
* Describe the behavior you observed and what behavior you expected
* Include logs and stack traces if applicable
* Note your environment (OS, Python version, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

* Use a clear and descriptive title
* Provide a detailed description of the proposed functionality
* Include examples of how the feature would be used
* Explain why this enhancement would be useful
* List any alternatives you've considered

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows our coding standards
6. Submit the pull request!

## Development Process

1. Set up your development environment:
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   pip install -r requirements.txt
   ```

2. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. Make your changes:
   - Follow our coding standards
   - Add tests for new functionality
   - Update documentation as needed

4. Test your changes:
   - Run the test suite
   - Test in both normal and test modes
   - Verify all CI checks pass

5. Submit your pull request

## Coding Standards

### Python Style Guide

* Follow [PEP 8](https://www.python.org/dev/peps/pep-0008/)
* Use type hints for function arguments and return values
* Write descriptive docstrings following [PEP 257](https://www.python.org/dev/peps/pep-0257/)
* Keep functions focused and modular
* Use meaningful variable names

### Documentation

* Update README.md if needed
* Add docstrings to all public classes and methods
* Update Documentation.md for significant changes
* Include example YAML configurations where appropriate

### Testing

* Write unit tests for new functionality
* Include integration tests where appropriate
* Provide example pipelines for new features
* Test both success and failure cases
* Verify test mode functionality

### Git Commit Messages

* Use clear and descriptive commit messages
* Start with a concise summary line
* Include more detailed explanation if needed
* Reference issues and pull requests where appropriate

## Plugin Development Guidelines

When creating new plugins:

1. Choose the appropriate plugin type directory
2. Implement the BasePlugin interface
3. Include comprehensive configuration validation
4. Add detailed logging
5. Write thorough tests
6. Document all configuration options
7. Provide example pipeline configurations

## Need Help?

* Check the [Documentation](Documentation.md)
* Look through existing issues
* Join our community discussions
* Ask questions in the appropriate channels

## Recognition

Contributors who make significant improvements will be recognized in our documentation and release notes.

Thank you for contributing to Mimir-AIP!
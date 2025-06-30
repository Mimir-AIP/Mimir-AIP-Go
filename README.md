# Mimir-AIP

<img src="Docs/Assets/mimir-aip-svg-banners.svg" alt="Mimir AIP- Building the future of intelligent data pipelines"/>

[![License](https://img.shields.io/badge/License-GPL%203.0-blue.svg)](LICENSE)
[![Python](https://img.shields.io/badge/Python-3.8%2B-blue)](https://www.python.org/downloads/)
[![Documentation](https://img.shields.io/badge/docs-Documentation.md-green)](Documentation.md)
![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/Mimir-AIP/Mimir-AIP?utm_source=oss&utm_medium=github&utm_campaign=Mimir-AIP%2FMimir-AIP&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

## Overview

Mimir-AIP (Artificial Intelligence Platform) is a powerful open-source framework designed to seamlessly connect data inputs and outputs with AI systems. Built with modularity and flexibility at its core, it enables rapid development of AI-powered data pipelines.

### Key Features

- 🔌 **Modular Plugin System**: Easily extend functionality with plugins for:
  - Data input from various sources (APIs, databases, files)
  - AI/ML processing with both local and cloud models
  - Custom output formats and destinations
  - Data transformation and analysis
  - Interactive web interface with real-time updates

- 🤖 **LLM Integration**: 
  - Support for both local and cloud-hosted LLMs
  - Easy integration with OpenAI, OpenRouter, and other providers
  - Flexible prompt management and response handling

- 📊 **Advanced Processing**:
  - Video and image processing capabilities
  - Real-time data streaming
  - Customizable data transformation pipelines
  - Report generation with multiple formats
  - **Context Management**:
    - Thread-safe operations using locking
    - Versioning via snapshots
    - Configurable conflict resolution (overwrite/keep/merge)
    - Dictionary merging support

- 🛠️ **Developer-Friendly**:
  - YAML-based pipeline configuration
  - Comprehensive testing support
  - Detailed logging and error handling
  - Environment-based configuration

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Mimir-AIP/Mimir-AIP.git
cd Mimir-AIP
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

3. Configure your environment:
```bash
cp src/Plugins/AIModels/AzureAI/.env.template .env
# Edit .env with your API keys
```

4. Run your first pipeline:
```bash
python src/main.py
```

## Architecture

```mermaid
flowchart TD
    subgraph subGraph0["Input Plugins"]
        A["Data Connection sources and plugins"]
    end
    subgraph subGraph1["Output Plugins"]
        E["Output plugins for graphs & visuals"]
    end
    A -->D["Data_Processing Plugins"]
    D-->B["LLM and other AI models"]
    B --> C["Storage"] & E
    C --> B
```

## Documentation

- [Full Documentation](Documentation.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Community & Support

- 🌟 Star this repository to show your support
- 🐛 Report issues via [GitHub Issues](https://github.com/Mimir-AIP/Mimir-AIP/issues)
- 💡 Submit feature requests through [GitHub Discussions](https://github.com/Mimir-AIP/Mimir-AIP/discussions)
- 🤝 Contribute via [Pull Requests](https://github.com/Mimir-AIP/Mimir-AIP/pulls)

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Mimir-AIP/Mimir-AIP&type=Date)](https://www.star-history.com/#Mimir-AIP/Mimir-AIP&Date)

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

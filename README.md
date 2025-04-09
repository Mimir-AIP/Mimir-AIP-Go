<img src="Docs/Assets/mimir-aip-svg-banners.svg" alt="Mimir AIP- Building the future of intelligent data pipelines"/>

Overview
---
The MIMIR Artificial Intelligence Platform is an open-source framework designed to connect inputs and outputs to AI systems with integrated storage capabilities. The platform is built to provide a modular and flexible architecture, allowing users to easily integrate various components and customize the system to suit their specific needs.

<mark>Warning</mark>
---
<mark>As of April 9, 2025, this repository has been made public. I am currently working on final changes and hope to have a Proof of Concept (POC) together in the coming months. In its current state, the platform is not functional, but it should provide an understanding of the intended goals of the project</mark>

If you're interested in this project, I encourage you to watch and star the repository to stay updated on progress and contribute to the development! Feel free to reach out if you have any questions or suggestions!

Modular Plugin System
---
MIMIR features a modular plugin system that enables users to extend the platform's functionality through the use of plugins. This system allows for the easy addition of new inputs, outputs, and data processing modules, making it simple to integrate with a wide range of data sources and AI systems.

- Inputs: Plugins can be created to connect to various data sources, such as databases, APIs, or file systems, allowing users to ingest data from multiple sources.
- Outputs: Output plugins enable the platform to send processed data to different destinations, including databases, messaging systems, or file systems.
- Data Processing: Data processing plugins can be used to perform tasks such as data transformation, filtering, and analysis, allowing users to customize the data processing pipeline.

Support for Large Language Models (LLMs)
---
MIMIR provides support for both locally hosted and remotely hosted Large Language Models (LLMs). This allows users to choose the deployment model that best suits their needs, whether it's hosting their own LLM or leveraging cloud-based services.

Open Alternative to Proprietary Solutions
---
The MIMIR Artificial Intelligence Platform is intended as an open alternative to proprietary solutions like Palantir AIP. By providing a flexible and customizable framework, MIMIR aims to democratize access to AI technology and enable users to build custom solutions that meet their specific requirements.

```mermaid
flowchart TD
 subgraph subGraph0["Input Plugins"]
        A["Data Connection sources and plugins"]
  end
 subgraph subGraph1["Output Plugins"]
        E["Output plugins for  graphs & visuals"]
  end
    A -->D["Data Processing Plugins"]
    D-->B["LLM and other AI models"]
    B --> C["Storage"] & E
    C --> B

```

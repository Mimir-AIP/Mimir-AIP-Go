<img src="Docs/Assets/mimir-aip-svg-banners.svg" alt="Mimir AIP- Building the future of intelligent data pipelines"/>

> This repo is a work in progress and is not in a functional state

Rewrite of https://github.com/Mimir-AIP/Mimir-AIP in the Go language, focusing on repliating existing functionality, improving performance, enabling agentic workflows and maintaining cross-compatability with pipeline yaml's created for the python variant.

```mermaid
flowchart TD
 subgraph subGraph0["Input Plugins"]
        A["Data Connection sources and plugins"]
  end
 subgraph subGraph1["Output Plugins"]
        E["Output plugins for  graphs & visuals"]
  end
    A -->D["Data_Processing Plugins"]
    D-->B["LLM and other AI models"]
    B --> C["Storage"] & E
    C --> B

```

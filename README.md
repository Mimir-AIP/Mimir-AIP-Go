# Mimir-AIP

<img src="Docs/Assets/mimir-aip-svg-banners.svg" alt="Mimir AIP- Building the future of intelligent data pipelines"/>

[![License](https://img.shields.io/badge/License-GPL%203.0-blue.svg)](LICENSE)
[![Python](https://img.shields.io/badge/Python-3.8%2B-blue)](https://www.python.org/downloads/)
[![Documentation](https://img.shields.io/badge/docs-Documentation.md-green)](Documentation.md)

## POC: BBC News Report Generator

This branch contains two versions of a Proof of Concept (POC) for generating BBC news reports. The pipeline:

1. Fetches news stories from the BBC News RSS feed
2. Processes each story to determine its importance
3. For important stories (score > 7):
   - Generates a section prompt
   - Creates a section summary
   - Collects the data
4. Writes all section summaries to a file
5. Loads the summaries and generates an HTML report

### Original Version Flowchart

```mermaid
graph TD
    Fetch_BBC_News_RSS_Feed[Fetch BBC News RSS Feed]
    Fetch_BBC_News_RSS_Feed --> Process_Stories
    Process_Stories[Process Stories]
    Determine_Importance[Determine Importance]
    Determine_Importance --> Process_Important_Story
    Process_Important_Story[Process Important Story]
    Generate_Search_Queries[Generate Search Queries]
    Generate_Search_Queries --> Fetch_Search_Results
    Fetch_Search_Results[Fetch Search Results]
    Fetch_Search_Results --> Scrape_Web_Results
    Scrape_Web_Results[Scrape Web Results]
    Scrape_Web_Results --> Generate_Report
    Generate_Report[Generate Report]
    Process_Important_Story_end[End]
    Generate_Report --> Process_Important_Story_end
    Process_Important_Story --> Process_Important_Story_end
    Process_Stories_end[End]
    Process_Important_Story --> Process_Stories_end
    Process_Stories --> Process_Stories_end
    Process_Stories --> Combine_Reports
    Combine_Reports[Combine Reports]
    Combine_Reports --> Generate_HTML_Report
    Generate_HTML_Report[Generate HTML Report]
```

### API Key Setup

The pipeline uses the OpenRouter plugin, which requires API keys. Follow these steps to set up your API keys:

1. Create a `.env` file in the `src/Plugins/AIModels/OpenRouter/` directory
2. Add your OpenRouter API key to the `.env` file:
   ```
   OPENROUTER_API_KEY=your_api_key_here
   ```

### Configuration

The `config.yaml` file controls which pipelines are enabled. You can configure it to run either the original or V2 version of the POC pipeline.

In this branch I have enabled both pipelines and you run the desired one via command line.

### Run Instructions

#### For Original Version
1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```
2. Run:
   ```bash
   python src/main.py --pipeline "BBC News Report Generator (Old)"
   ```

#### For V2 Version
1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```
2. Run:
   ```bash
   python src/main.py --pipeline "BBC News Report Generator (V2)"
   ```

Both versions will generate:
- `section_summaries.json` with the collected data
- `report.html` in the output directory
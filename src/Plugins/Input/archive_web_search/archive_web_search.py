import requests
import logging

class ArchiveWebSearchPlugin:
    """
    Plugin for querying the Internet Archive's advancedsearch API as an alternative to the DuckDuckGo web search plugin.
    """
    plugin_type = "Input"
    name = "archive-web-search"

    def __init__(self):
        self.base_url = "https://archive.org/advancedsearch.php"
        self.logger = logging.getLogger(self.__class__.__name__)

    def search(self, query, mediatype=None, max_results=10, advanced=None):
        """
        Perform a search on archive.org with the given query and optional mediatype filter.
        Returns a list of result dicts (title, identifier, mediatype, description, url).
        Supports advanced query parameters (e.g., date ranges, creators, etc.).
        """
        params = {
            "q": query,
            "output": "json",
            "rows": max_results
        }
        if mediatype:
            params["q"] += f" AND mediatype:{mediatype}"
        if advanced and isinstance(advanced, dict):
            # Add advanced query parameters (e.g., date ranges, creators)
            for k, v in advanced.items():
                params["q"] += f" AND {k}:{v}"
        self.logger.info(f"Querying Archive.org with: {params['q']}")
        response = requests.get(self.base_url, params=params)
        if response.status_code != 200:
            self.logger.error(f"Archive.org API error: {response.status_code}")
            return []
        data = response.json()
        docs = data.get("response", {}).get("docs", [])
        results = []
        for doc in docs:
            result = {
                "title": doc.get("title"),
                "identifier": doc.get("identifier"),
                "mediatype": doc.get("mediatype"),
                "description": doc.get("description"),
                "url": f"https://archive.org/details/{doc.get('identifier')}" if doc.get("identifier") else None
            }
            results.append(result)
        return results

    def extract_urls_from_response(self, response):
        """
        Extract all archive.org item URLs from the response (list or single result).
        """
        urls = []
        if isinstance(response, list):
            for item in response:
                url = item.get("url")
                if url:
                    urls.append(url)
        elif isinstance(response, dict):
            url = response.get("url")
            if url:
                urls.append(url)
        return urls

    def execute_pipeline_step(self, step_config, context):
        config = step_config["config"]
        query = config["query"]
        mediatype = config.get("mediatype")
        max_results = config.get("max_results", 10)
        advanced = config.get("advanced")
        extract_urls = config.get("extract_urls", False)

        # Support list of queries
        if isinstance(query, list):
            results = [self.search(q, mediatype=mediatype, max_results=max_results, advanced=advanced) for q in query]
            if extract_urls:
                urls = []
                for r in results:
                    urls.extend(self.extract_urls_from_response(r))
                return {step_config["output"]: urls}
            else:
                return {step_config["output"]: results}

        result = self.search(query, mediatype=mediatype, max_results=max_results, advanced=advanced)
        if extract_urls:
            urls = self.extract_urls_from_response(result)
            return {step_config["output"]: urls}
        else:
            return {step_config["output"]: result}

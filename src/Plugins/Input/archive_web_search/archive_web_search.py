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

    def search(self, query, mediatype=None, max_results=10, advanced=None, deduplicate=True):
        """
        Perform a search on archive.org with the given query and optional mediatype filter.
        Returns a list of result dicts (title, identifier, mediatype, description, url).
        Supports advanced query parameters (e.g., date ranges, creators, etc.).
        If deduplicate is True, removes duplicate URLs from the results.
        """
        print(f"[DEBUG] ArchiveWebSearchPlugin.search: query={query}, mediatype={mediatype}, max_results={max_results}, advanced={advanced}")  # DEBUG
        params = {
            "q": query,
            "output": "json",
            "rows": max_results
        }
        if mediatype:
            params["q"] += f" AND mediatype:{mediatype}"
        if advanced and isinstance(advanced, dict):
            for k, v in advanced.items():
                params["q"] += f" AND {k}:{v}"
        print(f"[DEBUG] ArchiveWebSearchPlugin.search: constructed params={params}")  # DEBUG
        response = requests.get(self.base_url, params=params)
        print(f"[DEBUG] ArchiveWebSearchPlugin.search: response.status_code={response.status_code}")  # DEBUG
        if response.status_code != 200:
            self.logger.error(f"Archive.org API error: {response.status_code}")
            return []
        data = response.json()
        print(f"[DEBUG] ArchiveWebSearchPlugin.search: raw API response={data}")  # DEBUG
        docs = data.get("response", {}).get("docs", [])
        results = []
        seen_urls = set()
        for doc in docs:
            url = f"https://archive.org/details/{doc.get('identifier')}" if doc.get("identifier") else None
            if deduplicate and url in seen_urls:
                continue
            if url:
                seen_urls.add(url)
            result = {
                "title": doc.get("title"),
                "identifier": doc.get("identifier"),
                "mediatype": doc.get("mediatype"),
                "description": doc.get("description"),
                "url": url
            }
            results.append(result)
        print(f"[DEBUG] ArchiveWebSearchPlugin.search: returning {len(results)} results")  # DEBUG
        return results

    def extract_urls_from_response(self, response, deduplicate=True):
        print(f"[DEBUG] extract_urls_from_response called with type: {type(response)}")  # DEBUG
        urls = []
        seen = set()
        # response can be a list of dicts or a dict
        if isinstance(response, list):
            for item in response:
                if not isinstance(item, dict):
                    print(f"[WARNING] Non-dict item encountered in response list: {item} (type: {type(item)})")
                    continue
                url = item.get("url")
                if url and (not deduplicate or url not in seen):
                    urls.append(url)
                    seen.add(url)
        elif isinstance(response, dict):
            url = response.get("url")
            if url:
                urls.append(url)
        else:
            print(f"[WARNING] Unexpected response type in extract_urls_from_response: {type(response)}. Value: {response}")
        print(f"[DEBUG] extract_urls_from_response returning URLs: {urls}")  # DEBUG
        return urls

    def execute_pipeline_step(self, step_config, context):
        print("[DEBUG] archive-web-search plugin execute_pipeline_step called")  # DEBUG
        config = step_config["config"]
        query = config["query"]
        mediatype = config.get("mediatype")
        max_results = config.get("max_results", 10)
        advanced = config.get("advanced")
        extract_urls = config.get("extract_urls", False)
        deduplicate = config.get("deduplicate", True)

        print(f"[DEBUG] archive-web-search plugin received query: {query}")  # DEBUG
        # Support list of queries
        if isinstance(query, list):
            results = [self.search(q, mediatype=mediatype, max_results=max_results, advanced=advanced, deduplicate=deduplicate) for q in query]
            if extract_urls:
                urls = []
                for r in results:
                    urls.extend(self.extract_urls_from_response(r, deduplicate=deduplicate))
                print(f"[DEBUG] archive-web-search plugin returning URLs (list): {urls}")  # DEBUG
                print(f"[DEBUG] archive-web-search plugin URLs type: {type(urls)}")  # DEBUG
                return {step_config["output"]: urls}
            else:
                print(f"[DEBUG] archive-web-search plugin returning results (list): {results}")  # DEBUG
                print(f"[DEBUG] archive-web-search plugin results type: {type(results)}")  # DEBUG
                return {step_config["output"]: results}

        result = self.search(query, mediatype=mediatype, max_results=max_results, advanced=advanced, deduplicate=deduplicate)
        if extract_urls:
            urls = self.extract_urls_from_response(result, deduplicate=deduplicate)
            print(f"[DEBUG] archive-web-search plugin returning URLs: {urls}")  # DEBUG
            print(f"[DEBUG] archive-web-search plugin URLs type: {type(urls)}")  # DEBUG
            return {step_config["output"]: urls}
        else:
            print(f"[DEBUG] archive-web-search plugin returning result: {result}")  # DEBUG
            print(f"[DEBUG] archive-web-search plugin result type: {type(result)}")  # DEBUG
            return {step_config["output"]: result}

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
        self.logger.debug(f"ArchiveWebSearchPlugin.search: query={query}, mediatype={mediatype}, max_results={max_results}, advanced={advanced}")
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
        self.logger.debug(f"ArchiveWebSearchPlugin.search: constructed params={params}")
        response = requests.get(self.base_url, params=params)
        self.logger.debug(f"ArchiveWebSearchPlugin.search: response.status_code={response.status_code}")
        if response.status_code != 200:
            self.logger.error(f"Archive.org API error: {response.status_code}")
            return []
        data = response.json()
        self.logger.debug(f"ArchiveWebSearchPlugin.search: raw API response={data}")
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
        self.logger.debug(f"ArchiveWebSearchPlugin.search: returning {len(results)} results")
        return results

    def extract_urls_from_response(self, response, deduplicate=True):
        self.logger.debug(f"extract_urls_from_response called with type: {type(response)}")
        urls = []
        seen = set()
        # response can be a list of dicts or a dict
        if isinstance(response, list):
            for item in response:
                if not isinstance(item, dict):
                    self.logger.warning(f"Non-dict item encountered in response list: {item} (type: {type(item)})")
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
            self.logger.warning(f"Unexpected response type in extract_urls_from_response: {type(response)}. Value: {response}")
        self.logger.debug(f"extract_urls_from_response returning URLs: {urls}")
        return urls

    def execute_pipeline_step(self, step_config, context):
        self.logger.debug("archive-web-search plugin execute_pipeline_step called")
        config = step_config["config"]
        query = config["query"]
        mediatype = config.get("mediatype")
        max_results = config.get("max_results", 10)
        advanced = config.get("advanced")
        extract_urls = config.get("extract_urls", False)
        deduplicate = config.get("deduplicate", True)

        self.logger.debug(f"archive-web-search plugin received query: {query}")
        # Support list of queries
        import logging
        logger = logging.getLogger(__name__)
        def parse_if_str(val):
            import ast
            if isinstance(val, str):
                try:
                    parsed = ast.literal_eval(val)
                    if isinstance(parsed, (list, dict)):
                        return parsed
                except Exception:
                    pass
            return val
        
        if isinstance(query, list):
            results = [self.search(q, mediatype=mediatype, max_results=max_results, advanced=advanced, deduplicate=deduplicate) for q in query]
            results = parse_if_str(results)
            if extract_urls:
                urls = []
                for r in results:
                    urls.extend(self.extract_urls_from_response(r, deduplicate=deduplicate))
                urls = parse_if_str(urls)
                logger.info(f"[archive_web_search] Returning type: {type(urls)}, sample: {str(urls)[:300]}")
                return {step_config["output"]: urls}
            else:
                logger.info(f"[archive_web_search] Returning type: {type(results)}, sample: {str(results)[:300]}")
                return {step_config["output"]: results}

        result = self.search(query, mediatype=mediatype, max_results=max_results, advanced=advanced, deduplicate=deduplicate)
        result = parse_if_str(result)
        if extract_urls:
            urls = self.extract_urls_from_response(result, deduplicate=deduplicate)
            urls = parse_if_str(urls)
            logger.info(f"[archive_web_search] Returning type: {type(urls)}, sample: {str(urls)[:300]}")
            return {step_config["output"]: urls}
        else:
            logger.info(f"[archive_web_search] Returning type: {type(result)}, sample: {str(result)[:300]}")
            return {step_config["output"]: result}

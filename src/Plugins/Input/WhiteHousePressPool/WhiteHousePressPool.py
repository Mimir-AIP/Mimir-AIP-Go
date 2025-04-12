"""
Plugin for scraping White House press pool reports
"""

import requests
import json
from datetime import datetime

class WhiteHousePressPoolScraper:
    """
    Plugin for scraping White House press pool reports from forth.news
    """
    plugin_type = "Input"

    def __init__(self):
        """
        Initialize the scraper with the API endpoint and request parameters
        """
        self.url = 'https://www.forth.news/api/graphql'
        self.headers = self._get_headers()
        self.payload = self._get_payload()

    def _get_headers(self):
        """
        Get headers for the API request
        """
        return {
            'accept': '*/*',
            'accept-language': 'en-GB,en-US;q=0.9,en;q=0.8',
            'baggage': 'sentry-environment=vercel-production,sentry-release=4217680f2075e5c6de53a1932b6b2e4b52688e42,sentry-public_key=f7e35dcc7b4a66bf894fbd3bacc728e7,sentry-trace_id=c779d0a50b133380f126707a9e0018c8,sentry-sample_rate=1,sentry-transaction=GET%20%2Flists%2F%5BshortName%5D,sentry-sampled=true',
            'content-type': 'application/json',
            'dnt': '1',
            'origin': 'https://www.forth.news',
            'priority': 'u=1, i',
            'referer': 'https://www.forth.news/whpool?ref=blog.forth.news',
            'sec-ch-ua': '"Not A(Brand";v="8", "Chromium";v="132", "Google Chrome";v="132"',
            'sec-ch-ua-mobile': '?0',
            'sec-ch-ua-platform': '"macOS"',
            'sec-fetch-dest': 'empty',
            'sec-fetch-mode': 'cors',
            'sec-fetch-site': 'same-origin',
            'sentry-trace': 'c779d0a50b133380f126707a9e0018c8-bfc71173ce587e0b-1',
            'user-agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36'
        }

    def _get_payload(self):
        """
        Get GraphQL query payload for the API request
        """
        return {
            "operationName": "getList",
            "variables": {"shortName": "whpool"},
            "query": "query getList($shortName: String!, $last: ID) {\n  list(shortName: $shortName) {\n    id\n    shortName\n    entries(last: $last) {\n      id\n      title\n      pvwText\n      createdAt\n      __typename\n    }\n    __typename\n  }\n}"
        }

    def fetch_data(self):
        """
        Fetch the latest press pool reports

        Returns:
            dict: JSON response containing press pool reports
        """
        response = requests.post(self.url, headers=self.headers, json=self.payload)
        if response.status_code == 200:
            data = response.json()
            return self._parse_data(data)
        else:
            return None

    def _parse_data(self, data):
        """
        Parse the JSON response into a JSON feed format
        """
        json_feed = {
            "version": "https://jsonfeed.org/version/1",
            "title": "Forth News - White House Pool",
            "home_page_url": "https://www.forth.news/whpool",
            "feed_url": "https://www.forth.news/api/graphql",
            "items": []
        }
        for entry in data['data']['list']['entries']:
            json_feed['items'].append({
                "id": entry['id'],
                "url": f"https://www.forth.news/whpool/{entry['id']}",
                "title": entry['title'],
                "content_text": entry['pvwText'],
                "date_published": datetime.fromtimestamp(int(entry['createdAt']) / 1000).isoformat()
            })
        return json_feed

if __name__ == "__main__":
    scraper = WhiteHousePressPoolScraper()
    result = scraper.fetch_data()
    print(json.dumps(result, indent=2))
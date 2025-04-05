"""
RSS, Atom and JSON feed parser

This plugin can be used to fetch and parse RSS, Atom and JSON feeds.

Example usage:
    plugin = FeedPlugin(url)
    feed_data = plugin.get_feed()
    print(feed_data)
"""

import requests
import json
import re

class rss_feed:
    """
    RSS, Atom and JSON feed parser
    """

    def __init__(self, input_data, is_url=True):
        """
        Initialize the plugin

        :param input_data: The URL or the direct feed content
        :param is_url: If True, input_data is a URL, otherwise it is the direct feed content
        """
        self.input_data = input_data
        self.is_url = is_url
        self.feed_type = None
        self.data = None

    def detect_feed_type(self):
        """
        Detect the type of feed
        """
        if self.is_url:
            response = requests.get(self.input_data)
            content = response.text
        else:
            content = self.input_data

        if '<rss' in content:
            self.feed_type = 'rss'
        elif '<feed' in content:
            self.feed_type = 'atom'
        elif re.search(r'^\s*\{', content):
            self.feed_type = 'json'
        else:
            raise ValueError("Unsupported feed type")

    def fetch_feed(self):
        """
        Fetch the feed
        """
        if not self.feed_type:
            self.detect_feed_type()

        if self.is_url:
            response = requests.get(self.input_data)
            content = response.text
        else:
            content = self.input_data

        if self.feed_type == 'rss':
            self.data = self.parse_rss(content)
        elif self.feed_type == 'atom':
            self.data = self.parse_atom(content)
        elif self.feed_type == 'json':
            self.data = self.parse_json(content)
        else:
            raise ValueError("Unsupported feed type")

    def parse_rss(self, content):
        """
        Parse an RSS feed
        """
        try:
            items = re.findall(r'<item>(.*?)</item>', content, re.DOTALL)
            feed = []
            for item in items:
                title = re.search(r'<title>(.*?)</title>', item)
                link = re.search(r'<link>(.*?)</link>', item)
                description = re.search(r'<description>(.*?)</description>', item)
                if title and link:
                    title = title.group(1)
                    link = link.group(1)
                    description = description.group(1) if description else ""
    
                    feed.append({
                        'title': title,
                        'link': link,
                        'description': description
                    })
            return feed
        except Exception as e:
            raise ValueError("Error parsing RSS feed: " + str(e))

    def parse_atom(self, content):
        """
        Parse an Atom feed
        """
        try:
            items = re.findall(r'<entry>(.*?)</entry>', content, re.DOTALL)
            feed = []
            for item in items:
                title = re.search(r'<title>(.*?)</title>', item).group(1)
                link = re.search(r'<link href="(.*?)"', item).group(1)
                summary = re.search(r'<summary>(.*?)</summary>', item) or re.search(r'<content>(.*?)</content>', item)
                if summary:
                    summary = summary.group(1)
                else:
                    summary = ""
                feed.append({
                    'title': title,
                    'link': link,
                    'summary': summary
                })
            return feed
        except Exception as e:
            raise ValueError("Error parsing Atom feed: " + str(e))

    def parse_json(self, content):
        """
        Parse a JSON feed
        """
        try:
            return json.loads(content)
        except Exception as e:
            raise ValueError("Error parsing JSON feed: " + str(e))

    def get_feed(self):
        """
        Get the feed
        """
        self.fetch_feed()
        if self.data is None:
            raise ValueError("Error fetching feed")
        return json.dumps(self.data, indent=2)

# Example usage
if __name__ == "__main__":
    # Example with URL
    url = "http://feeds.bbci.co.uk/news/world/rss.xml"
    plugin = rss_feed(url)
    feed_data = plugin.get_feed()
    print(feed_data)

    # Example with direct feed content
    direct_feed_content = """
    <feed xmlns="http://www.w3.org/2005/Atom">
        <title>Example Feed</title>
        <link href="http://example.org/"/>
        <updated>2023-10-01T12:00:00Z</updated>
        <entry>
            <title>Atom-Powered Robots Run Amok</title>
            <link href="http://example.org/2003/12/13/atom03"/>
            <id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
            <updated>2003-12-13T18:30:02Z</updated>
            <summary>Some text.</summary>
        </entry>
    </feed>
    """
    plugin = rss_feed(direct_feed_content, is_url=False)
    feed_data = plugin.get_feed()
    print(feed_data)


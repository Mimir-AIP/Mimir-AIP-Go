"""
RSS, Atom and JSON feed parser

This plugin can be used to fetch and parse RSS, Atom and JSON feeds.

Example usage:
    plugin = RssFeed()
    feed_data = plugin.execute_pipeline_step({
        "config": {
            "url": "http://example.com/feed.xml",
            "feed_name": "Example"
        },
        "output": "feed_data"
    }, {})
    print(feed_data)
"""

import requests
import json
import re
import logging
from Plugins.BasePlugin import BasePlugin


class RssFeed(BasePlugin):
    """
    RSS, Atom and JSON feed parser
    """

    plugin_type = "Input"

    def __init__(self):
        """Initialize the plugin"""
        self.input_data = None
        self.is_url = True
        self.feed_type = None
        self.data = None

    def execute_pipeline_step(self, step_config, context):
        """Execute a pipeline step for this plugin
        
        Expected step_config format:
        {
            "plugin": "RSS-Feed",
            "config": {
                "url": "http://example.com/feed.xml",
                "feed_name": "Example Feed"
            },
            "output": "feed_data"  # Optional, defaults to feed_{feed_name}
        }
        """
        logger = logging.getLogger(__name__)
        logger.debug(f"Executing pipeline step with config: {step_config}")
        config = step_config["config"]
        logger.debug(f"Setting input data: {config['url']}")
        self.set_input(config["url"])
        logger.debug(f"Fetching feed...")
        try:
            feed_data = self.fetch_feed()
            # Defensive patch: ensure output is always a native Python object, never a string
            import ast
            def parse_if_str(val):
                if isinstance(val, str):
                    try:
                        parsed = ast.literal_eval(val)
                        if isinstance(parsed, (list, dict)):
                            return parsed
                    except Exception:
                        pass
                return val
            feed_data = parse_if_str(feed_data)
            logger.info(f"[RSSFeed:execute_pipeline_step] Returning type: {type(feed_data)}, sample: {str(feed_data)[:300]}")
            output_key = step_config.get("output") or f"feed_{config.get('feed_name', 'data')}"
            return {output_key: feed_data}
        except Exception as e:
            logger.error(f"Error fetching feed: {e}")
            return {}

    def set_input(self, input_data, is_url=True):
        """
        Set the input data after initialization

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
        logger = logging.getLogger(__name__)
        if self.input_data is None:
            logger.error("Input data must be set before detecting feed type")
            raise ValueError("Input data must be set before detecting feed type")

        if self.is_url:
            response = requests.get(self.input_data)
            logger.debug(f"detect_feed_type: HTTP status {response.status_code}")
            logger.debug(f"detect_feed_type: Response content (truncated): {response.text[:500]}")
            content = response.text
            # Save the raw content to a file for inspection
            try:
                with open("rss_feed_raw.xml", "w", encoding="utf-8") as f:
                    f.write(content)
                logger.info("Raw RSS feed content saved to rss_feed_raw.xml")
            except Exception as e:
                logger.error(f"Failed to save raw RSS feed content: {e}")
        else:
            content = self.input_data

        if '<rss' in content:
            self.feed_type = 'rss'
        elif '<feed' in content:
            self.feed_type = 'atom'
        elif re.search(r'^\s*\{', content):
            self.feed_type = 'json'
        else:
            logger.error("detect_feed_type: Unsupported feed type. Content snippet: " + content[:500])
            raise ValueError("Unsupported feed type")

    def fetch_feed(self):
        """
        Fetch the feed
        """
        logger = logging.getLogger(__name__)
        try:
            if self.input_data is None:
                logger.error("fetch_feed: Input data must be set before fetching feed")
                raise ValueError("Input data must be set before fetching feed")

            if not self.feed_type:
                logger.debug("fetch_feed: Detecting feed type...")
                self.detect_feed_type()
            else:
                logger.debug(f"fetch_feed: Using existing feed_type {self.feed_type}")

            if self.is_url:
                response = requests.get(self.input_data)
                logger.debug(f"fetch_feed: HTTP status {response.status_code}")
                logger.debug(f"fetch_feed: Response content (truncated): {response.text[:500]}")
                response.raise_for_status()  # Raise an exception for HTTP errors
                content = response.text
                # Save the raw content to a file for inspection
                try:
                    with open("rss_feed_raw.xml", "w", encoding="utf-8") as f:
                        f.write(content)
                    logger.info("Raw RSS feed content saved to rss_feed_raw.xml")
                except Exception as e:
                    logger.error(f"Failed to save raw RSS feed content: {e}")
            else:
                content = self.input_data

            logger.debug(f"fetch_feed: Parsing as {self.feed_type}")
            if self.feed_type == 'rss':
                self.data = self.parse_rss(content)
            elif self.feed_type == 'atom':
                self.data = self.parse_atom(content)
            elif self.feed_type == 'json':
                self.data = self.parse_json(content)
            else:
                logger.error("fetch_feed: Unsupported feed type after detection")
                raise ValueError("Unsupported feed type")

            logger.info(f"[RSSFeed:fetch_feed] Returning type: {type(self.data)}, sample: {str(self.data)[:300]}")
            logger.debug(f"Successfully fetched feed with {len(self.data)} items")
            return self.data

        except requests.exceptions.HTTPError as e:
            logger.error(f"HTTP error occurred while fetching feed: {e}")
            raise
        except requests.exceptions.RequestException as e:
            logger.error(f"Request error occurred while fetching feed: {e}")
            raise
        except Exception as e:
            logger.error(f"Error fetching feed: {e}", exc_info=True)
            raise

    def parse_rss(self, content):
        """
        Parse an RSS feed
        """
        def strip_cdata(text):
            if text is None:
                return ""
            # Remove CDATA if present
            if text.startswith('<![CDATA[') and text.endswith(']]>'):
                return text[9:-3]
            return text

        try:
            items = re.findall(r'<item>(.*?)</item>', content, re.DOTALL)
            feed = []
            for idx, item in enumerate(items):
                title_match = re.search(r'<title>(.*?)</title>', item, re.DOTALL)
                link_match = re.search(r'<link>(.*?)</link>', item, re.DOTALL)
                description_match = re.search(r'<description>(.*?)</description>', item, re.DOTALL)
                if title_match and link_match:
                    title = strip_cdata(title_match.group(1).strip())
                    link = link_match.group(1).strip()
                    description = strip_cdata(description_match.group(1).strip()) if description_match else ""
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

# Example usage
if __name__ == "__main__":
    # Example with URL
    url = "http://feeds.bbci.co.uk/news/world/rss.xml"
    plugin = RssFeed()
    feed_data = plugin.execute_pipeline_step({
        "config": {
            "url": url,
            "feed_name": "BBC News"
        },
        "output": "feed_data"
    }, {})
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
    plugin = RssFeed()
    feed_data = plugin.execute_pipeline_step({
        "config": {
            "url": direct_feed_content,
            "feed_name": "Example Feed"
        },
        "output": "feed_data"
    }, {})
    print(feed_data)

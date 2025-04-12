import re
import requests
import html

class WebScraping:
    """
    Plugin for web scraping with configurable user agent, timeout, and proxy
    """

    plugin_type = "Input"

    def __init__(self, user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3", timeout=10, proxy=None):
        self.user_agent = user_agent
        self.timeout = timeout
        self.proxy = proxy

    def _clean_text(self, text):
        """Helper method to clean and normalize text content"""
        # First decode HTML entities
        text = html.unescape(text)
        # Replace newlines, tabs, etc. with spaces
        text = re.sub(r'[\n\r\t]', ' ', text)
        # Replace multiple spaces with a single space
        text = re.sub(r'\s+', ' ', text)
        # Remove special control characters
        text = re.sub(r'-->', '', text)
        # Strip leading/trailing whitespace
        return text.strip()

    def scrape(self, url, css_selector=None, scrape_type="text"):
        headers = {'User-Agent': self.user_agent}
        proxies = {'http': self.proxy, 'https': self.proxy} if self.proxy else None

        # Request the webpage
        response = requests.get(url, headers=headers, timeout=self.timeout, proxies=proxies)
        response.raise_for_status()  # Raise an exception for HTTP errors
        html_content = response.text

        if scrape_type == "text":
            # Remove all HTML tags to get the raw text content
            text = re.sub(r"<[^>]*>", "", html_content)
            return self._clean_text(text)
            
        elif scrape_type == "title":
            # Extract the content of the <title> tag
            match = re.search(r"<title>(.*?)</title>", html_content, re.IGNORECASE | re.DOTALL)
            if match:
                return self._clean_text(match.group(1))
            return None
            
        elif css_selector:
            # Support basic CSS selectors (only tag-based, no attributes/classes)
            # Find the matching tag's content
            tag_pattern = f"<{css_selector}[^>]*>(.*?)</{css_selector}>"
            tag_matches = re.findall(tag_pattern, html_content, re.IGNORECASE | re.DOTALL)

            # Clean the inner HTML (remove nested tags) for each match
            cleaned_matches = []
            for match in tag_matches:
                # Remove HTML tags
                clean_text = re.sub(r"<[^>]*>", "", match)
                # Clean up the text
                clean_text = self._clean_text(clean_text)
                # Filter out empty or whitespace-only strings
                if clean_text and not clean_text.isspace():
                    cleaned_matches.append(clean_text)
            
            return cleaned_matches
        else:
            return None

if __name__ == "__main__":
    # Example usage
    print("WebScraping Plugin")
    url = "https://books.toscrape.com/catalogue/the-project_856/index.html"
    plugin = WebScraping()

    # Scrape the page title
    title = plugin.scrape(url, scrape_type="title")
    print("Scraped title:", title)

    # Scrape all text of elements matching <h1> tags
    h1_text = plugin.scrape(url, css_selector="h1")
    print("Scraped text for h1 elements:", h1_text)

    # Scrape all text of elements matching <p> tags
    p_text = plugin.scrape(url, css_selector="p")
    print("Scraped text for p elements:", p_text)

    # Scrape all text
    all_text = plugin.scrape(url)
    print("Scraped all text:", all_text)
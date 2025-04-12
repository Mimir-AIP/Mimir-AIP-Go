import json
import re
import requests

class GazaMapsPlugin:
    plugin_type = "Input"

    def __init__(self):
        self.base_url = "https://gazamaps.com"

    def scrape_gaza_maps(self):
        """
        Scrapes Gaza Maps and returns a list of items.
        """
        items = []
        page_number = 1

        while True:
            response = requests.get(f"{self.base_url}?page={page_number}")
            if response.status_code != 200:
                print("Failed to fetch the website content.")
                break

            html_content = response.text

            # Regex to find the date, image source, and alt text
            regex = re.compile(
                r"<h2>(.*?)<\/h2>\s*<a href=\"(.*?)\">\s*<img.*?src=\"(.*?)\".*?alt=\"(.*?)\".*?>\s*<\/a>",
                re.DOTALL
            )

            matches = regex.findall(html_content)

            if not matches:
                print("No matches found using the regex.")
                break

            for match in matches:
                date = match[0].strip()
                displacement_url = match[1]
                image_url = self.base_url + match[2]  # Correctly construct the absolute URL
                alt_text = match[3]
                link_url = displacement_url  # No need to add the base URL again since it is already absolute

                item = {
                    "id": link_url,
                    "url": link_url,
                    "title": alt_text,
                    "content_text": alt_text,
                    "image": image_url,
                    "date_published": date,
                }
                items.append(item)

            # Check for the next page
            next_page_regex = re.compile(r'<a.*?href="([^"]*?page=(\d+))".*?>Next</a>', re.IGNORECASE)
            next_page_match = next_page_regex.search(html_content)
            if not next_page_match:
                break

            page_number += 1

        return items

    def get_gaza_maps_data(self):
        """
        Returns a JSON feed compatible with RSSGuard.
        """
        items = self.scrape_gaza_maps()
        return {"items": items}

if __name__ == "__main__":
    # Test the plugin
    plugin = GazaMapsPlugin()
    data = plugin.get_gaza_maps_data()
    if data and data['items']:  # Check also if the list contains something
        print(json.dumps(data, indent=2, ensure_ascii=False))
    else:
        print("No items found or an error occurred.")
        if data is not None:
            print("The regex likely failed to match any content.")
import json
import re
import requests

BASE_URL = "https://gazamaps.com"

def scrape_gaza_maps():
    items = []
    page_number = 1

    while True:
        response = requests.get(f"{BASE_URL}?page={page_number}")
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
            image_url = BASE_URL + match[2]  # Correctly construct the absolute URL
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

    # Generate JSON feed compatible with RSSGuard
    feed = {
        "version": "https://jsonfeed.org/version/1",
        "title": "Gaza Maps Feed",
        "home_page_url": BASE_URL,
        "feed_url": BASE_URL,
        "items": items,
    }

    return feed

if __name__ == "__main__":
    result = scrape_gaza_maps()
    if result and result['items']:  # Check also if the list contains something
        print(json.dumps(result, indent=2, ensure_ascii=False))
    else:
        print("No items found or an error occurred.")
        if result is not None:
            print("The regex likely failed to match any content.")
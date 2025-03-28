import requests
import json
import datetime

def bloomberg_to_rssguard(api_url):
    """
    Fetches data from the Bloomberg API and converts it to an RSSGuard-compatible JSON format.
    """
    try:
        response = requests.get(api_url)
        response.raise_for_status()  # Raise HTTPError for bad responses (4xx or 5xx)
        data = response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error fetching data from Bloomberg API: {e}")
        return None

    rssguard_data = {
        "version": 1,
        "title": "Bloomberg News Feed",  # Customize title as needed
        "link": "https://www.bloomberg.com/", # Customize link as needed
        "description": "Bloomberg News", # Customize description as needed
        "items": []
    }

    for item in data.get("items", []):
        rss_item = {
            "title": item.get("title", "No Title"),
            "link": item.get("link", ""),
            "guid": item.get("id", ""), # Use Bloomberg's ID as GUID
            "description": "", # You can add a snippet here if available
            "pubDate": datetime.datetime.now().isoformat(), # Set current date. Bloomberg API doesn't provide the original publish date.
            "author": "Bloomberg" # Set the author as Bloomberg
        }
        rssguard_data["items"].append(rss_item)

    return rssguard_data

if __name__ == "__main__":
    api_url = "https://feeds.bloomberg.com/news.json?ageHours=120&token=glassdoor%3Agd4bloomberg&tickers=NTRS%3AUS"  # Replace with your actual API URL
    rssguard_feed = bloomberg_to_rssguard(api_url)

    if rssguard_feed:
        print(json.dumps(rssguard_feed, indent=4)) # Output JSON to console
        # Optionally, save to a file:
        # with open("bloomberg_feed.json", "w") as f:
        #     json.dump(rssguard_feed, f, indent=4)

"""
Performs a search using the DuckDuckGo API
"""

import requests
import json


def search(query):
    """
    Perform a search using the DuckDuckGo API

    Args:
        query (str): The search query

    Returns:
        dict: A dictionary containing the search results
    """
    # The URL of the DuckDuckGo API
    url = "https://api.duckduckgo.com/"

    # The parameters to pass to the API
    # The "q" parameter specifies the search query
    # The "format" parameter specifies the format of the response (in this case, JSON)
    params = {
        "q": query,
        "format": "json"
    }

    # Make a GET request to the API with the specified parameters
    try:
        response = requests.get(url, params=params)
        # Raise an exception if the request fails
        response.raise_for_status()
    except requests.exceptions.RequestException as e:
        # If the request fails, return an error message
        return {"error": f"Search request failed: {e}"}

    # Parse the JSON response from the API
    data = response.json()

    # Create a list to store the search results
    results = []

    # Iterate over the search results
    for topic in data.get("RelatedTopics", []):
        # Extract the title, link, and snippet from the search result
        result = {
            "title": topic.get("Text"),
            "link": topic.get("FirstURL"),
            "snippet": topic.get("Text")
        }
        # Append the search result to the list of results
        results.append(result)

    # Return the list of search results
    return {
        "results": results
    }


if __name__ == "__main__":
    # Perform a search for the string "Python programming"
    query = "Python programming"
    results = search(query)
    # Print the search results in a human-readable format
    print(json.dumps(results, indent=2))


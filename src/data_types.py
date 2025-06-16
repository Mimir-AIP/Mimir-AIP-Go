"""
data_types module.

Defines standardized dictionary structures for various data types
used within the pipeline context, especially for binary data and references.
These structures are designed to be compatible with JSON Schema validation.
"""

from typing import Dict, Any, Literal, Union, Optional

# Define a type alias for primitive data types
PrimitiveType = Union[int, float, str, bool, None]

# Define the structure for binary data
BinaryDataType = Dict[str, Union[Literal["binary"], str, int, float, Dict[str, Any]]]

def create_binary_data(
    format: str,
    inline_data: Optional[str] = None,
    storage_reference: Optional[Dict[str, str]] = None, 
    inline_encoding: Optional[Literal["base64"]] = None,
    duration: Optional[float] = None,
    sample_rate: Optional[int] = None,
    channels: Optional[int] = None,
    dimensions: Optional[Dict[str, int]] = None,
    pages: Optional[int] = None
) -> BinaryDataType:
    """
    Creates a standardized dictionary representation for binary data.
    The data can be provided inline (Base64 encoded) or as a reference
    to an object in a storage backend.

    Args:
        format: The media type of the binary data (e.g., "image/jpeg", "video/mp4").
        inline_data: Optional. The binary data, Base64 encoded, if stored inline.
        storage_reference: Optional. A dictionary with 'namespace' and 'key' if data is referenced from storage.
        inline_encoding: Optional. The encoding for inline_data (e.g., "base64"). Required if inline_data is provided.
        duration: Optional duration in seconds (for audio/video).
        sample_rate: Optional sample rate in Hz (for audio).
        channels: Optional number of audio channels.
        dimensions: Optional dictionary with width/height (for images/videos).
        pages: Optional number of pages (for documents like PDFs).

    Returns:
        BinaryDataType: A dictionary representing the binary data.

    Raises:
        ValueError: If input parameters are inconsistent.
    """
    if not (inline_data is None) == (storage_reference is None):
        # XOR: one must be provided, but not both
        pass
    else:
        raise ValueError("Either 'inline_data' or 'storage_reference' must be provided, but not both.")

    base_binary_data: Dict[str, Any] = {
        "__type__": "binary",
        "format": format,
    }

    if inline_data is not None:
        if inline_encoding is None:
            raise ValueError("'inline_encoding' must be specified when 'inline_data' is provided.")
        if inline_encoding != "base64":
            raise ValueError("Only 'base64' encoding is currently supported for inline binary data.")
        base_binary_data["data_location"] = "inline"
        base_binary_data["encoding"] = inline_encoding
        base_binary_data["data"] = inline_data
    elif storage_reference is not None:
        if not isinstance(storage_reference, dict) or "namespace" not in storage_reference or "key" not in storage_reference:
            raise ValueError("'storage_reference' must be a dict with 'namespace' and 'key'.")
        base_binary_data["data_location"] = "referenced"
        base_binary_data["storage_reference"] = storage_reference

    # Add format-specific metadata
    if duration is not None:
        base_binary_data["duration"] = duration
    if sample_rate is not None:
        base_binary_data["sample_rate"] = sample_rate
    if channels is not None:
        base_binary_data["channels"] = channels
    if dimensions is not None:
        base_binary_data["dimensions"] = dimensions
    if pages is not None:
        base_binary_data["pages"] = pages

    return base_binary_data

# Define the structure for reference data
ReferenceType = Dict[str, Union[Literal["reference"], Literal["file_path", "url"], str]]

def create_reference_data(value: str, kind: Literal["file_path", "url"]) -> ReferenceType:
    """
    Creates a standardized dictionary representation for reference data.

    Args:
        value: The actual reference string (e.g., file path, URL).
        kind: The type of reference ("file_path" or "url").

    Returns:
        ReferenceType: A dictionary representing the reference data.
    """
    if kind not in ["file_path", "url"]:
        raise ValueError("Reference kind must be 'file_path' or 'url'.")
    return {
        "__type__": "reference",
        "kind": kind,
        "value": value
    }

# Example usage (for documentation/testing purposes)
if __name__ == "__main__":
    # Example 1: Inline binary data (e.g., small image)
    inline_image_data = create_binary_data(
        format="image/png",
        inline_data="aGVsbG8gd29ybGQ=", 
        inline_encoding="base64",
        dimensions={"width": 100, "height": 50}
    )
    print(f"Inline Binary Image Data: {inline_image_data}")

    # Example 2: Referenced binary data (e.g., large video file stored locally)
    referenced_video_data = create_binary_data(
        format="video/mp4",
        storage_reference={"namespace": "project_alpha", "key": "trailers/main_trailer.mp4"},
        duration=120.5,
        dimensions={"width": 1920, "height": 1080}
    )
    print(f"Referenced Binary Video Data: {referenced_video_data}")

    # Example 3: Inline audio data
    inline_audio_data = create_binary_data(
        format="audio/wav",
        inline_data="c29tZSBhdWRpbyBkYXRh", 
        inline_encoding="base64",
        duration=15.0,
        sample_rate=44100,
        channels=2
    )
    print(f"Inline Binary Audio Data: {inline_audio_data}")

    # Example 4: Referenced PDF document
    referenced_pdf_data = create_binary_data(
        format="application/pdf",
        storage_reference={"namespace": "shared_docs", "key": "reports/annual_report_2024.pdf"},
        pages=78
    )
    print(f"Referenced PDF Data: {referenced_pdf_data}")

    # Example reference data (unchanged from before, for context)
    file_ref = create_reference_data("/app/data/input.txt", "file_path")
    print(f"File Reference Data (legacy): {file_ref}")

    url_ref = create_reference_data("https://example.com/api/data", "url")
    print(f"URL Reference Data (legacy): {url_ref}")

    # Validation would now need to consider 'data_location' and conditionally
    # require 'data'/'encoding' or 'storage_reference'.
    # Example schema snippet for referenced data:
    # {
    #   "type": "object",
    #   "properties": {
    #     "__type__": { "type": "string", "const": "binary" },
    #     "format": { "type": "string" },
    #     "data_location": { "type": "string", "enum": ["inline", "referenced"] },
    #     // ... other common fields like duration, dimensions etc.
    #   },
    #   "required": ["__type__", "format", "data_location"],
    #   "if": {
    #     "properties": { "data_location": { "const": "inline" } }
    #   },
    #   "then": {
    #     "properties": {
    #       "encoding": { "type": "string", "const": "base64" },
    #       "data": { "type": "string" }
    #     },
    #     "required": ["encoding", "data"]
    #   },
    #   "else": { # data_location is "referenced"
    #     "properties": {
    #       "storage_reference": {
    #         "type": "object",
    #         "properties": {
    #           "namespace": { "type": "string" },
    #           "key": { "type": "string" }
    #         },
    #         "required": ["namespace", "key"]
    #       }
    #     },
    #     "required": ["storage_reference"]
    #   }
    # }
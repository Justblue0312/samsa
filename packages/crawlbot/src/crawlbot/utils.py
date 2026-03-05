from urllib.parse import urljoin, urlsplit


def get_site_name(url: str) -> str:
    """Get the site name from a URL."""
    parsed_url = urlsplit(url)
    return parsed_url.netloc


def build_url(base: str, *parts: str) -> str:
    """Build a URL from parts."""
    # We avoid quote_plus here to prevent encoding characters like '?' or '=' in the parts.
    # Instead, we just join the parts simply and let urljoin handle the rest.
    path = "/".join(part.strip("/") for part in parts)
    return urljoin(base if base.endswith("/") else base + "/", path)


def slugify(text: str) -> str:
    """Convert text to a slug."""
    import re
    import unicodedata

    # Convert to lowercase
    text = text.lower()
    # Normalize unicode characters to remove accents
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode("ascii")
    # Remove non-alphanumeric characters (except hyphens and spaces)
    text = re.sub(r"[^a-z0-9\s-]", "", text)
    # Replace spaces and multiple hyphens with a single hyphen
    text = re.sub(r"[\s-]+", "-", text).strip("-")
    return text


def strip_html_tags(html: str) -> str:
    """Strip HTML tags from a string."""
    from bs4 import BeautifulSoup

    soup = BeautifulSoup(html, "html.parser")
    return soup.get_text()

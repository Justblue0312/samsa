import asyncio
import json
import re
from typing import Any, Self

import aiofiles
import aiohttp
from bs4 import BeautifulSoup

from crawlbot.base import BaseStory
from crawlbot.settings import DATA_DIR
from crawlbot.utils import build_url, get_site_name, slugify


class RoyalRoad(BaseStory):
    _list_stories_route = "/fictions/best-rated?page={}"
    _story_route = "/fiction/{}"

    def __init__(self):
        self.url = "https://www.royalroad.com"
        self.headers: dict[str, Any] = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
        }

    def set_headers(self, headers) -> Self:
        self.headers.update(headers)
        return self

    def get_site_name(self) -> str:
        return get_site_name(self.url)

    async def get_story_urls(self, session: aiohttp.ClientSession, page: int = 1) -> list[str]:
        """Get a list of story URLs from the best-rated page."""
        if page < 1:
            page = 1
        listing_path = build_url(self.url, self._list_stories_route.format(page))
        try:
            async with session.get(listing_path) as response:
                if response.status == 200:
                    text = await response.text()
                    return self._extract_stories(text)
        except Exception as e:
            print(f"Error fetching listing page {page}: {e}")
        return []

    def _extract_stories(self, response: str) -> list[str]:
        """Extract stories from the listing page."""
        soup = BeautifulSoup(response, "html.parser")
        # Links are in h2.fiction-title a
        links = soup.select("h2.fiction-title a, .fiction-title a")
        urls = []
        for link in links:
            href = link.get("href")
            if href and "/fiction/" in href:
                urls.append(href)
        return list(dict.fromkeys(urls))

    def _extract_story_details(self, response: str) -> dict[str, Any]:
        """Extract story details and chapter links."""
        soup = BeautifulSoup(response, "html.parser")

        details = {"title": "Unknown", "description": "", "chapter_links": []}

        # Title: h1
        title_tag = soup.select_one("h1")
        if title_tag:
            details["title"] = title_tag.get_text(strip=True)

        # Description: .description
        desc_tag = soup.select_one(".description")
        if desc_tag:
            details["description"] = desc_tag.get_text("\n", strip=True)

        # Chapters: table#chapters tbody tr td:first-child a
        chapter_tags = soup.select("table#chapters tbody tr td:first-child a[href*='/chapter/']")
        for tag in chapter_tags:
            href = tag.get("href")
            if href:
                details["chapter_links"].append(href)

        return details

    def _extract_chapter_data(self, response: str) -> dict[str, Any]:
        """Extract chapter title and content."""
        soup = BeautifulSoup(response, "html.parser")

        data = {"title": "Unknown Chapter", "content": ""}

        # Title: h1
        title_tag = soup.select_one("h1")
        if title_tag:
            data["title"] = title_tag.get_text(strip=True)

        # Content: .chapter-inner.chapter-content
        content_tag = soup.select_one(".chapter-inner.chapter-content")
        if content_tag:
            # Royal Road often has "Author Note" or "End Note" inside.
            # We'll just take the whole thing for now, or clean if needed.
            data["content"] = content_tag.get_text("\n", strip=True)

        return data

    async def fetch_page(self, session: aiohttp.ClientSession, url: str) -> str:
        """Generic fetcher."""
        try:
            async with session.get(url, timeout=15) as response:  # type: ignore
                if response.status == 200:
                    return await response.text()
                else:
                    print(f"Failed to fetch {url}: Status {response.status}")
        except Exception as e:
            print(f"Error fetching {url}: {e}")
        return ""

    async def crawl_story_and_chapters(self, session: aiohttp.ClientSession, story_url: str, max_chapters: int = 1000):
        """Crawl story and a limited number of chapters."""
        full_story_url = build_url(self.url, story_url) if story_url.startswith("/") else story_url

        # Extract ID
        id_match = re.search(r"/fiction/(\d+)", story_url)
        story_id = id_match.group(1) if id_match else "unknown"

        print(f"--- Processing RoyalRoad Story: {story_id} ---")

        story_html = await self.fetch_page(session, full_story_url)
        if not story_html:
            return

        details = self._extract_story_details(story_html)
        story_slug = slugify(details["title"])
        if not story_slug or story_slug == "unknown":
            story_slug = f"story_{story_id}"

        story_dir = DATA_DIR / self.get_site_name() / story_slug
        story_dir.mkdir(parents=True, exist_ok=True)

        # Save details
        details_path = story_dir / "story_details.json"
        async with aiofiles.open(details_path, mode="w", encoding="utf-8") as f:
            await f.write(
                json.dumps(
                    {
                        "title": details["title"],
                        "slug": story_slug,
                        "description": details["description"],
                        "url": full_story_url,
                        "story_id": story_id,
                        "chapter_count": len(details["chapter_links"]),
                    },
                    ensure_ascii=False,
                    indent=4,
                )
            )

        # Crawl Chapters
        links_to_crawl = details["chapter_links"][:max_chapters]
        print(f"  - Crawling {len(links_to_crawl)} chapters...")

        for idx, ch_url in enumerate(links_to_crawl):
            full_ch_url = build_url(self.url, ch_url) if ch_url.startswith("/") else ch_url
            ch_num = idx + 1
            print(f"    - Chapter {ch_num}: {full_ch_url}")

            ch_html = await self.fetch_page(session, full_ch_url)
            if not ch_html:
                continue

            ch_data = self._extract_chapter_data(ch_html)
            ch_file = story_dir / f"chapter_{ch_num}.json"
            async with aiofiles.open(ch_file, mode="w", encoding="utf-8") as f:
                await f.write(
                    json.dumps(
                        {
                            "title": ch_data["title"],
                            "chapter_number": ch_num,
                            "content": ch_data["content"],
                            "url": full_ch_url,
                        },
                        ensure_ascii=False,
                        indent=4,
                    )
                )

            await asyncio.sleep(0.5)

    async def run(self, page_range: range, batch_size: int = 1):
        async with aiohttp.ClientSession(headers=self.headers) as session:
            print(f"Fetching RR stories from pages {list(page_range)}...")
            for page in page_range:
                story_urls = await self.get_story_urls(session, page)
                print(f"Found {len(story_urls)} stories on page {page}.")

                for story_url in story_urls:
                    await self.crawl_story_and_chapters(session, story_url)

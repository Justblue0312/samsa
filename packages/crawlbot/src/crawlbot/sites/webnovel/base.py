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


class WebNovel(BaseStory):
    _list_stories_route = "/stories/novel?pageIndex={}"
    _story_route = "/book/{}"
    _chapter_route = "/book/{}/{}"

    def __init__(self):
        self.url = "https://www.webnovel.com"
        self.headers: dict[str, Any] = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
        }

    def set_headers(self, headers) -> Self:
        self.headers.update(headers)
        return self

    def get_site_name(self) -> str:
        return get_site_name(self.url)

    async def get_story_urls(self, session: aiohttp.ClientSession, page: int = 1) -> list[str]:
        """Get a list of story URLs from the given page index."""
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
        """Extract stories from the response text using JSON-LD or HTML selectors."""
        story_urls = []

        # Method 1: JSON-LD (reliable for list)
        soup = BeautifulSoup(response, "html.parser")
        json_ld_scripts = soup.find_all("script", type="application/ld+json")
        for script in json_ld_scripts:
            try:
                data = json.loads(script.get_text())
                if isinstance(data, dict) and data.get("@type") == "ItemList":
                    for item in data.get("itemListElement", []):
                        if item.get("url"):
                            story_urls.append(item["url"])
                elif isinstance(data, dict) and data.get("@type") == "Book":
                    if data.get("url"):
                        story_urls.append(data["url"])
            except Exception:
                continue

        # Method 2: HTML selectors (fallback)
        if not story_urls:
            stories = soup.select(".book-img-text li h3 a, .j_bookList li h3 a, a[href*='/book/']")
            for story in stories:
                href = story.get("href")
                if href and "/book/" in href and "_" in href:
                    story_urls.append(href)

        return list(dict.fromkeys(story_urls))

    def _extract_story_details(self, response: str, story_url: str) -> dict[str, Any]:
        """Extract book details from the story page HTML."""
        details = {"title": "Unknown", "description": "", "bookId": "", "firstChapterId": "", "totalChapterNum": 0}

        # Extract bookId from URL if possible
        # URL format: https://www.webnovel.com/book/shadow-slave_22196546206090805
        url_match = re.search(r"_(\d+)$", story_url.split("?")[0].rstrip("/"))
        if url_match:
            details["bookId"] = url_match.group(1)

        # Try to find bookId and firstChapterId in JSON-like strings in the HTML
        if not details["bookId"]:
            id_match = re.search(r'["\']bookId["\']\s*:\s*["\'](\d+)["\']', response)
            if id_match:
                details["bookId"] = id_match.group(1)

        first_chap_match = re.search(r'["\']firstChapterId["\']\s*:\s*["\'](\d+)["\']', response)
        if first_chap_match:
            details["firstChapterId"] = first_chap_match.group(1)

        # Fallback to Next.js props if mobile
        if not details["bookId"] or not details["firstChapterId"]:
            next_data_match = re.search(r'<script id="__NEXT_DATA__".*?>(.*?)</script>', response, re.DOTALL)
            if next_data_match:
                try:
                    data = json.loads(next_data_match.group(1))
                    book_info = data.get("props", {}).get("pageProps", {}).get("bookDetail", {}).get("bookInfo", {})
                    if book_info:
                        if not details["bookId"]:
                            details["bookId"] = book_info.get("bookId", "")
                        if not details["firstChapterId"]:
                            details["firstChapterId"] = book_info.get("firstChapterId", "")
                        details["title"] = book_info.get("bookName", "")
                        details["description"] = book_info.get("description", "")
                except Exception:
                    pass

        # Fallback to general HTML
        soup = BeautifulSoup(response, "html.parser")
        if details["title"] == "Unknown":
            title_tag = soup.select_one(".det-hd h1, h1.font-bold, title")
            if title_tag:
                details["title"] = title_tag.get_text(strip=True).replace(" - WebNovel", "")

        if not details["description"]:
            desc_tag = soup.select_one(".j_synopsis, meta[name='description']")
            if desc_tag:
                if desc_tag.name == "meta":
                    details["description"] = desc_tag.get("content", "")
                else:
                    details["description"] = desc_tag.get_text("\n", strip=True)

        return details

    def _extract_chapter_data(self, response: str) -> dict[str, Any]:
        """Extract chapter title, content, and nextChapterId from chapter page."""
        data = {"title": "", "content": "", "nextChapterId": ""}

        # 1. Extract nextChapterId
        next_match = re.search(r'["\']nextcId["\']\s*:\s*["\'](\d*)["\']', response)
        if not next_match:
            next_match = re.search(r'["\']nextChapterId["\']\s*:\s*["\'](\d*)["\']', response)

        if next_match:
            data["nextChapterId"] = next_match.group(1)

        # 2. Extract content and title from chapInfo JSON or similar
        chap_info_match = re.search(r"var chapInfo\s*=\s*(\{.*?\});", response, re.DOTALL)
        if not chap_info_match:
            chap_info_match = re.search(r'["\']chapInfo["\']\s*:\s*(\{.*?\})', response, re.DOTALL)

        if chap_info_match:
            try:
                chap_json = json.loads(chap_info_match.group(1))
                chap_info = chap_json.get("chapterInfo", {})
                data["title"] = chap_info.get("chapterName", "")

                items = chap_info.get("chapterItems", [])
                paragraphs = []
                for item in items:
                    p_content = item.get("content", "")
                    # Clean <p> tags and other HTML-like escapings
                    p_content = re.sub(r"\\?/?p>", "", p_content)
                    p_content = p_content.replace('\\"', '"').replace("\\'", "'")
                    paragraphs.append(p_content)

                data["content"] = "\n\n".join(paragraphs)
            except Exception:
                pass

        # 3. Fallback to HTML selectors
        if not data["content"]:
            soup = BeautifulSoup(response, "html.parser")
            title_tag = soup.select_one(".cha-tit h3, ._cha-tit h3, .cha-hd-mn h1")
            if title_tag:
                data["title"] = title_tag.get_text(strip=True)

            content_tags = soup.select(".cha-paragraph p, .cha-content p, ._cha-content p")
            if content_tags:
                data["content"] = "\n\n".join([p.get_text(strip=True) for p in content_tags])
            else:
                content_div = soup.select_one(".cha-content, ._cha-content")
                if content_div:
                    data["content"] = content_div.get_text("\n", strip=True)

        return data

    async def fetch_page(self, session: aiohttp.ClientSession, url: str) -> str:
        """Generic fetcher for HTML content."""
        try:
            async with session.get(url, timeout=15) as response:  # type: ignore
                if response.status == 200:
                    return await response.text()
                else:
                    print(f"Failed to fetch {url}: Status {response.status}")
        except Exception as e:
            print(f"Error fetching {url}: {e}")
        return ""

    async def crawl_story_and_chapters(
        self, session: aiohttp.ClientSession, story_url: str, max_chapters_limit: int = 1000
    ):
        """Crawl a single story and follow next chapter links."""
        full_story_url = build_url(self.url, story_url) if story_url.startswith("/") else story_url

        print(f"--- Processing Story: {full_story_url} ---")

        story_html = await self.fetch_page(session, full_story_url)
        if not story_html:
            return

        story_details = self._extract_story_details(story_html, full_story_url)
        book_id = story_details["bookId"]
        first_chap_id = story_details["firstChapterId"]

        if not book_id or not first_chap_id:
            print(f"Could not find bookId or firstChapterId for {story_url}")
            # Try mobile version if desktop fails
            mobile_url = full_story_url.replace("www.", "m.")
            print(f"Trying mobile version: {mobile_url}")
            story_html = await self.fetch_page(session, mobile_url)
            if story_html:
                story_details = self._extract_story_details(story_html, mobile_url)
                book_id = story_details["bookId"]
                first_chap_id = story_details["firstChapterId"]

        if not book_id or not first_chap_id:
            print(f"FAILED to find IDs for {story_url}")
            return

        story_slug = slugify(story_details["title"])
        if not story_slug or story_slug == "unknown":
            story_slug = f"story_{book_id}"

        story_dir = DATA_DIR / self.get_site_name() / story_slug
        story_dir.mkdir(parents=True, exist_ok=True)

        # Save story details
        details_path = story_dir / "story_details.json"
        async with aiofiles.open(details_path, mode="w", encoding="utf-8") as f:
            await f.write(
                json.dumps(
                    {
                        "title": story_details["title"],
                        "slug": story_slug,
                        "description": story_details["description"],
                        "url": full_story_url,
                        "bookId": book_id,
                        "firstChapterId": first_chap_id,
                    },
                    ensure_ascii=False,
                    indent=4,
                )
            )

        # Crawl Chapters sequentially
        current_chap_id = first_chap_id
        count = 0

        while current_chap_id and count < max_chapters_limit:
            count += 1
            # URL format: /book/{bookId}/{chapterId}
            chap_url = f"{self.url}/book/{book_id}/{current_chap_id}"
            print(f"  - Crawling chapter {count}: {current_chap_id}...")

            chap_html = await self.fetch_page(session, chap_url)
            if not chap_html:
                break

            chap_data = self._extract_chapter_data(chap_html)

            # Save chapter
            ch_file = story_dir / f"chapter_{count}.json"
            async with aiofiles.open(ch_file, mode="w", encoding="utf-8") as f:
                await f.write(
                    json.dumps(
                        {
                            "title": chap_data["title"],
                            "chapter_number": count,
                            "chapter_id": current_chap_id,
                            "content": chap_data["content"],
                            "url": chap_url,
                        },
                        ensure_ascii=False,
                        indent=4,
                    )
                )

            current_chap_id = chap_data["nextChapterId"]

            # Small delay to be nice
            await asyncio.sleep(0.5)

        print(f"  - Finished crawling {count} chapters for '{story_details['title']}'")

    async def run(self, page_range: range, batch_size: int = 1):
        async with aiohttp.ClientSession(headers=self.headers) as session:
            print(f"Fetching story URLs from pages {list(page_range)}...")
            for page in page_range:
                story_urls = await self.get_story_urls(session, page)
                print(f"Found {len(story_urls)} stories on page {page}.")

                for story_url in story_urls:
                    await self.crawl_story_and_chapters(session, story_url)

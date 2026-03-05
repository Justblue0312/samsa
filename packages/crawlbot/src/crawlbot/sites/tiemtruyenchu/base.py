import asyncio
import json
import re
import unicodedata
from typing import Any, Self

import aiofiles
from bs4 import BeautifulSoup
from curl_cffi import requests

from crawlbot.base import BaseStory, Chapter, Story
from crawlbot.settings import DATA_DIR
from crawlbot.utils import build_url, get_site_name


def slugify_vietnamese(text: str) -> str:
    """
    Convert Vietnamese text to a slug.
    Handle Vietnamese specific characters correctly.
    """
    if not text:
        return ""

    # 1. Map Vietnamese characters to ASCII equivalents
    vietnamese_map = {
        "à": "a",
        "á": "a",
        "ả": "a",
        "ã": "a",
        "ạ": "a",
        "ă": "a",
        "ằ": "a",
        "ắ": "a",
        "ẳ": "a",
        "ẵ": "a",
        "ặ": "a",
        "â": "a",
        "ầ": "a",
        "ấ": "a",
        "ẩ": "a",
        "ẫ": "a",
        "ậ": "a",
        "đ": "d",
        "è": "e",
        "é": "e",
        "ẻ": "e",
        "ẽ": "e",
        "ẹ": "e",
        "ê": "e",
        "ề": "e",
        "ế": "e",
        "ể": "e",
        "ễ": "e",
        "ệ": "e",
        "ì": "i",
        "í": "i",
        "ỉ": "i",
        "ĩ": "i",
        "ị": "i",
        "ò": "o",
        "ó": "o",
        "ỏ": "o",
        "õ": "o",
        "ọ": "o",
        "ô": "o",
        "ồ": "o",
        "ố": "o",
        "ổ": "o",
        "ỗ": "o",
        "ộ": "o",
        "ơ": "o",
        "ờ": "o",
        "ớ": "o",
        "ở": "o",
        "ỡ": "o",
        "ợ": "o",
        "ù": "u",
        "ú": "u",
        "ủ": "u",
        "ũ": "u",
        "ụ": "u",
        "ư": "u",
        "ừ": "u",
        "ứ": "u",
        "ử": "u",
        "ữ": "u",
        "ự": "u",
        "ỳ": "y",
        "ý": "y",
        "ỷ": "y",
        "ỹ": "y",
        "ỵ": "y",
    }

    text = text.lower()
    for char, replacement in vietnamese_map.items():
        text = text.replace(char, replacement)
        # Also handle uppercase equivalents if any are left
        text = text.replace(char.upper(), replacement)

    # 2. Normalize to remove any remaining combining marks
    text = unicodedata.normalize("NFKD", text)
    text = "".join([c for c in text if not unicodedata.combining(c)])

    # 3. Replace non-alphanumeric with hyphens
    text = re.sub(r"[^a-z0-9]+", "-", text)

    # 4. Remove duplicate hyphens and trailing/leading hyphens
    text = re.sub(r"-+", "-", text).strip("-")

    return text


class TiemTruyenChu(BaseStory):
    _list_stories_route = "/danh-sach?page={}"
    _story_route = "/truyen/{}"
    _chapter_route = "/doc-truyen/{}/chuong/{}"

    def __init__(self):
        self.url = "https://www.tiemtruyenchu.com"
        self.headers: dict[str, Any] = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36 Edg/145.0.0.0",
        }
        self.story_urls: list[str] = []

    def set_headers(self, headers) -> Self:
        self.headers.update(headers)
        return self

    def get_site_name(self) -> str:
        return get_site_name(self.url)

    async def get_story_urls(self, session: requests.AsyncSession, page: int = 1) -> list[str]:
        """Get a list of story URLs from the given page."""
        if page < 1:
            page = 1
        story_path = build_url(self.url, self._list_stories_route.format(page))
        try:
            response = await session.get(story_path)
            if response.status_code == 200:
                return self._extract_stories(response.text)
            else:
                print(f"Failed to fetch listing page {page}: Status {response.status_code}")
        except Exception as e:
            print(f"Error fetching listing page {page}: {e}")
        return []

    def _extract_stories(self, response: str) -> list[str]:
        """Extract stories from the response text and return a list of story URLs."""
        soup = BeautifulSoup(response, "html.parser")
        story_container = soup.find("div", id="story-list-container")
        if not story_container:
            story_container = soup.find("div", class_=re.compile(r"main|story-list"))

        if not story_container:
            return []

        stories = story_container.find_all("div", class_="story-item")
        story_urls = []
        for story in stories:
            a_tag = story.find("a", class_="story-title")
            if a_tag and a_tag.get("href"):
                story_urls.append(a_tag["href"])
        return story_urls

    def _extract_story(self, response: str) -> Story:
        """Extract the story info from the response text."""
        soup = BeautifulSoup(response, "html.parser")

        title = ""
        header = soup.find("h2", class_="fw-bold")
        if header:
            title_span = header.find("span", class_="align-middle", string=True)  # type: ignore
            if title_span:
                title = title_span.get_text(strip=True)
            else:
                title = header.get_text(separator=" ", strip=True)
                title = re.sub(r"^(Dịch|Convert|Sáng Tác)\s+", "", title)

        description = ""
        info_tab = soup.find("div", id="tab-info")
        if info_tab:
            desc_div = info_tab.find("div", class_="content-text")
            if desc_div:
                description = desc_div.get_text("\n", strip=True)

        max_chapters = 0
        chapter_tab_btn = soup.find("button", attrs={"data-bs-target": "#tab-chuong"})
        if chapter_tab_btn:
            max_chapters = get_text_in_brackets(chapter_tab_btn.get_text())

        return Story(title=title, description=description, max_chapters=max_chapters)

    def _extract_chapters(self, response: str, chapter_num: int) -> Chapter:
        """Extract chapter title and content."""
        soup = BeautifulSoup(response, "html.parser")
        title_attr = soup.find("h2", class_="text-success")
        content_attr = soup.find("div", class_="chapter-content")

        return Chapter(
            title=title_attr.get_text(strip=True) if title_attr else f"Chương {chapter_num}",
            content=content_attr.get_text("\n", strip=True) if content_attr else "",
            chapter_number=chapter_num,
        )

    async def fetch_page(self, session: requests.AsyncSession, url: str) -> str:
        """Generic fetcher for HTML content."""
        try:
            response = await session.get(url, timeout=15)
            if response.status_code == 200:
                return response.text
            else:
                print(f"Failed to fetch {url}: Status {response.status_code}")
        except Exception as e:
            print(f"Error fetching {url}: {e}")
        return ""

    async def crawl_story_and_chapters(self, session: requests.AsyncSession, story_url: str, batch_size: int = 4):
        """Crawl a single story and all its chapters."""
        full_story_url = build_url(self.url, story_url) if story_url.startswith("/") else story_url

        story_id_match = re.search(r"/truyen/(\d+)", story_url)
        story_id = story_id_match.group(1) if story_id_match else story_url.strip("/").split("/")[-1]

        print(f"--- Processing Story: {story_id} ---")

        story_html = await self.fetch_page(session, full_story_url)
        if not story_html:
            return

        story_info = self._extract_story(story_html)
        if not story_info.title:
            story_info.title = f"Story_{story_id}"

        # slugify title for folder name
        story_slug = slugify_vietnamese(story_info.title)
        story_dir = DATA_DIR / self.get_site_name() / story_slug
        story_dir.mkdir(parents=True, exist_ok=True)

        # Save story details
        details_path = story_dir / "story_details.json"
        async with aiofiles.open(details_path, mode="w", encoding="utf-8") as f:
            await f.write(
                json.dumps(
                    {
                        "title": story_info.title,
                        "slug": story_slug,
                        "description": story_info.description,
                        "max_chapters": story_info.max_chapters,
                        "url": full_story_url,
                        "story_id": story_id,
                    },
                    ensure_ascii=False,
                    indent=4,
                )
            )

        # 2. Crawl Chapters in Batches
        max_chaps = story_info.max_chapters or 0
        print(f"Crawling {max_chaps} chapters for '{story_info.title}' (slug: {story_slug})...")

        for i in range(1, max_chaps + 1, batch_size):
            end_idx = min(i + batch_size, max_chaps + 1)
            batch_range = range(i, end_idx)

            tasks = []
            for ch_num in batch_range:
                ch_url = build_url(self.url, self._chapter_route.format(story_id, ch_num))
                tasks.append(self.fetch_page(session, ch_url))

            responses = await asyncio.gather(*tasks)

            for ch_num, ch_html in zip(batch_range, responses):
                if ch_html:
                    chapter = self._extract_chapters(ch_html, ch_num)
                    ch_file = story_dir / f"chapter_{ch_num}.json"
                    async with aiofiles.open(ch_file, mode="w", encoding="utf-8") as f:
                        await f.write(
                            json.dumps(
                                {
                                    "title": chapter.title,
                                    "chapter_number": chapter.chapter_number,
                                    "content": chapter.content,
                                },
                                ensure_ascii=False,
                                indent=4,
                            )
                        )

            print(f"  - Completed chapters {i} to {end_idx - 1}")

    async def run(self, page_range: range, batch_size: int = 4):
        # impersonate='chrome' helps bypass Cloudflare
        async with requests.AsyncSession(headers=self.headers, impersonate="chrome") as session:
            print(f"Fetching story URLs from pages {list(page_range)}...")
            for page in page_range:
                story_urls = await self.get_story_urls(session, page)

                if not story_urls:
                    print(f"No stories found on page {page}.")
                    continue

                print(f"Found {len(story_urls)} stories on page {page}.")

                for story_url in story_urls:
                    await self.crawl_story_and_chapters(session, story_url, batch_size)


def get_text_in_brackets(text):
    """
    Returns the integer found inside parentheses.
    """
    matches = re.findall(r"\((\d+)\)", text)
    if matches:
        return int(matches[0])
    return 0

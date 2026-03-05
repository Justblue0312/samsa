import asyncio

from crawlbot.sites.royalroad.base import RoyalRoad
from crawlbot.sites.scribblehub.base import ScribbleHub
from crawlbot.sites.tiemtruyenchu.base import TiemTruyenChu
from crawlbot.sites.webnovel.base import WebNovel


async def run_bot_tiemtruyenchu():
    bot = TiemTruyenChu()
    # No cookies needed now with curl_cffi
    await bot.run(page_range=range(1, 2), batch_size=4)


async def run_bot_webnovel():
    bot = WebNovel()
    # Webnovel might be sensitive to headers
    bot.set_headers(
        {
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
            "Accept-Language": "en-US,en;q=0.9",
            "Cache-Control": "max-age=0",
            "Sec-Ch-Ua": '"Not A(Brand";v="8", "Chromium";v="132", "Microsoft Edge";v="132"',
            "Sec-Ch-Ua-Mobile": "?0",
            "Sec-Ch-Ua-Platform": '"Windows"',
        }
    )

    # Run crawler for page 1
    await bot.run(page_range=range(1, 2), batch_size=4)


async def run_bot_royalroad():
    bot = RoyalRoad()
    await bot.run(page_range=range(1, 2))


async def run_bot_scribblehub():
    bot = ScribbleHub()
    await bot.run(page_range=range(1, 2))


def main() -> None:
    # You can change this to any of the run_bot_* functions or add CLI args
    asyncio.run(run_bot_tiemtruyenchu())


if __name__ == "__main__":
    main()

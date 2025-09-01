# url_pdf.py
from playwright.async_api import async_playwright
import httpx

UA = ("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
      "KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

async def resolve_pdf_url(public_url: str) -> str:
    """เปิดหน้า public (SPA) → รอ <embed type=application/pdf> → คืนลิงก์ PDF (ตัด #toolbar ออก)"""
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context(user_agent=UA, locale="th-TH")
        page = await context.new_page()
        try:
            await page.goto(public_url, wait_until="networkidle", timeout=45_000)
            await page.wait_for_selector('embed[type="application/pdf"]', timeout=20_000)
            src = await page.eval_on_selector('embed[type="application/pdf"]', "el => el.src")
            if not src:
                raise RuntimeError("พบ <embed> แต่ไม่มีค่า src")
            return src.split("#", 1)[0]
        finally:
            await context.close()
            await browser.close()

async def download_pdf(pdf_url: str, referer: str | None = None) -> bytes:
    """ดาวน์โหลดไฟล์ PDF (แนบ Referer/UA) และยืนยันชนิดเนื้อหา"""
    headers = {"User-Agent": UA}
    if referer:
        headers["Referer"] = referer
    async with httpx.AsyncClient(timeout=60, follow_redirects=True) as client:
        r = await client.get(pdf_url, headers=headers)
        if r.status_code != 200:
            raise RuntimeError(f"ดาวน์โหลดไม่สำเร็จ: HTTP {r.status_code}")
        data = r.content or b""
        ctype = r.headers.get("Content-Type", "").lower()
        if not (ctype.startswith("application/pdf") or data.startswith(b"%PDF")):
            raise RuntimeError(f"เนื้อหาไม่ใช่ PDF (content-type={ctype})")
        return data

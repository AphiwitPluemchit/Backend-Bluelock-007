# url_test.py
# ใช้ทดสอบการเข้าถึงหน้า ThaiMOOC public (SPA) แล้วดึง <embed type="application/pdf">.src
# พร้อมทั้งลองดาวน์โหลดไฟล์เพื่อตรวจสอบว่าเป็น PDF จริง

from pathlib import Path
import asyncio
from playwright.async_api import async_playwright
import httpx
import os
import argparse
from urllib.parse import urlsplit

UA = ("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
      "(KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

async def resolve_pdf_url(public_url: str) -> str:
    # เปิดเบราว์เซอร์ headless แล้วโหลดหน้า SPA ให้ JS รันจนเติม <embed> เข้ามา
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
            return src.split("#", 1)[0]  # ตัดพารามิเตอร์ viewer (#toolbar=...) ทิ้ง
        finally:
            await context.close()
            await browser.close()

async def check_download(pdf_url: str, referer: str | None = None) -> tuple[int, str, int]:
    # ดาวน์โหลดเพื่อตรวจสอบ HTTP 200 และ Content-Type เป็น application/pdf
    headers = {"User-Agent": UA}
    if referer:
        headers["Referer"] = referer
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        r = await client.get(pdf_url, headers=headers)
        return r.status_code, r.headers.get("Content-Type", ""), len(r.content or b"")

async def download_pdf(pdf_url: str, referer: str | None = None) -> bytes:
    """ดาวน์โหลด PDF พร้อมตั้ง header ที่ช่วยให้ผ่าน WAF/ตรวจ Referer"""
    headers = {"User-Agent": UA}
    if referer:
        headers["Referer"] = referer
    async with httpx.AsyncClient(timeout=60, follow_redirects=True) as client:
        r = await client.get(pdf_url, headers=headers)
        if r.status_code != 200:
            raise RuntimeError(f"ดาวน์โหลดไม่สำเร็จ: HTTP {r.status_code}")
        data = r.content or b""
        # บางระบบไม่ใส่ Content-Type ให้ ตรวจ magic header ของ PDF เพิ่มเติม
        if not (r.headers.get("Content-Type","").lower().startswith("application/pdf") or data.startswith(b"%PDF")):
            raise RuntimeError(f"เนื้อหาไม่ใช่ PDF (content-type={r.headers.get('Content-Type')})")
        return data

def deduce_filename(pdf_url: str, out_arg: str | None) -> Path:
    """เดาชื่อไฟล์จาก URL; ถ้ามี -o ให้เคารพค่านั้น (รองรับทั้งโฟลเดอร์/ไฟล์)"""
    if out_arg:
        out_path = Path(out_arg)
        if out_path.is_dir():
            # ถ้า -o เป็นโฟลเดอร์ ให้เอาชื่อจาก URL ไปวางในโฟลเดอร์นั้น
            name = Path(urlsplit(pdf_url).path).name or "certificate.pdf"
            if not name.lower().endswith(".pdf"):
                name += ".pdf"
            return out_path / name
        else:
            # ระบุเป็นชื่อไฟล์ตรง ๆ
            if not out_path.name.lower().endswith(".pdf"):
                out_path = out_path.with_suffix(".pdf")
            return out_path

    # ไม่ได้ระบุ -o → ใช้ชื่อท้าย URL
    name = Path(urlsplit(pdf_url).path).name or "certificate.pdf"
    if not name.lower().endswith(".pdf"):
        name += ".pdf"
    return Path(name)

async def main():
    public_url = "https://learner.thaimooc.ac.th/credential-wallet/10793bb5-6e4f-4873-9309-f25f216a46c7/sahaphap.rit/public"
    
    # set path pdf to pdf folder
    pdf_path = "pdf"
    if not os.path.exists(pdf_path):
        os.makedirs(pdf_path)

    parser = argparse.ArgumentParser(description="Resolve & Download ThaiMOOC certificate PDF")
    parser.add_argument("-o", "--out", type=str, default="ocr/pdf/certificate.pdf", help="Output file name")
    args = parser.parse_args()

    print(f"[1] เปิดหน้า public:\n    {public_url}")

    pdf_url = await resolve_pdf_url(public_url)
    print(f"[2] เจอ PDF URL:\n    {pdf_url}")

    status, ctype, size = await check_download(pdf_url, referer=public_url)
    ok = (status == 200 and "application/pdf" in ctype.lower())
    print(f"[3] ตรวจดาวน์โหลด: status={status}, content-type={ctype}, size={size} bytes")

    data = await download_pdf(pdf_url, referer=public_url)
    out_path = deduce_filename(pdf_url, args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_bytes(data)

    print(f"[4] ดาวน์โหลด PDF: {out_path} : ({len(data)} bytes)")
    print(f"[result] {'OK ✅' if ok else 'NOT OK ❌'}")

if __name__ == "__main__":
    asyncio.run(main())

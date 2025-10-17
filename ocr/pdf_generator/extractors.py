import io
import time
from typing import List, Tuple, Optional

import fitz  # PyMuPDF
from PIL import Image

try:
    import pytesseract
    pytesseract.pytesseract.tesseract_cmd = r"C:\Program Files\Tesseract-OCR\tesseract.exe" 
except Exception:
    pytesseract = None

try:
    import easyocr
except Exception:
    easyocr = None

try:
    from paddleocr import PaddleOCR
except Exception:
    PaddleOCR = None


def extract_text_pymupdf(pdf_bytes: bytes, max_pages: int = 5) -> Tuple[str, float]:
    start = time.time()
    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    parts = []
    total = 0
    for i, page in enumerate(doc):
        parts.append(page.get_text("text"))
        total += len(parts[-1])
        if i + 1 >= max_pages or total > 8000:
            break
    duration = time.time() - start
    return "\n".join(parts), duration


def pdf_to_images_with_fitz(pdf_bytes: bytes, dpi: int = 300, max_pages: int = 2) -> List[Image.Image]:
    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    imgs = []
    for i, page in enumerate(doc):
        if i >= max_pages:
            break
        mat = fitz.Matrix(dpi / 72.0, dpi / 72.0)
        pix = page.get_pixmap(matrix=mat, alpha=False)
        img = Image.frombytes("RGB", [pix.width, pix.height], pix.samples)
        imgs.append(img)
    return imgs


def extract_text_tesseract(pdf_bytes: bytes, psm: int = 6, max_pages: int = 2) -> Tuple[str, float]:
    if pytesseract is None:
        raise RuntimeError("pytesseract not installed")
    imgs = pdf_to_images_with_fitz(pdf_bytes, max_pages=max_pages)
    start = time.time()
    parts = []
    config = f"--psm {psm} -l tha+eng"
    for im in imgs:
        parts.append(pytesseract.image_to_string(im, config=config))
    duration = time.time() - start
    return "\n".join(parts), duration


_easy_reader = None
def extract_text_easyocr(pdf_bytes: bytes, lang_list=('th','en'), max_pages: int = 2) -> Tuple[str, float]:
    global _easy_reader
    if easyocr is None:
        raise RuntimeError("easyocr not installed")
    if _easy_reader is None:
        _easy_reader = easyocr.Reader(lang_list, gpu=False)
    imgs = pdf_to_images_with_fitz(pdf_bytes, max_pages=max_pages)
    start = time.time()
    parts = []
    for im in imgs:
        res = _easy_reader.readtext(im)
        parts.append(" ".join([t[1] for t in res]))
    duration = time.time() - start
    return "\n".join(parts), duration


_paddle = None
def extract_text_paddle(pdf_bytes: bytes, lang: str = 'th', max_pages: int = 2) -> Tuple[str, float]:
    global _paddle
    if PaddleOCR is None:
        raise RuntimeError("paddleocr not installed")
    if _paddle is None:
        _paddle = PaddleOCR(use_angle_cls=True, lang='th', use_gpu=False)
    imgs = pdf_to_images_with_fitz(pdf_bytes, max_pages=max_pages)
    start = time.time()
    parts = []
    for im in imgs:
        res = _paddle.ocr(im, cls=True)
        parts.append(" ".join([line[1][0] for line in res]))
    duration = time.time() - start
    return "\n".join(parts), duration


def extract_all(pdf_bytes: bytes):
    """Return dictionary of extractor_name -> (text, duration)"""
    out = {}
    try:
        out['pymupdf'] = extract_text_pymupdf(pdf_bytes)
    except Exception as e:
        out['pymupdf'] = (f"ERROR: {e}", 0.0)
    try:
        out['tesseract'] = extract_text_tesseract(pdf_bytes)
    except Exception as e:
        out['tesseract'] = (f"ERROR: {e}", 0.0)
    try:
        out['easyocr'] = extract_text_easyocr(pdf_bytes)
    except Exception as e:
        out['easyocr'] = (f"ERROR: {e}", 0.0)
    try:
        out['paddleocr'] = extract_text_paddle(pdf_bytes)
    except Exception as e:
        out['paddleocr'] = (f"ERROR: {e}", 0.0)
    return out

import io
import time
from typing import List, Tuple, Optional

import fitz  # PyMuPDF
from PIL import Image
import numpy as np

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
    for i in range(min(max_pages, doc.page_count)):
        page = doc.load_page(i)
        txt = page.get_text("text")
        parts.append(txt)
        total += len(txt)
        if total > 8000:
            break
    duration = time.time() - start
    return "\n".join(parts), duration


def pdf_to_images_with_fitz(pdf_bytes: bytes, dpi: int = 300, max_pages: int = 2) -> List[Image.Image]:
    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    imgs = []
    for i in range(min(max_pages, doc.page_count)):
        page = doc.load_page(i)
        mat = fitz.Matrix(dpi / 72.0, dpi / 72.0)
        pix = page.get_pixmap(matrix=mat, alpha=False)
        img = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
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
        # EasyOCR expects a file path, bytes, or a numpy array. Convert PIL Image -> numpy array
        im_arr = np.array(im)
        res = _easy_reader.readtext(im_arr, detail=0)
        # res is now a list of strings (texts)
        # Join each detected text with newline to preserve line structure
        texts = [str(text) for text in res]
        parts.append("\n".join(texts))
    duration = time.time() - start
    return "\n".join(parts), duration


_paddle = None
def extract_text_paddle(pdf_bytes: bytes, lang: str = 'th', max_pages: int = 2) -> Tuple[str, float]:
    global _paddle
    if PaddleOCR is None:
        raise RuntimeError("paddleocr not installed")
    if _paddle is None:
        # Some paddleocr versions do not accept `use_gpu` in the constructor.
        # Pass only supported args (use_angle_cls, lang). GPU usage is controlled elsewhere.
        _paddle = PaddleOCR(use_angle_cls=True, lang=lang)
    imgs = pdf_to_images_with_fitz(pdf_bytes, max_pages=max_pages)
    start = time.time()
    parts = []
    for im in imgs:
        # PaddleOCR expects file path or numpy array; convert PIL Image -> numpy array
        im_arr = np.array(im)
        # Call ocr without cls parameter (use use_angle_cls in constructor instead)
        res = _paddle.ocr(im_arr)
        # `res` is a list of lists: each element contains [box, (text, prob)]
        if res and res[0]:
            parts.append("\n".join([str(line[1][0]) for line in res[0]]))
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

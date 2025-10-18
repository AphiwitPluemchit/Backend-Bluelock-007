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
        # Use new parameter name for newer PaddleOCR versions
        _paddle = PaddleOCR(use_textline_orientation=True, lang=lang)
    imgs = pdf_to_images_with_fitz(pdf_bytes, max_pages=max_pages)
    start = time.time()
    parts = []
    for im in imgs:
        # PaddleOCR expects file path or numpy array; convert PIL Image -> numpy array
        im_arr = np.array(im)
        # Call ocr - newer versions return OCRResult objects
        res = _paddle.ocr(im_arr)
        
        # Handle both old and new PaddleOCR result formats
        if res and len(res) > 0:
            ocr_result = res[0]
            
            # New format: OCRResult object with 'rec_texts' attribute/key
            if hasattr(ocr_result, 'get') and 'rec_texts' in ocr_result:
                texts = ocr_result['rec_texts']
                boxes = ocr_result.get('rec_polys', []) if 'rec_polys' in ocr_result else []
                
                if texts and boxes and len(texts) == len(boxes):
                    # Group texts by their vertical position (y-coordinate) to form lines
                    # This helps combine words that are on the same line
                    text_with_pos = []
                    for i, (text, box) in enumerate(zip(texts, boxes)):
                        if isinstance(box, np.ndarray) and len(box) > 0:
                            # Get average y coordinate of the box (for line grouping)
                            y_coords = box[:, 1] if box.ndim == 2 else [box[1]]
                            avg_y = float(np.mean(y_coords))
                            # Get leftmost x coordinate (for sorting within a line)
                            x_coords = box[:, 0] if box.ndim == 2 else [box[0]]
                            min_x = float(np.min(x_coords))
                            text_with_pos.append((avg_y, min_x, str(text)))
                    
                    # Sort by y coordinate first (top to bottom)
                    text_with_pos.sort(key=lambda t: t[0])
                    
                    # Group texts that are on similar y-coordinates into lines
                    lines = []
                    current_line_items = []
                    current_y = None
                    y_threshold = 30  # pixels - adjust based on font size
                    
                    for y, x, text in text_with_pos:
                        if current_y is None or abs(y - current_y) < y_threshold:
                            # Same line - add with x position for later sorting
                            current_line_items.append((x, text))
                            if current_y is None:
                                current_y = y
                            else:
                                # Update current_y to be average of items in line
                                current_y = (current_y + y) / 2
                        else:
                            # New line - sort current line by x position (left to right) and join
                            if current_line_items:
                                current_line_items.sort(key=lambda t: t[0])
                                line_text = ' '.join([t[1] for t in current_line_items])
                                lines.append(line_text)
                            current_line_items = [(x, text)]
                            current_y = y
                    
                    # Don't forget the last line
                    if current_line_items:
                        current_line_items.sort(key=lambda t: t[0])
                        line_text = ' '.join([t[1] for t in current_line_items])
                        lines.append(line_text)
                    
                    if lines:
                        parts.append("\n".join(lines))
                elif texts:
                    # Fallback if boxes not available
                    parts.append("\n".join([str(t) for t in texts]))
            # Old format: list of [box, (text, confidence)]
            elif isinstance(ocr_result, list) and len(ocr_result) > 0:
                if isinstance(ocr_result[0], (list, tuple)) and len(ocr_result[0]) >= 2:
                    parts.append("\n".join([str(line[1][0]) for line in ocr_result]))
    
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

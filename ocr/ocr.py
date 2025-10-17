from enum import unique
from dotenv import load_dotenv
from PIL import Image, ImageOps, ImageFilter
from io import BytesIO
from typing import List
import fitz
import os
import pytesseract
import numpy as np
import logging
load_dotenv()

path = os.getenv('TESSERACT_PATH')

if path:
    pytesseract.pytesseract.tesseract_cmd = path

# Logger for OCR module
logger = logging.getLogger("ocr")
if not logger.handlers:
    # If no handlers configured, set a reasonable default. Do not override existing logging config.
    logging.basicConfig(level=logging.INFO)

def ocr_images_tesseract(images: List[Image.Image], psm: int = 6) -> str:
    # PSM 6: บล็อกข้อความทั่วไป; ลองเปลี่ยนเป็น 4 ถ้าเลย์เอาต์แนวนิตยสาร
    cfg = f"--oem 3 --psm {psm}"
    logger.info("OCR engine: tesseract (pytesseract). pages=%d psm=%d", len(images), psm)
    texts = []
    index = 0
    for img in images:
        index += 1
        prep = preprocess_for_ocr(img,index)
        texts.append(pytesseract.image_to_string(prep, lang="tha+eng", config=cfg))
        
    return "\n".join(texts)


_easyocr_reader = None

def ocr_images_easyocr(images: List[Image.Image]) -> str:
    """Run OCR using easyocr. Initializes the reader lazily. Returns joined text per page."""
    try:
        import easyocr
    except Exception as e:
        raise RuntimeError(f"easyocr not available: {e}")

    global _easyocr_reader
    if _easyocr_reader is None:
        # Force CPU-only operation per requirement (do not use GPU)
        # easyocr.Reader accepts gpu=False to disable GPU usage.
        logger.info("Initializing EasyOCR reader (languages: th,en) - CPU only")
        _easyocr_reader = easyocr.Reader(['th', 'en'], gpu=False)

    texts = []
    for idx, img in enumerate(images, start=1):
        arr = np.array(img)
        try:
            results = _easyocr_reader.readtext(arr, detail=0)
        except Exception as e:
            # If easyocr fails for a page, continue with empty result for that page
            logger.warning("easyocr readtext error on page %d: %s", idx, e)
            results = []
        texts.append(" ".join(map(str, results)))

    return "\n".join(texts)


def ocr_images(images: List[Image.Image], psm: int = 6) -> str:
    engine = os.getenv('OCR_ENGINE', 'tesseract').lower()
    logger.info("Selected OCR_ENGINE=%s", engine)
    if engine in ('easyocr', 'easy'):
        return ocr_images_easyocr(images)
    return ocr_images_tesseract(images, psm=psm)


def pdf_to_images_with_fitz(pdf_data: bytes, dpi: int = 300, max_pages: int = 2) -> List[Image.Image]:
    doc = fitz.open(stream=pdf_data, filetype="pdf")
    zoom = dpi / 72.0
    mat = fitz.Matrix(zoom, zoom)
    images: List[Image.Image] = []

    # iterate by index to avoid typing/iterator issues with PyMuPDF's Document stubs
    total_pages = len(doc)
    pages_to_process = min(total_pages, max_pages)
    for i in range(pages_to_process):
        page = doc.load_page(i)
        pix = page.get_pixmap(matrix=mat, alpha=False)  # RGB, no alpha
        img = Image.open(BytesIO(pix.tobytes("png")))   # สะดวกสุด
        images.append(img)
    return images

def preprocess_for_ocr(img: Image.Image, index: int) -> Image.Image:
    # เรียบง่ายและได้ผลทั่วๆไป
    gray = ImageOps.grayscale(img)
    gray = ImageOps.autocontrast(gray, cutoff=1)   # ลด clipping นิดหน่อย
    gray = gray.filter(ImageFilter.SHARPEN)
    # หมายเหตุ: บางใบแปลงเป็นขาวดำแข็งๆอาจแย่ลง เลยส่งเป็น grayscale ให้ Tesseract ตัดสินเอง

    # บันทึกภาพ สำหรับ debug
    # gray.save("gray_{}.png".format(index))
    return gray


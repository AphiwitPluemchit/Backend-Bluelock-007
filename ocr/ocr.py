from enum import unique
from dotenv import load_dotenv
from PIL import Image, ImageOps, ImageFilter
from io import BytesIO
from typing import List
import fitz
import os
import pytesseract
load_dotenv()

path = os.getenv('TESSERACT_PATH')

if path:
    pytesseract.pytesseract.tesseract_cmd = path

def ocr_images_tesseract(images: List[Image.Image], psm: int = 6) -> str:
    # PSM 6: บล็อกข้อความทั่วไป; ลองเปลี่ยนเป็น 4 ถ้าเลย์เอาต์แนวนิตยสาร
    cfg = f"--oem 3 --psm {psm}"
    texts = []
    index = 0
    for img in images:
        index += 1
        prep = preprocess_for_ocr(img,index)
        texts.append(pytesseract.image_to_string(prep, lang="tha+eng", config=cfg))
        
    return "\n".join(texts)


def pdf_to_images_with_fitz(pdf_data: bytes, dpi: int = 300, max_pages: int = 2) -> List[Image.Image]:
    doc = fitz.open(stream=pdf_data, filetype="pdf")
    zoom = dpi / 72.0
    mat = fitz.Matrix(zoom, zoom)
    images: List[Image.Image] = []

    for i, page in enumerate(doc):
        pix = page.get_pixmap(matrix=mat, alpha=False)  # RGB, no alpha
        img = Image.open(BytesIO(pix.tobytes("png")))   # สะดวกสุด
        images.append(img)
        if i + 1 >= max_pages:
            break
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


import fitz
from text_norm import normalize_min, strip_prefix 
from fuzzy_match import best_score
from ocr import pdf_to_images_with_fitz, ocr_images_tesseract

# --------- main verify (ไม่ใช้ OCR) ---------
def thaimooc_verify(pdf_data: bytes, student_th: str, student_en: str, course_name: str, course_name_en: str):
    # Step A: text layer (เร็ว/แม่นกว่า)
    raw_text = extract_textlayer(pdf_data)
    hay_text = ''

    # ถ้า text layer ว่างมาก ให้ถือว่าไม่ผ่าน (ภายหลังค่อย fallback เป็น OCR)
    used_ocr = False
    if len(hay_text) < 10:
        # Step B: ไม่มี/น้อยมาก → OCR fallback
        used_ocr = True
        imgs = pdf_to_images_with_fitz(pdf_data, dpi=300, max_pages=2)  # ส่วนใหญ่ 1 หน้า
        if not imgs:
            return {"isVerified": False, "isNameMatch": False, "isCourseMatch": False}
        hay_text = ocr_images_tesseract(imgs, psm=6)
 
    # normalize ข้อความรวมทั้ง expected
    hay = normalize_min(hay_text, remove_thai_internal_spaces=True,  remove_all_spaces=False)
    print(hay)
    stu_th = normalize_min(strip_prefix(student_th))
    print(stu_th)
    stu_en = normalize_min(strip_prefix(student_en))
    print(stu_en)
    crs    = normalize_min(course_name)
    print(crs)
    crs_en = normalize_min(course_name_en)
    print(crs_en)

    # ชื่อ: ใช้คะแนนที่ดีที่สุดระหว่าง TH/EN
    name_score_th = best_score(stu_th, hay)
    name_score_en = best_score(stu_en, hay)
    course_score = best_score(crs, hay)
    course_score_en = best_score(crs_en, hay)

    # เกณฑ์เบื้องต้น (ปรับได้ตามจริง)
    isNameMatch   = name_score_th >= 95 or name_score_en >= 95
    isCourseMatch = course_score >= 95 or course_score_en >= 95
    isVerified    = isNameMatch and isCourseMatch

    return {
        "isVerified": isVerified,
        "isNameMatch": isNameMatch,
        "isCourseMatch": isCourseMatch,
        "nameScoreTh": name_score_th,
        "nameScoreEn": name_score_en,
        "courseScore": course_score,
        "courseScoreEn": course_score_en,
        "usedOcr": used_ocr,
    }


def extract_textlayer(pdf_data: bytes, max_pages: int = 5) -> str:
    doc = fitz.open(stream=pdf_data, filetype="pdf")
    parts = []
    total = 0
    for i, page in enumerate(doc):
        parts.append(page.get_text("text"))
        total += len(parts[-1])
        # พอทดสอบ: จำกัดหน้าเพื่อความไว (เช่น 5 หน้าแรก หรือจนกว่าข้อความจะเยอะพอ)
        if i + 1 >= max_pages or total > 8000:
            break
    return "\n".join(parts)



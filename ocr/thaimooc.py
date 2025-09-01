import fitz
from text_norm import normalize_min, strip_prefix 
from fuzzy_match import best_score


# --------- main verify (ไม่ใช้ OCR) ---------
def thaimooc_verify(pdf_data: bytes, student_th: str, student_en: str, course_name: str):
    raw_text = _extract_textlayer(pdf_data)
    # ถ้า text layer ว่างมาก ให้ถือว่าไม่ผ่าน (ภายหลังค่อย fallback เป็น OCR)
    if len(raw_text.strip()) < 10:
        return {
            "isVerified": False,
            "isNameMatch": False,
            "isCourseMatch": False,
            # "reason": "No selectable text in PDF (likely scanned)."
        }
 
    print("raw_text", raw_text)

    # normalize ข้อความรวมทั้ง expected
    hay = normalize_min(raw_text.replace("\n", " "))
    stu_th = normalize_min(strip_prefix(student_th))
    stu_en = normalize_min(strip_prefix(student_en))
    crs    = normalize_min(course_name)

    # ชื่อ: ใช้คะแนนที่ดีที่สุดระหว่าง TH/EN
    name_score = max(best_score(stu_th, hay), best_score(stu_en, hay))
    course_score = best_score(crs, hay)

    # เกณฑ์เบื้องต้น (ปรับได้ตามจริง)
    isNameMatch   = name_score   >= 85
    isCourseMatch = course_score >= 80
    isVerified    = isNameMatch and isCourseMatch

    print("isVerified", isVerified)
    print("isNameMatch", isNameMatch)
    print("isCourseMatch", isCourseMatch)
    print("name_score", name_score)
    print("course_score", course_score)

    return {
        "isVerified": isVerified,
        "isNameMatch": isNameMatch,
        "isCourseMatch": isCourseMatch,
        "nameScore": name_score,
        "courseScore": course_score,
        # ถ้าต้องการ debug ระหว่างทดสอบ ค่อยปลดคอมเมนต์:
        # "scores": {"name": name_score, "course": course_score, "textLen": len(hay)},
    }


def _extract_textlayer(pdf_data: bytes, max_pages: int = 5) -> str:
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
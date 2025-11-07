from models import BuuInput
from text_norm import normalize_min, strip_prefix
from fuzzy_match import best_score
from typing import Optional
import re


def _safe_norm_optional(s: Optional[str]) -> str:
    if not s:
        return ""
    return normalize_min(s)


def _strip_html_tags(html: str) -> str:
    """Strip HTML tags and extract only text content"""
    if not html:
        return ""
    # Remove script and style elements
    html = re.sub(r'<script[^>]*>.*?</script>', ' ', html, flags=re.DOTALL | re.IGNORECASE)
    html = re.sub(r'<style[^>]*>.*?</style>', ' ', html, flags=re.DOTALL | re.IGNORECASE)
    # Remove HTML tags
    html = re.sub(r'<[^>]+>', ' ', html)
    # Decode common HTML entities
    html = html.replace('&nbsp;', ' ')
    html = html.replace('&amp;', '&')
    html = html.replace('&lt;', '<')
    html = html.replace('&gt;', '>')
    html = html.replace('&quot;', '"')
    html = html.replace('&#39;', "'")
    # Compress multiple spaces
    html = re.sub(r'\s+', ' ', html).strip()
    return html


def buumooc_verify(payload: BuuInput):
    # Strip HTML tags first, then normalize
    html_text = _strip_html_tags(payload.html)
    html_norm = normalize_min(html_text)
    
    # Normalize expected values; handle optional english fields
    student_norm = normalize_min(strip_prefix(payload.student_th))
    student_norm_en = _safe_norm_optional(payload.student_en)
    course_norm = normalize_min(payload.course_name)
    course_norm_en = _safe_norm_optional(payload.course_name_en)
    
    # Try to extract student name from "is presented to" section
    name_section = ""
    # Extract everything after "is presented to" until we hit common delimiters
    presented_to_pattern = r'is\s+presented\s+to\s+(.+?)(?:\s+has\s+|\s+for\s+|\.|\n|$)'
    match = re.search(presented_to_pattern, html_text, re.IGNORECASE)
    if match:
        name_section = normalize_min(match.group(1).strip())
        print(f"Extracted Name Section: {name_section}")
    
    # log normalized values for debugging
    # print(f"Stripped HTML Text: {html_text[:400]}...")
    print(f"Normalized HTML: {html_norm[:400]}...")  # print first
    print(f"Normalized Student TH: {student_norm}")
    print(f"Normalized Student EN: {student_norm_en}")
    print(f"Normalized Course Name: {course_norm}")
    print(f"Normalized Course Name EN: {course_norm_en}")

    # Compute name scores: try name_section first, fallback to full html_norm
    if name_section:
        # เทียบกับ name section ก่อน
        name_score = best_score(student_norm, name_section)
        name_score_en = best_score(student_norm_en, name_section) if student_norm_en else 0
        print(f"Name scores from 'is presented to' section - TH: {name_score}, EN: {name_score_en}")
        
        # ถ้าคะแนนต่ำกว่า 70 ให้ลองเทียบกับ HTML ทั้งหมด
        # if name_score < 70 and name_score_en < 70:
        #     name_score_fallback = best_score(student_norm, html_norm)
        #     name_score_en_fallback = best_score(student_norm_en, html_norm) if student_norm_en else 0
        #     print(f"Fallback to full HTML - TH: {name_score_fallback}, EN: {name_score_en_fallback}")
        #     # ใช้คะแนนที่ดีกว่า
        #     name_score = max(name_score, name_score_fallback)
        #     name_score_en = max(name_score_en, name_score_en_fallback)
    else:
        # ถ้าไม่เจอ name section ให้เทียบกับ HTML ทั้งหมดเลย
        name_score = best_score(student_norm, html_norm)
        name_score_en = best_score(student_norm_en, html_norm) if student_norm_en else 0
        print(f"No 'is presented to' section found, using full HTML matching")
    
    # Course scores ยังเทียบกับ HTML ทั้งหมดตามเดิม
    course_score = best_score(course_norm, html_norm)
    course_score_en = best_score(course_norm_en, html_norm) if course_norm_en else 0

    isNameMatch = (name_score >= 95) or (name_score_en >= 95)
    isCourseMatch = (course_score >= 95) or (course_score_en >= 95)

    return {
        "isVerified": isNameMatch and isCourseMatch,
        "isNameMatch": isNameMatch,
        "isCourseMatch": isCourseMatch,
        "nameScoreTh": name_score,
        "nameScoreEn": None if student_norm_en == "" else name_score_en,
        "courseScore": course_score,
        "courseScoreEn": None if course_norm_en == "" else course_score_en,
        "usedOcr": False,
    }


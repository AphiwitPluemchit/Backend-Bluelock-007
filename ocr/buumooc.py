from models import BuuInput
from text_norm import normalize_min, strip_prefix
from fuzzy_match import best_score
from typing import Optional


def _safe_norm_optional(s: Optional[str]) -> str:
    if not s:
        return ""
    return normalize_min(s)


def buumooc_verify(payload: BuuInput):
    # normalize html and expected values; handle optional english fields
    html_norm = normalize_min(payload.html.replace("\n", " "))
    student_norm = normalize_min(strip_prefix(payload.student_th))
    student_norm_en = _safe_norm_optional(payload.student_en)
    course_norm = normalize_min(payload.course_name)
    course_norm_en = _safe_norm_optional(payload.course_name_en)

    # compute scores (best_score safely handles empty needle/hay)
    name_score = best_score(student_norm, html_norm)
    name_score_en = best_score(student_norm_en, html_norm) if student_norm_en else 0
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


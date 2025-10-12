from models import BuuInput
from text_norm import normalize_min, strip_prefix
from fuzzy_match import best_score

def buumooc_verify(payload: BuuInput):
    # ใช้งาน
    html_norm = normalize_min(payload.html.replace("\n", " "))
    student_norm   = normalize_min(strip_prefix(payload.student_th))
    course_norm    = normalize_min(payload.course_name)

    name_score   = best_score(student_norm, html_norm)
    name_score_en = best_score(payload.student_en, html_norm)
    course_score = best_score(course_norm, html_norm)
    course_score_en = best_score(payload.course_name_en, html_norm)

    if name_score >= 95 or name_score_en >= 95:
        isNameMatch = True
    else:
        isNameMatch = False

    isCourseMatch = course_score >= 95 or course_score_en >= 95

    return {
        "isVerified": True,
        "isNameMatch": isNameMatch,
        "isCourseMatch": isCourseMatch,
        "nameScore": name_score,
        "nameScoreEn": name_score_en,
        "courseScore": course_score,
        "courseScoreEn": course_score_en,
    }


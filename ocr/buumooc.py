from models import BuuInput
from text_norm import normalize_min, strip_prefix
from fuzzy_match import best_score

def buumooc_verify(payload: BuuInput):
    # ใช้งาน
    full_text_norm = normalize_min(payload.html.replace("\n", " "))
    student_norm   = normalize_min(strip_prefix(payload.student_th))
    course_norm    = normalize_min(payload.course_name)

    name_score   = best_score(student_norm, full_text_norm)
    name_score_en = best_score(payload.student_en, full_text_norm)
    course_score = best_score(course_norm, full_text_norm)

    if name_score >= 95 or name_score_en >= 95:
        isNameMatch = True
    else:
        isNameMatch = False

    isCourseMatch = course_score >= 95

    print("name_score", name_score)
    print("course_score", course_score)
    print("isNameMatch", isNameMatch)
    print("isCourseMatch", isCourseMatch)
    return {
        "ok": True,
        "received": {
            "student_th": payload.student_th,
            "student_en": payload.student_en,
            "course_name": payload.course_name,
            "isNameMatch": isNameMatch,
            "isCourseMatch": isCourseMatch,
        }
    }


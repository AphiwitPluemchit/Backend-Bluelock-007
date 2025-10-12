from pydantic import BaseModel
from typing import Optional


class BuuInput(BaseModel):
    html: str
    student_th: str
    # optional: sometimes the english name may not be provided
    student_en: Optional[str] = None
    course_name: str
    # optional: course name in english may be missing
    course_name_en: Optional[str] = None

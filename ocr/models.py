from pydantic import BaseModel

class BuuInput(BaseModel):
    html: str
    student_th: str
    student_en: str
    course_name: str

import io
import random
from typing import List
from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.lib.pagesizes import A4, landscape
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont

# Simple PDF synthesizer: draws a background image and writes a selectable text layer
TEMPLATES_DIR = Path(__file__).parent
OUT_DIR = TEMPLATES_DIR / "synth_pdfs"
OUT_DIR.mkdir(exist_ok=True)

# Register a font that supports Thai (if available in system). We'll try common ones.
def register_fonts():
    candidates = [
        "C:/Windows/Fonts/THSarabunNew.ttf",
        "C:/Windows/Fonts/TH Sarabun New.ttf",
        "/usr/share/fonts/truetype/thsarabun/THSarabunNew.ttf",
    ]
    for i, p in enumerate(candidates):
        try:
            p = Path(p)
            if p.exists():
                pdfmetrics.registerFont(TTFont(f"TH{i}", str(p)))
                return f"TH{i}"
        except Exception:
            continue
    # fallback to Helvetica
    return "Helvetica"


def synthesize_pdfs(names_th: List[str], courses_th: List[str], names_en: List[str], courses_en: List[str], n: int = 10):
    font_name = register_fonts()
    # fallback defaults when caller passes empty lists
    if not names_th:
        names_th = [
            "ปวรปัชญ์ ศิริพิสุทธิวิมล",
            "สหภาพ ฤทธิ์เนติกุล",
            "นางสาว สมศรี ตัวอย่าง",
        ]
    if not courses_th:
        courses_th = [
            "เตรียมสหกิจศึกษา (15 ชั่วโมงการเรียนรู้)",
            "การสร้างหน้าเว็บด้วย HTML และ CSS (10 ชั่วโมง)",
        ]
    if not names_en:
        names_en = [
            "Paworpatch Siripisutthiwimon",
            "Sahapap Rithnetikul",
            "Somsri Example",
        ]
    if not courses_en:
        courses_en = [
            "Introduction to Web Development (10h)",
            "Preparation for Cooperative Education (15h)",
        ]
    for i in range(n):
        name_th = random.choice(names_th)
        course_th = random.choice(courses_th)
        name_en = random.choice(names_en)
        course_en = random.choice(courses_en)

        out = OUT_DIR / f"synth_{i+1:03d}.pdf"
        c = canvas.Canvas(str(out), pagesize=landscape(A4))
        width, height = landscape(A4)

        # Background: simple shapes to mimic certificate header
        c.setFillColorRGB(0.95, 0.95, 0.96)
        c.rect(0, height * 0.6, width, height * 0.4, fill=1, stroke=0)

        # Title
        c.setFont(font_name, 36)
        c.setFillColorRGB(0.05, 0.05, 0.06)
        c.drawString(80, height - 120, "CERTIFICATE OF COMPLETION")

        # Thai name (center)
        c.setFont(font_name, 28)
        c.drawCentredString(width / 2, height / 2 + 30, name_th)

        # Course name (Thai)
        c.setFont(font_name, 18)
        c.drawCentredString(width / 2, height / 2 - 10, course_th)

        # English name below (text-layer)
        c.setFont(font_name, 14)
        c.drawCentredString(width / 2, height / 2 - 40, name_en)
        c.drawCentredString(width / 2, height / 2 - 60, course_en)

        # Footer small text
        c.setFont(font_name, 10)
        c.drawString(40, 40, f"Generated ID: synth-{i+1:03d}")

        c.showPage()
        c.save()

    print(f"Wrote {n} synth PDFs to: {OUT_DIR}")


if __name__ == '__main__':
    # small sample lists (Thai/EN)
    names_th = [
        "ปวรปัชญ์ ศิริพิสุทธิวิมล",
        "สหภาพ ฤทธิ์เนติกุล",
        "นางสาว สมศรี ตัวอย่าง",
    ]
    courses_th = [
        "เตรียมสหกิจศึกษา (15 ชั่วโมงการเรียนรู้)",
        "การสร้างหน้าเว็บด้วย HTML และ CSS (10 ชั่วโมง)",
    ]
    names_en = [
        "Paworpatch Siripisutthiwimon",
        "Sahapap Rithnetikul",
        "Somsri Example",
    ]
    courses_en = [
        "Introduction to Web Development (10h)",
        "Preparation for Cooperative Education (15h)",
    ]

    synthesize_pdfs(names_th, courses_th, names_en, courses_en, n=10)

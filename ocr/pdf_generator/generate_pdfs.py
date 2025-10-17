import io
import io
import random
from typing import List, Optional
from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.lib.pagesizes import A4, landscape
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from PIL import Image
import fitz
import tempfile

# Simple PDF synthesizer: draws a background image (template.pdf/png) and writes a selectable text layer
TEMPLATES_DIR = Path(__file__).parent
OUT_DIR = TEMPLATES_DIR / "synth_pdfs"
OUT_DIR.mkdir(exist_ok=True)


def find_thai_font() -> Optional[str]:
    """Search common Windows/Linux font locations for a Thai-capable TTF and return path."""
    candidates = [
        "C:/Windows/Fonts/THSarabunNew.ttf",
        "C:/Windows/Fonts/TH Sarabun New.ttf",
        "C:/Windows/Fonts/Sarabun-Regular.ttf",
        "C:/Windows/Fonts/LeelawUI.ttf",
        "C:/Windows/Fonts/Leelawadee UI.ttf",
        "C:/Windows/Fonts/LeelUIsl.ttf",
        "C:/Windows/Fonts/ARIALUNI.TTF",
        "/usr/share/fonts/truetype/thsarabun/THSarabunNew.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    ]
    # local fonts folder first
    local_fonts = TEMPLATES_DIR / "fonts"
    if local_fonts.exists():
        for f in local_fonts.glob("*.ttf"):
            try:
                return str(f)
            except Exception:
                continue

    for p in candidates:
        try:
            path = Path(p)
            if path.exists():
                return str(path)
        except Exception:
            continue
    # Search Windows Fonts dir generically
    win_fonts = Path("C:/Windows/Fonts")
    if win_fonts.exists():
        for f in win_fonts.glob("*.ttf"):
            name = f.name.lower()
            if any(k in name for k in ("sara", "leelaw", "arialuni", "tahoma")):
                return str(f)
    return None


def _render_template_to_image(template_path: Path, width: int, height: int) -> Optional[str]:
    """If template is PDF or image, render first page to a temporary PNG and return path."""
    if not template_path.exists():
        return None
    suffix = template_path.suffix.lower()
    tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".png")
    tmp.close()
    out_path = Path(tmp.name)
    try:
        if suffix == ".pdf":
            doc = fitz.open(str(template_path))
            page = doc.load_page(0)
            # increase scale for better clarity (use higher matrix)
            scale = 4.0
            mat = fitz.Matrix(scale, scale)
            pix = page.get_pixmap(matrix=mat, alpha=False)
            pix.save(str(out_path))
        else:
            # assume image - use high-quality resize
            img = Image.open(str(template_path))
            img = img.convert("RGB")
            # compute target pixels; ReportLab uses points but drawImage expects image pixels
            scale = 4
            target_w = max(1, int(width * scale))
            target_h = max(1, int(height * scale))
            # Pillow resampling compatibility: prefer Resampling.LANCZOS, fallback to ANTIALIAS
            try:
                resample_filter = Image.Resampling.LANCZOS
            except Exception:
                # older Pillow exposes ANTIALIAS
                resample_filter = getattr(Image, 'ANTIALIAS', Image.NEAREST)
            img = img.resize((target_w, target_h), resample=resample_filter)
            img.save(str(out_path))
        return str(out_path)
    except Exception:
        try:
            # fallback: copy as-is
            Image.open(str(template_path)).save(str(out_path))
            return str(out_path)
        except Exception:
            return None


def synthesize_pdfs(names_th: List[str], courses_th: List[str], names_en: List[str], courses_en: List[str], n: int = 10, template: Optional[str] = None):
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

    # try to find a Thai-capable font
    # Prefer Sarabun from local fonts folder (if present)
    font_path = None
    local_sarabun = TEMPLATES_DIR / "fonts" / "Sarabun" / "Sarabun-Regular.ttf"
    if local_sarabun.exists():
        font_path = str(local_sarabun)
    else:
        # Prefer Sarabun if present on system
        candidates_pref = [
            "C:/Windows/Fonts/THSarabunNew.ttf",
            "C:/Windows/Fonts/Sarabun-Regular.ttf",
            "C:/Windows/Fonts/TH Sarabun New.ttf",
        ]
        for p in candidates_pref:
            if Path(p).exists():
                font_path = p
                break
        if not font_path:
            font_path = find_thai_font()
    font_name = None
    if font_path:
        try:
            font_name = "TH_FONT"
            pdfmetrics.registerFont(TTFont(font_name, font_path))
        except Exception:
            font_name = None

    if not font_name:
        # fallback
        font_name = "Helvetica"
    else:
        print(f"Registered font for PDF text layer: {font_path} (name={font_name})")

    # optional template: prefer template/template.png (subfolder) then files in script dir
    template_path = None
    if template:
        template_path = Path(template)
    else:
        # first check subfolder 'template'
        for cand in (TEMPLATES_DIR / "template" / "template.pdf", TEMPLATES_DIR / "template" / "template.png", TEMPLATES_DIR / "template" / "template.jpg"):
            if cand.exists():
                template_path = cand
                break
        if template_path is None:
            for cand in (TEMPLATES_DIR / "template.pdf", TEMPLATES_DIR / "template.png", TEMPLATES_DIR / "template.jpg"):
                if cand.exists():
                    template_path = cand
                    break

    for i in range(n):
        name_th = random.choice(names_th)
        course_th = random.choice(courses_th)
        name_en = random.choice(names_en)
        course_en = random.choice(courses_en)

        out = OUT_DIR / f"synth_{i+1:03d}.pdf"
        c = canvas.Canvas(str(out), pagesize=landscape(A4))
        width, height = landscape(A4)

        # If template exists, render and draw it as full-page image
        if template_path:
            img_path = _render_template_to_image(Path(template_path), int(width), int(height))
            if img_path:
                try:
                    c.drawImage(img_path, 0, 0, width=width, height=height)
                except Exception:
                    pass
        else:
            # simple background: light rectangle
            c.setFillColorRGB(0.95, 0.95, 0.96)
            c.rect(0, height * 0.6, width, height * 0.4, fill=1, stroke=0)

        # Thai name: centered, slightly smaller to avoid overlapping diacritics
        name_y = height * 0.53
        c.setFont(font_name, 28)
        c.drawCentredString(width / 2, name_y, name_th)

        # Insert requested Thai sentence under name (small)
        info_y = name_y - 28
        c.setFont(font_name, 14)
        c.drawCentredString(width / 2, info_y, "ได้ผ่านเกณฑ์หลักสูตรออน์ไลน์จนได้รับประกาศนียบัตรในรายวิชา")

        # Course name (Thai): centered below the info line
        course_y = info_y - 30
        c.setFont(font_name, 18)
        c.drawCentredString(width / 2, course_y, course_th)

        # English name and course: smaller and lower (if present)
        c.setFont(font_name, 12)
        c.drawCentredString(width / 2, course_y - 30, name_en)
        c.drawCentredString(width / 2, course_y - 48, course_en)

        # Footer small text (left) and small generated id
        c.setFont(font_name, 10)
        c.drawString(40, 40, f"Generated ID: synth-{i+1:03d}")
        # signature area note (not drawing signature) - reserved bottom-right
        # we keep the template signature visible; text layer won't overlap

        c.showPage()
        c.save()

    print(f"Wrote {n} synth PDFs to: {OUT_DIR}")


def verify_pdf_text(pdf_path: Path, max_pages: int = 2):
    try:
        doc = fitz.open(str(pdf_path))
        parts = []
        page_count = min(max_pages, doc.page_count)
        for i in range(page_count):
            page = doc.load_page(i)
            parts.append(page.get_text("text"))
        text = "\n".join(parts)
        print("--- Extracted text preview ---")
        print(text)
        return text
    except Exception as e:
        print("verify_pdf_text: error", e)
        return ""


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

    # if you have a template, put it at the same folder as this script and name it template.pdf/png
    synthesize_pdfs(names_th, courses_th, names_en, courses_en, n=3)
    # quick verification of the first generated PDF
    first = OUT_DIR / "synth_001.pdf"
    if first.exists():
        verify_pdf_text(first)
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

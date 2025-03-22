from fastapi import FastAPI, File, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from pdf2image import convert_from_bytes
from PIL import Image, ImageFilter
import pytesseract
import re
import io
import unicodedata
from fuzzywuzzy import fuzz, process
import requests
from bs4 import BeautifulSoup

# print all packages version
# import pkg_resources
# installed_packages = pkg_resources.working_set
# for package in installed_packages:
#     print(package.key, package.version)

# pytesseract.pytesseract.tesseract_cmd = r"C:\\Program Files\\Tesseract-OCR\\tesseract.exe"

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

course_db = {
    "ภาษาอังกฤษเพื่อการสื่อสาร (English for Communication)": 10,
    "Experiential English": 12,
    "การประยุกต์ใช้ Generative AI ในการทำงาน": 15,
    "การประยุตก์ใช้ Collaboration tools ในการเพิ่มประสิทธิภาพในการทำงานและประสานงานภายในองค์กร": 15,
    "Data Visualization with Tableau Desktop": 5,
}

def validate_url(url: str) -> bool:
    try:
        response = requests.head(url, timeout=5)
        return response.status_code == 200
    except requests.RequestException:
        return False

def compare_with_url_data(url, ocr_name, ocr_course):
    try:
        response = requests.get(url, timeout=5)
        if response.status_code != 200:
            print("❌ ไม่สามารถเข้าถึง URL ได้")
            return False

        html = response.text
        soup = BeautifulSoup(html, "html.parser")
        text = soup.get_text(separator="\n")
        print("🌐 Raw text จาก URL:\n", text)

        name_match = fuzz.partial_ratio(ocr_name.lower(), text.lower()) > 99
        course_match = fuzz.partial_ratio(ocr_course.lower(), text.lower()) > 99

        if name_match and course_match:
            print("✅ ข้อมูลตรงกับ URL จริง (ชื่อ + วิชา)")
            return True
        else:
            print("❌ ข้อมูลไม่ตรงกับ URL (อาจปลอม/ผิด)")
            return False

    except Exception as e:
        print("❌ เกิดข้อผิดพลาดในการตรวจสอบ URL:", str(e))
        return False

def clean_text(text):
    text = unicodedata.normalize("NFKC", text)
    return re.sub(r"\s+", "", text)

def fuzzy_match_course_name(course_raw):
    cleaned_input = clean_text(course_raw)
    cleaned_keys = [clean_text(k) for k in course_db.keys()]
    best_match, score = process.extractOne(cleaned_input, cleaned_keys, scorer=fuzz.token_sort_ratio)
    if score > 40:
        return list(course_db.keys())[cleaned_keys.index(best_match)]
    return course_raw

def extract_url_from_image(image: Image.Image):
    width, height = image.size
    cropped = image.crop((0, int(height * 0.92), int(width * 0.5), height))
    cropped = cropped.resize((cropped.width * 3, cropped.height * 3))
    cropped = cropped.convert("L")
    cropped = cropped.point(lambda x: 0 if x < 180 else 255, '1')
    cropped = cropped.filter(ImageFilter.SHARPEN)
    cropped.save("debug_url_zone.png")

    config = "--psm 7 --oem 3"
    text = pytesseract.image_to_string(cropped, lang="eng", config=config)
    print("🧾 OCR URL zone:", repr(text))

    cleaned = text.replace(" ", "").replace("|", "").replace("\n", "").strip()
    cleaned = re.sub(r'O', '0', cleaned)
    cleaned = re.sub(r'¢', 'c', cleaned)
    
    print("🧹 Cleaned:", cleaned)

    base_match = re.search(r'(https?://[a-zA-Z0-9./_\-]+/certificates/)', cleaned)
    uuid_parts = re.findall(r'[a-fA-F0-9]{4,}', cleaned)
    joined_uuid = ''.join(uuid_parts)

    if base_match and len(joined_uuid) >= 32:
        final_url = base_match.group(1) + joined_uuid[:32]
        print("✅ Final Cleaned URL:", final_url)
        return final_url
    else:
        print("❌ ไม่พบ base URL หรือ UUID")
        return ""

def extract_fields_from_image(image: Image.Image):
    text = pytesseract.image_to_string(image, lang="eng+tha")
    print("🧠 OCR Full Text:\n", text)

    direct_url_match = re.search(r'https?://[^\s]+', text)
    direct_url = direct_url_match.group(0) if direct_url_match else ""
    direct_url = re.sub(r'O', '0', direct_url)
    print("🔍 URL (จาก full text):", direct_url)

    if validate_url(direct_url):
        final_url = direct_url
        print("✅ ใช้ลิงก์จาก full text OCR ได้เลย:", final_url)
    else:
        print("❌ ลิงก์จาก full text ใช้ไม่ได้ ลอง OCR จากล่างสุดแทน")
        final_url = extract_url_from_image(image)

    name_match = re.search(r"presented to\s+(.+)", text)
    student_name = name_match.group(1).strip() if name_match else ""

    course_match = re.search(r"completed the Open Online Course\s+(.+)", text)
    course_raw = course_match.group(1).strip() if course_match else ""
    course_name = fuzzy_match_course_name(course_raw)

    date_match = re.search(r"On\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})", text)
    course_date = date_match.group(1).strip() if date_match else ""

    # ✅ เปรียบเทียบกับข้อมูลในหน้า URL (ถ้ามี)
    match_with_url = compare_with_url_data(final_url, student_name, course_name)
    print("🧾 เปรียบเทียบกับ URL:", match_with_url)

    fields = {
        "student_name": student_name,
        "course_name": course_name,
        "date": course_date,
        "url": final_url,
        "verified": match_with_url
    }

    print("✅ Fields Extracted:", fields)
    return fields

@app.post("/ocr")
async def ocr_certificate(file: UploadFile = File(...)):
    print("🚀 [FastAPI] เรียก /ocr แล้ว")
    filename = file.filename.lower()
    contents = await file.read()
    print("📥 ได้รับไฟล์:", filename)

    if filename.endswith(".pdf"):
        print("📄 แปลง PDF เป็นภาพ...")
        images = convert_from_bytes(contents, dpi=400)
        image = images[0]
    else:
        print("🖼️ โหลดภาพโดยตรง...")
        image = Image.open(io.BytesIO(contents))

    print("🔍 เริ่ม OCR...")
    fields = extract_fields_from_image(image)
    return {"status": "success", "data": fields}
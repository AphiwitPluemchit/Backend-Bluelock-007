import csv
from pathlib import Path
from generate_pdfs import synthesize_from_records, OUT_DIR

OUT_DIR.mkdir(exist_ok=True)

# More varied samples including honorifics, long names, punctuation, numeric suffixes, diacritics
names_th = [
    "นาย สมหมาย ใจดี",
    "นางสาว สมศรี ตัวอย่าง",
    "ดร. วิทยา พิทักษ์",
    "ผศ.ดร. วีระชัย ศรีสุข",
    "นพ. ภูมิพัฒน์ เมธี",
    "เด็กชาย ธันวา น้อย",
    "เด็กหญิง ดาวเรือง จิตร",
    "สุภาพสตรียอดเยี่ยม ภูมิใจ",
    "นาย กิตติพงษ์ (Kittipong)",
    "นาง Apinya-Somchai",
]
# Add some very long Thai names to test wrapping/diacritics
long_names_th = [
    "นาย สมชาย พิพัฒน์สมบูรณ์กิจ ผู้มีความยาวชื่อมากเป็นกรณีทดสอบเพื่อดูการตัดคำและวรรณยุกต์",
    "นางสาว ประภัสสร จันทร์จำปา วงศ์สุวรรณสกุล นักพัฒนาเทคโนโลยีสารสนเทศและการสื่อสาร",
]

courses_th = [
    "การวิเคราะห์ข้อมูลเชิงสถิติและการประยุกต์ (40 ชั่วโมง)",
    "การพัฒนาซอฟต์แวร์เชิงปฏิบัติ (32 ชั่วโมง)",
    "การออกแบบ UX/UI สำหรับผู้เริ่มต้น (16 ชั่วโมง)",
    "ภาษาไทยสำหรับวิชาชีพ (12 ชั่วโมง)",
]

names_en = [
    "Sommai Jaidee",
    "Somsri Example",
    "Dr. Witya Pitak",
    "Assoc. Prof. Weerachai Srisuk",
    "Kittipong (K.)",
    "Apinya-Somchai",
]

courses_en = [
    "Advanced Data Analysis (40h)",
    "Practical Software Development (32h)",
    "Intro to UX/UI (16h)",
    "Thai for Professionals (12h)",
]

records = []
# Create 100 varied records combining pools and adding edge cases
for i in range(1, 101):
    if i % 15 == 0:
        name_th = long_names_th[i % len(long_names_th)]
    else:
        name_th = names_th[i % len(names_th)]
    # add some punctuation and numeric suffixes occasionally
    if i % 7 == 0:
        name_th = f"{name_th} #{i}"
    course_th = courses_th[i % len(courses_th)]
    name_en = names_en[i % len(names_en)]
    course_en = courses_en[i % len(courses_en)]

    r = {
        'pdf_filename': f"varied_{i:03d}.pdf",
        'name_th': name_th,
        'course_th': course_th,
        'name_en': name_en,
        'course_en': course_en,
    }
    records.append(r)

# write labels CSV
labels_path = Path(__file__).parent / 'labels_varied.csv'
with open(labels_path, 'w', newline='', encoding='utf-8') as f:
    writer = csv.DictWriter(f, fieldnames=['pdf_filename', 'name_th', 'course_th', 'name_en', 'course_en'])
    writer.writeheader()
    for r in records:
        writer.writerow(r)

print(f"Wrote labels CSV: {labels_path}")

# synthesize PDFs from the labels
written = synthesize_from_records(records)
print(f"Wrote {len(written)} PDFs to: {OUT_DIR}")

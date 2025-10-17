import csv
from pathlib import Path
from generate_pdfs import synthesize_from_records, OUT_DIR

OUT_DIR.mkdir(exist_ok=True)

# Simple sample pools - you can replace with your full list
names_th = [
    "ปวรปัชญ์ ศิริพิสุทธิวิมล",
    "สหภาพ ฤทธิ์เนติกุล",
    "นางสาว สมศรี ตัวอย่าง",
    "นาย สมหมาย ใจดี",
    "ดร. วิทยา พิทักษ์",
]
courses_th = [
    "เตรียมสหกิจศึกษา (15 ชั่วโมงการเรียนรู้)",
    "การสร้างหน้าเว็บด้วย HTML และ CSS (10 ชั่วโมง)",
    "การวิเคราะห์ข้อมูลเบื้องต้น (20 ชั่วโมง)",
]
names_en = [
    "Paworpatch Siripisutthiwimon",
    "Sahapap Rithnetikul",
    "Somsri Example",
    "Sommai Jaidee",
    "Dr. Witya Pitak",
]
courses_en = [
    "Preparation for Cooperative Education (15h)",
    "Introduction to Web Development (10h)",
    "Intro to Data Analysis (20h)",
]

records = []
for i in range(1, 101):
    r = {
        'pdf_filename': f"labeled_{i:03d}.pdf",
        'name_th': names_th[i % len(names_th)],
        'course_th': courses_th[i % len(courses_th)],
        'name_en': names_en[i % len(names_en)],
        'course_en': courses_en[i % len(courses_en)],
    }
    records.append(r)

# write labels CSV
labels_path = Path(__file__).parent / 'labels.csv'
with open(labels_path, 'w', newline='', encoding='utf-8') as f:
    writer = csv.DictWriter(f, fieldnames=['pdf_filename', 'name_th', 'course_th', 'name_en', 'course_en'])
    writer.writeheader()
    for r in records:
        writer.writerow(r)

print(f"Wrote labels CSV: {labels_path}")

# synthesize PDFs from the labels
written = synthesize_from_records(records)
print(f"Wrote {len(written)} PDFs to: {OUT_DIR}")

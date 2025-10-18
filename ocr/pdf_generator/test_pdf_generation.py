"""
Test script to verify the updated create_varied_labels_and_pdfs.py
Creates a small sample to validate before generating all 500 PDFs.
"""
import csv
from pathlib import Path
from generate_pdfs import synthesize_from_records, OUT_DIR

# Test with just 5 PDFs first
names_th = ["นาย ทดสอบ ระบบ", "นางสาว สมศรี ตัวอย่าง", "ดร. วิทยา พิทักษ์"]
names_en = ["Test System", "Somsri Example", "Dr. Witya Pitak"]
courses_th = ["การพัฒนาซอฟต์แวร์ (32 ชั่วโมง)", "การวิเคราะห์ข้อมูล (40 ชั่วโมง)"]
courses_en = ["Software Development (32h)", "Data Analysis (40h)"]

records = []
for i in range(1, 6):
    r = {
        'pdf_filename': f"test_{i:03d}.pdf",
        'name_th': names_th[i % len(names_th)],
        'course_th': courses_th[i % len(courses_th)],
        'name_en': names_en[i % len(names_en)],
        'course_en': courses_en[i % len(courses_en)],
    }
    records.append(r)

print(f"Creating {len(records)} test PDFs...")
written = synthesize_from_records(records)
print(f"✓ Created {len(written)} test PDFs in {OUT_DIR}")
print(f"\nTest files:")
for w in written:
    print(f"  - {w.name}")

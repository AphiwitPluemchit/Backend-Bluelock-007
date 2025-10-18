import csv
from pathlib import Path
from generate_pdfs import synthesize_from_records, OUT_DIR
import random
import argparse

OUT_DIR.mkdir(exist_ok=True)

# 50 varied Thai names with different honorifics and structures
names_th = [
    "นาย สมชาย ใจดี",
    "นางสาว สมศรี ตัวอย่าง",
    "ดร. วิทยา พิทักษ์",
    "ผศ.ดร. วีระชัย ศรีสุข",
    "นพ. ภูมิพัฒน์ เมธี",
    "เด็กชาย ธันวา น้อย",
    "เด็กหญิง ดาวเรือง จิตร",
    "นาง ประภา สุขสันต์",
    "นาย กิตติพงษ์ วงศ์สวัสดิ์",
    "นางสาว อรอนงค์ บุญมี",
    "ดร. ประเสริฐ มั่นคง",
    "รศ.ดร. สุรชัย เจริญผล",
    "ผศ. นิธิวัฒน์ แสงทอง",
    "นาย วิศรุต ศักดิ์ศรี",
    "นางสาว กนกวรรณ อุดมสุข",
    "นพ. สมศักดิ์ วัฒนธรรม",
    "พญ. อรทัย สุขเกษม",
    "เด็กชาย ปฏิภาณ ดีเลิศ",
    "เด็กหญิง ธิดารัตน์ รุ่งเรือง",
    "นาย อานนท์ ชัยวัฒน์",
    "นางสาว พิมพ์ชนก สมบูรณ์",
    "ดร. ธนพล เจริญรุ่ง",
    "รศ. ชัยณรงค์ พัฒนกิจ",
    "นาง วิไลวรรณ สุวรรณ",
    "นาย ณัฐพล มีสุข",
    "นางสาว จิราพร แก้วใส",
    "ดร. สุรพล ทองดี",
    "ผศ.ดร. อภิชาติ วิทยากร",
    "นาย ธีรพงษ์ ศรีประเสริฐ",
    "นางสาว กัญญา บุญธรรม",
    "นพ. พิพัฒน์ รักษ์ศิลป์",
    "นาง สุดารัตน์ เพชรรัตน์",
    "นาย ชนินทร์ อุดมพร",
    "นางสาว ณัฐริกา สวัสดี",
    "ดร. อดิศักดิ์ เกียรติกุล",
    "รศ.ดร. มนต์ชัย ปัญญาวัน",
    "นาย ธนากร ธรรมศิริ",
    "นางสาว ปาริชาติ สุขใจ",
    "ผศ. กฤษฎา วงศ์วิทยา",
    "นาง รัตนา พูลสวัสดิ์",
    "นาย ชัยวัฒน์ มงคล",
    "นางสาว วราภรณ์ แสงแก้ว",
    "ดร. ประพันธ์ เลิศบุญ",
    "นพ. ธวัชชัย รัตนพันธ์",
    "นาย ศุภชัย จันทรา",
    "นางสาว อัญชลี ดวงใจ",
    "รศ. ปิยะพงศ์ ศรีทอง",
    "นาง สุภาพร วิชัยดิษฐ",
    "นาย พงศ์พัฒน์ อนันต์",
    "นางสาว ธัญญารัตน์ ธนบดี",
]

# 20 varied Thai courses with different durations and topics
courses_th = [
    "การวิเคราะห์ข้อมูลเชิงสถิติและการประยุกต์ (40 ชั่วโมง)",
    "การพัฒนาซอฟต์แวร์เชิงปฏิบัติ (32 ชั่วโมง)",
    "การออกแบบ UX/UI สำหรับผู้เริ่มต้น (16 ชั่วโมง)",
    "ภาษาไทยสำหรับวิชาชีพ (12 ชั่วโมง)",
    "การจัดการโครงการเทคโนโลยีสารสนเทศ (24 ชั่วโมง)",
    "ปัญญาประดิษฐ์และการเรียนรู้เชิงลึก (48 ชั่วโมง)",
    "การพัฒนาเว็บแอปพลิเคชันสมัยใหม่ (36 ชั่วโมง)",
    "ความมั่นคงปลอดภัยทางไซเบอร์ (28 ชั่วโมง)",
    "การตลาดดิจิทัลและโซเชียลมีเดีย (20 ชั่วโมง)",
    "การวิเคราะห์ธุรกิจและข้อมูลเชิงลึก (32 ชั่วโมง)",
    "การพัฒนาแอปพลิเคชันมือถือ (40 ชั่วโมง)",
    "การออกแบบกราฟิกและมัลติมีเดีย (24 ชั่วโมง)",
    "การบริหารจัดการฐานข้อมูล (30 ชั่วโมง)",
    "เทคโนโลยีบล็อกเชนและสกุลเงินดิจิทัล (16 ชั่วโมง)",
    "การพัฒนาเกมและกราฟิก 3 มิติ (44 ชั่วโมง)",
    "การเขียนโปรแกรม Python ขั้นสูง (28 ชั่วโมง)",
    "คลาวด์คอมพิวติ้งและ DevOps (36 ชั่วโมง)",
    "การออกแบบและพัฒนา API (20 ชั่วโมง)",
    "วิทยาศาสตร์ข้อมูลและการวิเคราะห์ (48 ชั่วโมง)",
    "อินเทอร์เน็ตของสรรพสิ่งและระบบฝังตัว (32 ชั่วโมง)",
]

# 50 varied English names matching Thai names
names_en = [
    "Somchai Jaidee",
    "Somsri Example",
    "Dr. Witya Pitak",
    "Assoc. Prof. Weerachai Srisuk",
    "Dr. Phumipat Methee",
    "Thanwa Noi",
    "Daoreung Chit",
    "Prapa Sooksan",
    "Kittipong Wongsawat",
    "Oranong Boonmee",
    "Dr. Prasert Munkhong",
    "Assoc. Prof. Surachai Charoenphon",
    "Asst. Prof. Nithiwat Saengthong",
    "Wisarut Saksri",
    "Kanokwan Udomsuk",
    "Dr. Somsak Wattanatham",
    "Dr. Orathai Sukkasem",
    "Patipan Deelerd",
    "Thidarat Rungruang",
    "Anon Chaiwat",
    "Pimchanok Somboon",
    "Dr. Thanapol Charoenrung",
    "Assoc. Prof. Chainarong Pattanakij",
    "Wilaiwan Suwan",
    "Natthaphon Meesuk",
    "Jiraporn Kaewsai",
    "Dr. Suraphon Thongdee",
    "Asst. Prof. Aphichat Witthayakorn",
    "Theerapong Sriprasert",
    "Kanya Boontham",
    "Dr. Phiphat Raksilp",
    "Sudarat Phetrat",
    "Chanin Udomporn",
    "Nattarika Sawatdee",
    "Dr. Adisak Kiatkul",
    "Assoc. Prof. Monchai Panyawan",
    "Thanakorn Thammasiri",
    "Parichat Sukchai",
    "Asst. Prof. Kritsada Wongwitthaya",
    "Rattana Poolsawat",
    "Chaiwat Mongkhon",
    "Waraporn Saengkaew",
    "Dr. Praphan Lerdbooon",
    "Dr. Thawatchai Rattanaphan",
    "Suphachai Chantra",
    "Anchalee Duangjai",
    "Assoc. Prof. Piyapong Srithong",
    "Suphaporn Wichaidit",
    "Phongphat Anan",
    "Thanyarat Thanabodee",
]

# 20 varied English courses matching Thai courses
courses_en = [
    "Statistical Data Analysis & Applications (40h)",
    "Practical Software Development (32h)",
    "Intro to UX/UI Design (16h)",
    "Thai for Professionals (12h)",
    "IT Project Management (24h)",
    "Artificial Intelligence & Deep Learning (48h)",
    "Modern Web Application Development (36h)",
    "Cybersecurity Fundamentals (28h)",
    "Digital Marketing & Social Media (20h)",
    "Business Analytics & Data Insights (32h)",
    "Mobile Application Development (40h)",
    "Graphic Design & Multimedia (24h)",
    "Database Administration (30h)",
    "Blockchain Technology & Cryptocurrency (16h)",
    "Game Development & 3D Graphics (44h)",
    "Advanced Python Programming (28h)",
    "Cloud Computing & DevOps (36h)",
    "API Design & Development (20h)",
    "Data Science & Analytics (48h)",
    "Internet of Things & Embedded Systems (32h)",
]

# Some extra long Thai names for edge case testing
long_names_th = [
    "นาย สมชาย พิพัฒน์สมบูรณ์กิจ",
    "นางสาว ประภัสสร จันทร์จำปา วงศ์สุวรรณสกุล",
    "ดร. อภิสิทธิ์ เจริญพงศ์พัฒนากิจการ",
]

long_names_en = [
    "Somchai Phipatsomboonkij",
    "Prapasorn Chanjampa Wongsuwan",
    "Dr. Apisit Chareonphongphattanakijkan",
]


def create_pdfs(num_pdfs=500):
    """Generate varied PDF certificates with different names and courses.
    
    Args:
        num_pdfs: Number of PDFs to generate (default: 500)
    """
    # Generate varied PDFs
    records = []

    print(f"Generating {num_pdfs} varied certificate records...")

    for i in range(1, num_pdfs + 1):
        # Occasionally use long names (every 15th record)
        if i % 15 == 0 and i % len(long_names_th) < len(long_names_th):
            name_th = long_names_th[i % len(long_names_th)]
            name_en = long_names_en[i % len(long_names_en)]
        else:
            name_th = names_th[i % len(names_th)]
            name_en = names_en[i % len(names_en)]
        
        # Add variation: some names with punctuation/suffixes
        if i % 7 == 0:
            name_th = f"{name_th} (รุ่น {i//7})"
            name_en = f"{name_en} (Batch {i//7})"
        
        # Randomly select courses to increase variety
        course_idx = i % len(courses_th)
        course_th = courses_th[course_idx]
        course_en = courses_en[course_idx]

        r = {
            'pdf_filename': f"varied_{i:03d}.pdf",
            'name_th': name_th,
            'course_th': course_th,
            'name_en': name_en,
            'course_en': course_en,
        }
        records.append(r)

    print(f"Generated {len(records)} records")

    # Write labels CSV
    labels_path = Path(__file__).parent / 'labels_varied.csv'
    print(f"Writing labels CSV to: {labels_path}")

    with open(labels_path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=['pdf_filename', 'name_th', 'course_th', 'name_en', 'course_en'])
        writer.writeheader()
        for r in records:
            writer.writerow(r)

    print(f"✓ Wrote {len(records)} records to labels CSV")

    # Synthesize PDFs from the labels
    print(f"\nGenerating {len(records)} PDF files...")
    print("This may take a while...")

    written = synthesize_from_records(records)

    print(f"\n{'='*60}")
    print(f"✓ Successfully created {len(written)} PDF certificates!")
    print(f"✓ Output directory: {OUT_DIR}")
    print(f"✓ Labels file: {labels_path}")
    print(f"{'='*60}")
    print(f"\nSummary:")
    print(f"  - Total PDFs: {len(written)}")
    print(f"  - Unique names: {len(names_th)} Thai + {len(names_en)} English")
    print(f"  - Unique courses: {len(courses_th)} Thai + {len(courses_en)} English")
    print(f"  - Variations: Long names, batch numbers, punctuation")
    print(f"\nTo run benchmark on all PDFs:")
    print(f"  python benchmark.py")
    print(f"\nTo run benchmark on first 10 PDFs:")
    print(f"  python benchmark.py --limit 10")


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Generate varied certificate PDFs for OCR benchmarking'
    )
    parser.add_argument(
        '-n', '--num',
        type=int,
        default=500,
        help='Number of PDFs to generate (default: 500)'
    )
    parser.add_argument(
        '--quick-test',
        action='store_true',
        help='Quick test mode: generate only 10 PDFs'
    )
    
    args = parser.parse_args()
    
    # Determine number of PDFs
    num_pdfs = 10 if args.quick_test else args.num
    
    print(f"\n{'='*60}")
    print(f"PDF Certificate Generator")
    print(f"{'='*60}")
    print(f"Mode: {'Quick Test' if args.quick_test else 'Full Generation'}")
    print(f"PDFs to generate: {num_pdfs}")
    print(f"{'='*60}\n")
    
    create_pdfs(num_pdfs)

How to run

PostInstall => poppler

1. Create virtual environment with python -m venv .venv
2. Activate virtual environment with .venv\Scripts\activate
3. Install dependencies with pip install -r requirements.txt
4. Run the app with uvicorn main:app --reload

How to use

1. Send POST request to /ocr
2. Include studentName, courseName, cer_type in the request body
3. Include file in the request body



.env
TESSERACT_PATH=C:\Program Files\Tesseract-OCR\tesseract.exe
MODE=development


// ปัญหา OCR Tesseract ตอนนี้คือ เมื่อใช้ตรวจ 2 ภาษาพร้อมกัน ทำให้การดึงข้อความนั้นไม่สมบูรณ์ เช่น Cer ที่มี ภาษาไทยและ Eng ปนรวมกัน
https://colab.research.google.com/drive/1CZ6PqMy0t1WedvZ0Y959rg5G4ewWRrtl?usp=sharing


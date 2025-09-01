# app.py
from fastapi import FastAPI, UploadFile, File, Form, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from buumooc import buumooc_verify
from models import BuuInput
from thaimooc import thaimooc_verify

app = FastAPI(title="Cert Verify (minimal)")

# CORS แบบง่าย
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"], allow_methods=["*"], allow_headers=["*"],
)

@app.get("/health")
async def health():
    return {"status": "ok"}

# ---------- 1) BUU MOOC: รับ HTML + ชื่อ นศ. (TH/EN) + ชื่อคอร์ส ----------
@app.post("/buumooc")
async def buumooc_receive(payload: BuuInput):
    # แค่ยืนยันว่ารับครบ และส่งสรุปกลับ
    return buumooc_verify(payload)

# ---------- 2) ThaiMOOC: รับ PDF (multipart) + ชื่อ นศ. (TH/EN) + ชื่อคอร์ส ----------
@app.post("/thaimooc")
async def thaimooc_receive(
    pdf: UploadFile = File(...),
    student_th: str = Form(...),
    student_en: str = Form(...),
    course_name: str = Form(...),
):
    # ตรวจ content-type แบบหลวม ๆ พอทดสอบ
    if pdf.content_type not in {"application/pdf", "application/octet-stream", "binary/octet-stream"}:
        raise HTTPException(status_code=415, detail=f"Unsupported file type: {pdf.content_type}")

    pdf_data = await pdf.read()

    return thaimooc_verify(pdf_data, student_th, student_en, course_name)

# รันทดสอบ: uvicorn app:app --reload --host 0.0.0.0 --port 8000

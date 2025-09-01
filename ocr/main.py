# app.py
from fastapi import FastAPI, UploadFile, File, Form, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

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
class BuuInput(BaseModel):
    html: str
    student_th: str
    student_en: str
    course_name: str

@app.post("/buumooc")
async def buumooc_receive(payload: BuuInput):
    # แค่ยืนยันว่ารับครบ และส่งสรุปกลับ
    print("payload", payload.html != None)
    print("student_th", payload.student_th)
    print("student_en", payload.student_en)
    print("course_name", payload.course_name)
    return {
        "ok": True,
        "received": {
            "student_th": payload.student_th,
            "student_en": payload.student_en,
            "course_name": payload.course_name,
            "html_len": len(payload.html),
        }
    }

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

    data = await pdf.read()
    print("data", data != None)
    print("student_th", student_th)
    print("student_en", student_en)
    print("course_name", course_name)
    return {
        "ok": True,
        "received": {
            "student_th": student_th,
            "student_en": student_en,
            "course_name": course_name,
            "pdf_filename": pdf.filename,
            "pdf_bytes": len(data),
        }
    }

# รันทดสอบ: uvicorn app:app --reload --host 0.0.0.0 --port 8000

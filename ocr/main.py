from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import logging
from utils import validate_file_type, process_ocr_from_file

app = FastAPI(
    title="Thai Certificate OCR API",
    description="API for extracting information from Thai certificates using Tesseract OCR with Thai language support",
    version="1.0.0"
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

@app.post("/ocr", summary="Extract information from Thai certificate image")
async def ocr_certificate(file: UploadFile = File(...), studentName: str = "", courseName: str = "",cer_type: str = ""):
    """
    Process a certificate image or PDF and extract relevant information
    """
    logger.info(f"🚀 Processing OCR request for file: {file.filename}")

    if not studentName or not courseName:
        raise HTTPException(status_code=400, detail="Student name and course name are required.")

    try:
        contents = await file.read()
        if not validate_file_type(file.content_type):
            raise HTTPException(status_code=400, detail="Invalid file type. Only image and PDF files are allowed.")
        
        fields = process_ocr_from_file(contents, file.content_type, studentName, courseName,cer_type)
        return {"status": "success", "data": fields}
    
    except Exception as e:
        logger.error(f"Error processing OCR: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error processing OCR: {str(e)}")
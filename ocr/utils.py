import io
from pdf_to_image import convert_pdf_to_images
from ocr import extract_fields_from_image
from PIL import Image

def validate_file_type(content_type: str) -> bool:
    """
    Validate the file type (only images and PDFs are allowed)
    """
    return content_type.startswith('image/') or content_type.startswith('application/pdf')

def process_ocr_from_file(contents: bytes, content_type: str, studentName: str, courseName: str, courseType: str):
    """
    Process OCR based on file type (image or PDF)
    """
    if content_type.startswith('image/'):
        image = Image.open(io.BytesIO(contents))
        return extract_fields_from_image(image, studentName, courseName, courseType)
    
    elif content_type.startswith('application/pdf'):
        images = convert_pdf_to_images(contents)

        # use first image
        image = images[0]
        return extract_fields_from_image(image, studentName, courseName, courseType)

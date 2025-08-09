from pdf2image import convert_from_bytes
import logging

logger = logging.getLogger(__name__)

def convert_pdf_to_images(pdf_file: bytes) -> list:
    """
    Convert PDF file to list of image objects
    Args:
        pdf_file: PDF byte stream
    Returns:
        List of PIL Image objects
    """
    try:
        # ใช้ convert_from_bytes เพื่อแปลง PDF จาก byte stream
        images = convert_from_bytes(pdf_file)
        logger.info(f"Converted PDF to {len(images)} images.")
        return images
    except Exception as e:
        logger.error(f"Error converting PDF: {str(e)}")
        raise

from dotenv import load_dotenv
import os
import pytesseract
from PIL import Image
from preprocess import preprocess_image,crop_image
from pythainlp.util import normalize
import logging
import re
import requests
from fuzzywuzzy import fuzz
import numpy as np
load_dotenv()



# ตั้งค่าพาธของ Tesseract
path = os.getenv('TESSERACT_PATH')

if path:
    pytesseract.pytesseract.tesseract_cmd = path

# pytesseract.pytesseract.tesseract_cmd = os.getenv('TESSERACT_PATH')  # Windows path ปิดไว้เพราะใช้ docker ติดตั้งบน Ubuntu


logger = logging.getLogger(__name__)

def extract_fields_from_image(image: Image.Image, studentName: str, courseName: str, courseType: str) -> dict:
    """
    Extract relevant fields from the certificate image
    """

    # Debug log
    logger.info(f"🧠 Student Name: {studentName}")
    logger.info(f"🧠 Course Name: {courseName}")
    logger.info(f"🧠 Course Type: {courseType}")

    # Preprocess the image
    preprocessed_image = preprocess_image(image)

    image_np = np.array(preprocessed_image)
    
    # Perform OCR to get the text
    full_text = pytesseract.image_to_string(image_np, lang='eng+tha')

    # Normalize Thai text
    full_text = normalize(full_text)

    logger.info(f"🧠 OCR Full Text:\n{full_text}")

    # Initialize defaults to avoid UnboundLocalError
    url = ""
    isNameMatch = False
    isCourseMatch = False

    if courseType == "buumooc":
        # Extract URL 
        url = extract_url_from_cropped_image(preprocessed_image, courseType)

        # Check if URL matches name and course name only if URL was found
        if url is not None:
            isNameMatch, isCourseMatch = url_matching(url, studentName, courseName)

    elif courseType == "thaimooc":
        # Remove all \n in full_text
        full_text = full_text.replace("\n", " ")  # ใช้ space แทน \n เพื่อไม่ให้ข้อความติดกันมากเกินไป

        # Fuzzy Matching สำหรับการตรวจจับชื่อ
        name_match_score = fuzz.partial_ratio(studentName.lower(), full_text.lower())
        logger.info(f"🧠 Fuzzy Matching Name Score: {name_match_score}")
        isNameMatch = name_match_score >= 90  # ตั้งค่า threshold ไว้ที่ 90% สำหรับการจับคู่ชื่อ

        # Fuzzy Matching สำหรับชื่อหลักสูตร
        course_match_score = fuzz.partial_ratio(courseName.lower(), full_text.lower())
        logger.info(f"🧠 Fuzzy Matching Course Score: {course_match_score}")
        isCourseMatch = course_match_score >= 90  # ตั้งค่า threshold ไว้ที่ 90% สำหรับการจับคู่ชื่อหลักสูตร
 
    print("url: ", url)
    print("isNameMatch: ", isNameMatch)
    print("isCourseMatch: ", isCourseMatch)
    print("full_text: ", full_text)


    if os.getenv('MODE') == 'production':
        return {
            "url": url,
            "isNameMatch": isNameMatch,
            "isCourseMatch": isCourseMatch,
        }

    else:
        return {
            "student_name": studentName,
            "course_name": courseName,
            "courseType": courseType,
            "url": url,
            "isNameMatch": isNameMatch,
            "isCourseMatch": isCourseMatch,
            "full_text": full_text,
        }

def extract_url_from_cropped_image(image: Image.Image,courseType: str) -> str:
    """
    Perform OCR on the cropped image to extract the URL
    """
    # Crop the image to focus on the bottom-left portion
    cropped_image = crop_image(image)
    image_np = np.array(cropped_image)
    # Perform OCR to get the text
    full_text = pytesseract.image_to_string(image_np, lang='eng+tha')



   # ดึง Certificate ID Number จาก full_text โดยดึงข้อความหลัง Certificate ID Number :
    url_match = re.search(r'Certificate ID Number\s*:\s*([^\n\r]+)', full_text)

    if url_match:
        url = url_match.group(1).strip()
        # Clean the URL by removing any unnecessary spaces
        url = re.sub(r'\s+', '', url)
        # check url is id or http
        if not url.startswith('http'):
            if courseType == "buumooc":
                url = 'https://mooc.buu.ac.th/certificates/' + url
        return url

    else:
        # Fallback: find a plain URL in the text
        fallback = re.search(r'https?://[^\s]+', full_text)
        return fallback.group(0) if fallback else None

 

    # Match URL to name and course name
def url_matching(url: str, studentName: str, courseName: str) -> bool:
    try:
        response = requests.get(url)
        html = response.text
        isNameMatch = studentName in html
        isCourseMatch = courseName in html

        return isNameMatch,isCourseMatch
    except Exception as e:
        logger.error(f"Error matching URL: {str(e)}")
        return False,False

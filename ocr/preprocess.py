from PIL import Image, ImageEnhance
import os
from dotenv import load_dotenv
load_dotenv()

def preprocess_image(image: Image.Image, upscale: int = 5, contrast: float = 1.2, threshold: int = 160) -> Image.Image:
    """
    Preprocess image for better OCR results
    Args:
        image: Input PIL Image
        upscale: Image upscaling factor
        contrast: Contrast enhancement factor
        threshold: Binarization threshold
    Returns:
        Preprocessed PIL Image
    """
    image = image.convert('L')  # Convert to grayscale
    
    if upscale > 1:
        image = image.resize((image.width * upscale, image.height * upscale), Image.LANCZOS)
    
    if contrast != 1.0:
        enhancer = ImageEnhance.Contrast(image)
        image = enhancer.enhance(contrast)
    
    image = image.point(lambda x: 0 if x < threshold else 255)  # Apply thresholding
    
    if os.getenv('MODE') == 'development':
        os.makedirs("img/preprocess", exist_ok=True)
        image.save("img/preprocess/preprocessed_image.jpg")
    
    return image

def crop_image(image: Image.Image, crop_percentage: float = 0.2) -> Image.Image:
    """
    Crop the bottom-left part of the image. The crop_percentage determines how much of the bottom is kept.
    """
    width, height = image.size
    crop_box = (0, int(height * (1 - crop_percentage)), int(width/1.5), height)
    cropped_image = image.crop(crop_box)
    
    if os.getenv('MODE') == 'development':
        # Save the cropped image for debugging
        os.makedirs("img/preprocess", exist_ok=True)
        cropped_image.save("img/preprocess/cropped_image.png")
    
    return cropped_image


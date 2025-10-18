from paddleocr import PaddleOCR
import numpy as np
from pathlib import Path
from extractors import pdf_to_images_with_fitz

p = Path('synth_pdfs/varied_001.pdf')
b = p.read_bytes()
imgs = pdf_to_images_with_fitz(b, max_pages=1)

paddle = PaddleOCR(use_textline_orientation=True, lang='th')
im_arr = np.array(imgs[0])
result = paddle.ocr(im_arr)

print(f'Result type: {type(result)}')
print(f'Result length: {len(result)}')
print(f'\nResult[0] type: {type(result[0])}')

ocr_result = result[0]

# Check if it has keys method
if hasattr(ocr_result, 'keys'):
    keys = list(ocr_result.keys())
    print(f'\nAvailable keys: {keys}')
    
    for key in keys:
        val = ocr_result[key]
        if isinstance(val, list) and len(val) > 0:
            print(f'\n{key}: list with {len(val)} items')
            print(f'  First item type: {type(val[0])}')
            if isinstance(val[0], str):
                print(f'  First 3 items: {val[:3]}')
                if key == 'rec_texts':
                    print(f'\n  All rec_texts:')
                    for i, txt in enumerate(val):
                        print(f'    [{i}]: {txt}')
        elif isinstance(val, dict):
            print(f'\n{key}: dict with keys {list(val.keys())[:5]}...')
        else:
            print(f'\n{key}: {type(val)} = {str(val)[:100]}')

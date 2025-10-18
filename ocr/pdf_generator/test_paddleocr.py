from pathlib import Path
import traceback
import numpy as np
from PIL import Image

from extractors import pdf_to_images_with_fitz

# Test PaddleOCR structure
try:
    from paddleocr import PaddleOCR
    
    p = Path('synth_pdfs/varied_001.pdf')
    print('pdf path:', p.resolve())
    print('exists:', p.exists())
    
    if not p.exists():
        raise SystemExit('PDF not found')
    
    b = p.read_bytes()
    imgs = pdf_to_images_with_fitz(b, max_pages=1)
    
    print(f'\nInitializing PaddleOCR...')
    paddle = PaddleOCR(use_angle_cls=True, lang='th')
    
    print(f'\nProcessing {len(imgs)} images...')
    for idx, img in enumerate(imgs):
        print(f'\n--- Image {idx+1} ---')
        im_arr = np.array(img)
        print(f'Image shape: {im_arr.shape}')
        
        # Call OCR
        result = paddle.ocr(im_arr)
        
        print(f'\nResult type: {type(result)}')
        print(f'Result length: {len(result) if result else 0}')
        
        if result:
            print(f'\nFirst element type: {type(result[0])}')
            print(f'First element: {result[0] is not None}')
            
            # Check if it's OCRResult object
            ocr_result = result[0]
            print(f'\nOCRResult attributes: {dir(ocr_result)}')
            
            # Try different ways to access the data
            if hasattr(ocr_result, 'boxes'):
                print(f'\nHas boxes attribute, type: {type(ocr_result.boxes)}')
            if hasattr(ocr_result, 'rec_text'):
                print(f'Has rec_text attribute, type: {type(ocr_result.rec_text)}')
                print(f'rec_text content: {ocr_result.rec_text}')
            if hasattr(ocr_result, 'rec_score'):
                print(f'Has rec_score attribute')
            if hasattr(ocr_result, '__iter__'):
                print(f'\nOCRResult is iterable, trying to iterate...')
                try:
                    texts = []
                    for idx, item in enumerate(ocr_result):
                        print(f'  Item {idx}: type={type(item)}, value={item}')
                        texts.append(str(item))
                        if idx >= 5:  # limit output
                            print(f'  ... ({len(ocr_result)} total items)')
                            break
                    print(f'\nJoined text from iteration: {" ".join(texts[:10])}')
                except Exception as e:
                    print(f'  Error iterating: {e}')
            
            # Try to get json representation
            if hasattr(ocr_result, 'json'):
                import json as json_module
                ocr_json = ocr_result.json
                print(f'\nOCRResult.json keys: {list(ocr_json.keys())}')
                if 'res' in ocr_json:
                    res = ocr_json['res']
                    print(f'res keys: {list(res.keys())}')
                    
                    # Look for text data
                    if 'rec_text' in res:
                        print(f'\nrec_text found: {res["rec_text"][:500] if res["rec_text"] else "empty"}')
                    if 'rec_score' in res:
                        print(f'rec_score found: {res["rec_score"][:10] if isinstance(res["rec_score"], list) else res["rec_score"]}')
                    
                    # Print first few items from json
                    print(f'\nFull JSON structure (formatted):')
                    print(json_module.dumps(ocr_json, indent=2, ensure_ascii=False)[:2000])
                    
            # Try accessing as dict
            print(f'\n\nTrying dict access:')
            if 'rec_text' in ocr_result:
                rec_texts = ocr_result['rec_text']
                print(f'rec_text via dict access: type={type(rec_texts)}, len={len(rec_texts) if hasattr(rec_texts, "__len__") else "N/A"}')
                if isinstance(rec_texts, list):
                    print(f'First 5 texts: {rec_texts[:5]}')
                    print(f'\nJoined text:\n{chr(10).join(rec_texts[:20])}')
        
except ImportError as e:
    print(f'PaddleOCR not installed: {e}')
except Exception as e:
    print(f'EXCEPTION: {e}')
    traceback.print_exc()

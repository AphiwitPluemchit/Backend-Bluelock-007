Switching OCR engine

This OCR helper supports two engines:

- tesseract (default)
- easyocr

Set which engine to use via the OCR_ENGINE environment variable (case-insensitive):

- OCR_ENGINE=tesseract (default)
- OCR_ENGINE=easyocr

We force EasyOCR to CPU-only mode (gpu=False) per project requirement. The existing Tesseract code remains unchanged and will be used when OCR_ENGINE is not `easyocr`.

Notes:

- `easyocr` is already included in `requirements.txt` in this folder. Install with:

```bash
pip install -r requirements.txt
```

- If you want to enable GPU later, modify `ocr/ocr.py` reader initialization to set `gpu=True`.

Examples

- Use default (tesseract):

```bash
OCR_ENGINE=tesseract python your_fastapi_or_script.py
```

- Use EasyOCR (CPU-only):

```bash
OCR_ENGINE=easyocr python your_fastapi_or_script.py
```

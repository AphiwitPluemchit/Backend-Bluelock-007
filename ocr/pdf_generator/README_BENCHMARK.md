Benchmarking PDF text extraction methods

This folder contains scripts to synthesize simple PDF certificates and benchmark different extraction methods: PyMuPDF (text layer), Tesseract OCR, EasyOCR and PaddleOCR.

Files added:

- `generate_pdfs.py`: synthesize PDFs with selectable text layer (Thai/EN names and courses).
- `extractors.py`: wrappers for PyMuPDF, pytesseract, easyocr and paddleocr.
- `metrics.py`: CER, WER, BLEU (with sacrebleu/nltk fallback), Levenshtein, and simple heuristics.
- `benchmark.py`: runner that generates PDFs (if missing), runs extractors and writes `benchmark_results.csv`.
- `requirements_benchmark.txt`: suggested pip packages.

How to run (Windows PowerShell):

1. Create and activate a virtualenv (optional):

```powershell
python -m venv .venv; .\.venv\Scripts\Activate.ps1
pip install -r requirements_benchmark.txt
```

2. Run benchmark (this will generate synthetic PDFs and run each extractor):

```powershell
python .\ocr\benchmark.py 10
```

Notes and recommended extra metrics:

- Character Error Rate (CER) and Word Error Rate (WER) are implemented.
- BLEU score (sentence) is included; sacrebleu is preferred for deterministic BLEU.
- Additional useful metrics:
  - Exact match (boolean) for name and course.
  - Normalized Levenshtein distance (1 - distance/max_len).
  - Precision/Recall on named-entity tokens (if you extract tokens like PERSON/COURSE).
  - Confidence-weighted metrics: average OCR confidence for matched tokens (EasyOCR and PaddleOCR expose confidences).
  - Token-level F1 after normalization (good for partial matches).
  - Layout-aware metrics: whether the name was found in expected bounding box (requires OCR boxes).

Limitations & next steps:

- The heuristic extractor for name/course is naive (largest line). For real certificates, implement layout rules (position, font-size) or use simple detection with coordinates.
- PaddleOCR and EasyOCR can return confidences and bounding boxes; incorporate them to compute more precise matching.
- For multilingual normalization (Thai), you may want to apply the project's `text_norm.py` functions before comparing.

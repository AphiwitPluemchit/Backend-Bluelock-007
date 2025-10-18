## Troubleshooting & quick examples

If PowerShell says a .bat file is not recognized, it usually means you're not running the command from the folder that contains the script (PowerShell doesn't search the current directory by default).

Simple ways to run the helper scripts:

- Change into the `pdf_generator` folder and run the short wrapper:

```powershell
Set-Location .\pdf_generator
.\setup_benchmark.bat
.\bench.bat --limit 10
```

- Or run using a relative path from the repo root:

```powershell
.\ocr\pdf_generator\setup_benchmark.bat
.\ocr\pdf_generator\bench.bat --limit 10
```

- If you prefer PowerShell script (`bench.ps1`) you might need to relax the execution policy for the current session:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\ocr\pdf_generator\bench.ps1 --limit 10
```

If the setup script cannot find a Python in `.venv`, it will fall back to the system `python` executable. Make sure your virtualenv is activated or that `python` on PATH points to the environment you want.

If you still have issues, list files in the folder to verify the script exists:

```powershell
Get-ChildItem -Path .\ocr\pdf_generator
```

If you want, I can also add a small top-level Makefile or a PowerShell profile helper to make `bench` available from repo root without typing the path. Tell me which you prefer and I will add it.
For multilingual normalization (Thai), you may want to apply the project's `text_norm.py` functions before comparing.
Benchmarking PDF text extraction methods

This folder contains scripts to synthesize simple PDF certificates and benchmark different extraction methods: PyMuPDF (text layer), Tesseract OCR, EasyOCR and PaddleOCR.

Files included:

**Core Scripts:**

- `benchmark.py`: Main runner with Thai tokenization support and smart label matching
- `generate_pdfs.py`: Synthesize PDFs with selectable text layer (Thai/EN names and courses)
- `extractors.py`: Wrappers for PyMuPDF, pytesseract, easyocr and paddleocr
- `metrics.py`: CER, WER, BLEU with normalization and heuristic text extraction

**Setup & Convenience Scripts:**

- `setup_benchmark.bat`: One-time dependency installation
- `bench.bat` / `bench.ps1`: Short command wrapper for easy usage
- `run_benchmark_quick.bat`: Quick 10-PDF test
- `run_benchmark.bat`: Full benchmark run

**Data Files:**

- `requirements_benchmark.txt`: Python dependencies (includes pythainlp for Thai)
- `labels_varied.csv`: Ground-truth labels for name/course extraction

How to run:

## Quick Setup & Usage

### 1. One-time Setup

```bash
# Run setup script to install all dependencies
setup_benchmark.bat
```

### 2. Quick Commands (Recommended)

```bash
# Quick test (10 PDFs) - Fast!
bench --limit 10

# Custom number of PDFs
bench --limit 5     # 5 PDFs only
bench --limit 20    # 20 PDFs

# Full benchmark (all PDFs)
bench -n 0

# With custom labels file
bench -l my_labels.csv --limit 15
```

### 3. Alternative: Pre-built Scripts

```bash
# Quick test (10 PDFs)
run_benchmark_quick.bat

# Full benchmark (all available PDFs)
run_benchmark.bat
```

## Manual Setup (Advanced)

If you prefer manual setup:

```powershell
# Create virtualenv (if needed)
python -m venv .venv
.\.venv\Scripts\Activate.ps1

# Install dependencies
pip install -r requirements_benchmark.txt

# Run benchmark manually
python benchmark.py --limit 10
```

## Command Options

- `--limit N` : Process only first N PDFs (for quick testing)
- `-n N` : Generate N synthetic PDFs if none exist
- `-l path` : Use custom labels CSV file (auto-detects `labels_varied.csv`)

## Output

Results are saved to `benchmark_results.csv` with columns:

- `pdf`, `method`, `time_s`
- `ref_name`, `hyp_name`, `name_cer`, `name_wer`, `name_bleu`, `name_exact`
- `ref_course`, `hyp_course`, `course_cer`, `course_wer`, `course_bleu`, `course_exact`

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

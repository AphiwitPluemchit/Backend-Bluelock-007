# OCR Benchmark Results Summary

This directory contains tools for benchmarking OCR methods and analyzing their performance.

## Quick Start

### Run Benchmark

```bash
# Run on 2 PDFs (quick test)
python benchmark.py --limit 2

# Run on all PDFs
python benchmark.py

# Run with debug output
python benchmark.py --limit 2 --debug
```

### View Summary

```bash
# Quick compact summary (recommended)
python quick_summary.py

# Detailed summary with statistics
python summary_metrics.py

# Check specific results
python check_results.py
```

## Files

- **`benchmark.py`**: Main benchmark script - runs OCR on PDFs and saves results
- **`quick_summary.py`**: Shows compact summary table with averages
- **`summary_metrics.py`**: Shows detailed statistics and rankings
- **`check_results.py`**: Shows individual PDF results with PaddleOCR details
- **`benchmark_results.csv`**: Raw benchmark data (all methods × all PDFs)
- **`benchmark_summary.csv`**: Average metrics per method

## OCR Methods Compared

1. **PyMuPDF** - Extracts embedded text from PDFs (fastest, most accurate if text layer exists)
2. **Tesseract** - Traditional OCR engine (balanced speed/quality)
3. **EasyOCR** - Deep learning OCR (best accuracy for Thai text)
4. **PaddleOCR** - Deep learning OCR (good speed/quality balance)

## Metrics Explained

- **CER (Character Error Rate)**: Lower is better (0.0 = perfect)
- **WER (Word Error Rate)**: Lower is better (0.0 = perfect)
- **BLEU Score**: Higher is better (1.0 = perfect)
- **Time**: Seconds to process one page
- **Exact Match %**: Percentage of PDFs with 100% correct extraction

## Recent Results Summary

Based on current benchmark (2 PDFs):

| Method    | Avg CER | Avg WER | Avg BLEU | Avg Time | Recommendation                  |
| --------- | ------- | ------- | -------- | -------- | ------------------------------- |
| PyMuPDF   | 0.000   | 0.000   | 1.000    | 0.003s   | ⭐ Best if PDF has text layer   |
| EasyOCR   | 0.186   | 0.640   | 0.420    | 37.9s    | ⭐ Best for OCR (most accurate) |
| Tesseract | 0.254   | 0.523   | 0.590    | 0.965s   | ⚡ Fast alternative             |
| PaddleOCR | 0.328   | 0.380   | 0.570    | 20.9s    | 🔄 Good balance                 |

## Key Improvements Made

### PaddleOCR Fixes

1. ✅ Updated to use new API (`use_textline_orientation` instead of `use_angle_cls`)
2. ✅ Fixed result format handling (OCRResult object vs old list format)
3. ✅ Implemented line grouping using bounding boxes to properly combine words into lines
4. ✅ Template text now properly filtered out (header/footer removal works correctly)

**Result**: CER improved from 0.620 → 0.240 (61% better) on varied_001.pdf

### EasyOCR Fixes

1. ✅ Changed from space-joining to newline-joining to preserve line structure
2. ✅ Template text filtering now works correctly

**Result**: Body text extraction now works (was completely empty before)

## Usage Examples

### Run quick benchmark and see summary

```bash
python benchmark.py --limit 2
# Automatically shows summary at the end
```

### Run full benchmark on all PDFs

```bash
python benchmark.py
```

### View just the summary from existing results

```bash
python quick_summary.py
```

### Generate varied test PDFs

```bash
python generate_pdfs.py
```

## Troubleshooting

### Benchmark shows no summary

```bash
# Run summary manually
python summary_metrics.py
```

### Want to skip auto-summary

```bash
python benchmark.py --limit 2 --no-summary
```

### See detailed per-PDF results

```bash
python check_results.py
```

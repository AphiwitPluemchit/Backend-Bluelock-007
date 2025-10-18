# Dataset Generation Guide

## Overview

This document explains how to generate and use the varied PDF certificate dataset for OCR benchmarking.

## Dataset Statistics

- **Total PDFs**: 500 (configurable)
- **Unique Thai Names**: 50 base names + variations = 122 unique
- **Unique English Names**: 50 base names + variations = 122 unique
- **Unique Courses**: 20 in Thai + 20 in English
- **Variations**:
  - Long names (every 15th record)
  - Batch numbers (every 7th record)
  - Different honorifics (ดร., ผศ., รศ., นพ., พญ., etc.)

## Quick Start

### Generate Test Dataset (10 PDFs)

```bash
python create_varied_labels_and_pdfs.py --quick-test
```

### Generate Full Dataset (500 PDFs)

```bash
python create_varied_labels_and_pdfs.py -n 500
```

### Custom Number of PDFs

```bash
python create_varied_labels_and_pdfs.py -n 100  # Generate 100 PDFs
```

### Verify Generated Dataset

```bash
python verify_dataset.py
```

## Running Benchmarks

### Quick Benchmark (First 10 PDFs)

```bash
python benchmark.py --limit 10
```

### Medium Benchmark (First 50 PDFs)

```bash
python benchmark.py --limit 50
```

### Full Benchmark (All PDFs)

```bash
python benchmark.py
```

**Note**: Full benchmark on 500 PDFs will take a long time (estimated 5+ hours for EasyOCR)

### View Summary After Benchmark

```bash
python summary_metrics.py
python quick_summary.py
python comparison_table.py
```

## File Structure

```
pdf_generator/
├── create_varied_labels_and_pdfs.py  # Main generator script
├── verify_dataset.py                 # Dataset verification
├── labels_varied.csv                 # Generated labels
├── synth_pdfs/                       # Generated PDF files
│   ├── varied_001.pdf
│   ├── varied_002.pdf
│   └── ... (500 total)
└── benchmark_results.csv             # Benchmark output
```

## Name Variations

### Thai Names (50 base + variations)

- Standard names: "นาย สมชาย ใจดี", "นางสาว สมศรี ตัวอย่าง"
- Academic titles: "ดร. วิทยา พิทักษ์", "ผศ.ดร. วีระชัย ศรีสุข"
- Medical titles: "นพ. ภูมิพัฒน์ เมธี", "พญ. อรทัย สุขเกษม"
- Long names: "นาย สมชาย พิพัฒน์สมบูรณ์กิจ"
- Batch variations: "นาย สมชาย ใจดี (รุ่น 1)"

### English Names (50 base + variations)

- Standard names: "Somchai Jaidee", "Somsri Example"
- Academic titles: "Dr. Witya Pitak", "Assoc. Prof. Weerachai Srisuk"
- Long names: "Somchai Phipatsomboonkij"
- Batch variations: "Somchai Jaidee (Batch 1)"

### Courses (20 per language)

Thai courses include:

- การพัฒนาซอฟต์แวร์เชิงปฏิบัติ (32 ชั่วโมง)
- การออกแบบ UX/UI สำหรับผู้เริ่มต้น (16 ชั่วโมง)
- ภาษาไทยสำหรับวิชาชีพ (12 ชั่วโมง)
- และอื่นๆ อีก 17 รายการ

English courses include:

- Practical Software Development (32h)
- Intro to UX/UI Design (16h)
- Thai for Professionals (12h)
- และอื่นๆ อีก 17 รายการ

## Expected Benchmark Performance

Based on previous tests:

| Method    | Avg CER | Avg Speed | Best For                   |
| --------- | ------- | --------- | -------------------------- |
| PyMuPDF   | 0.000   | 0.003s    | Native PDF text extraction |
| EasyOCR   | 0.186   | 37.9s     | OCR with best accuracy     |
| Tesseract | 0.254   | 0.96s     | Balanced speed/accuracy    |
| PaddleOCR | 0.328   | 20.9s     | Alternative OCR method     |

**Note**: These metrics are from a 2-PDF test. Results may vary with 500 PDFs.

## Tips

1. **Start Small**: Use `--quick-test` (10 PDFs) first to verify setup
2. **Medium Test**: Use `--limit 50` for initial benchmark
3. **Full Test**: Run full 500 PDFs overnight or during breaks
4. **Monitor Progress**: Benchmark script shows progress bar
5. **Check Results**: Use summary scripts to analyze metrics

## Troubleshooting

### PDFs Not Generating

- Check if `generate_pdfs.py` exists
- Verify reportlab is installed: `pip install reportlab`
- Check write permissions in `synth_pdfs/` directory

### Benchmark Taking Too Long

- Use `--limit` flag to test smaller subset
- Skip EasyOCR initially (slowest method)
- Run overnight for full 500 PDF benchmark

### Memory Issues

- Close other applications
- Run benchmark in smaller batches
- Increase virtual memory if needed

## Next Steps

After generating 500 PDFs:

1. Run quick benchmark: `python benchmark.py --limit 10`
2. Review initial results: `python quick_summary.py`
3. If results look good, run full benchmark: `python benchmark.py`
4. Analyze comprehensive metrics: `python summary_metrics.py`
5. Compare methods: `python comparison_table.py`

---

**Last Updated**: January 2025
**Dataset Version**: 2.0 (500 PDFs, 50 names, 20 courses)

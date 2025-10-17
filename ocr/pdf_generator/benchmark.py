import csv
import sys
from pathlib import Path
from generate_pdfs import OUT_DIR, synthesize_pdfs
from extractors import extract_all
from metrics import heuristic_extract_name_course, summarize_metrics


PDF_DIR = OUT_DIR
RESULTS = Path(__file__).parent / "benchmark_results.csv"


def run_benchmark(num_pdfs: int = 10):
    # Ensure PDFs exist
    if not PDF_DIR.exists() or not any(PDF_DIR.iterdir()):
        print("Generating synthetic PDFs...")
        synthesize_pdfs([], [], [], [], n=num_pdfs)

    rows = []
    for pdf in sorted(PDF_DIR.glob("synth_*.pdf")):
        print("Processing:", pdf)
        pdf_bytes = pdf.read_bytes()
        extracted = extract_all(pdf_bytes)
        # We need reference text: open the PDF with PyMuPDF text layer for ground truth
        ref_text = extracted.get('pymupdf', ("", 0.0))[0]
        ref_name, ref_course = heuristic_extract_name_course(ref_text)
        for method, (text, duration) in extracted.items():
            hyp_name, hyp_course = heuristic_extract_name_course(text)
            nm = summarize_metrics(ref_name, hyp_name)
            cm = summarize_metrics(ref_course, hyp_course)
            rows.append({
                'pdf': pdf.name,
                'method': method,
                'time_s': round(duration, 3),
                'ref_name': ref_name,
                'hyp_name': hyp_name,
                'name_cer': nm['cer'],
                'name_wer': nm['wer'],
                'name_bleu': nm['bleu'],
                'name_exact': nm['exact'],
                'ref_course': ref_course,
                'hyp_course': hyp_course,
                'course_cer': cm['cer'],
                'course_wer': cm['wer'],
                'course_bleu': cm['bleu'],
                'course_exact': cm['exact'],
            })

    # write CSV
    with open(RESULTS, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=list(rows[0].keys()) if rows else [])
        writer.writeheader()
        for r in rows:
            writer.writerow(r)

    print(f"Wrote results to: {RESULTS}")


if __name__ == '__main__':
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    run_benchmark(n)

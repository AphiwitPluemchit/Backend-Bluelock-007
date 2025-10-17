import argparse
import csv
from pathlib import Path
from typing import Optional
from generate_pdfs import OUT_DIR, synthesize_pdfs
from extractors import extract_all
from metrics import heuristic_extract_name_course, summarize_metrics, cer, wer


PDF_DIR = OUT_DIR
RESULTS = Path(__file__).parent / "benchmark_results.csv"


def load_labels_csv(path: Path):
    mapping = {}
    if not path or not path.exists():
        return mapping
    with open(path, newline='', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for r in reader:
            # accept multiple possible column names for filename, name and course
            key = r.get('pdf_filename') or r.get('pdf') or r.get('filename') or r.get('file')
            if not key:
                print("Warning: CSV row missing pdf filename column; skipping row")
                continue
            # prefer Thai name/course columns but fall back to generic ones
            name = (r.get('name_th') or r.get('name') or r.get('name_en') or '').strip()
            course = (r.get('course_th') or r.get('course') or r.get('course_en') or '').strip()
            mapping[key] = (name, course)
    return mapping


def _validate_labels_map(mapping: dict) -> bool:
    """Return True if mapping looks valid (non-empty values for at least some entries)."""
    if not mapping:
        return False
    # check at least one entry has non-empty name or course
    for k, (n, c) in mapping.items():
        if (n or c):
            return True
    return False


def run_benchmark(num_pdfs: int = 10, labels_csv: Optional[Path] = None):
    """Run benchmark over PDFs.

    Behavior:
    - If `labels_csv` is provided and contains mappings, use it as the ground-truth (ref_name/ref_course).
    - Otherwise, use the PyMuPDF text-layer (`pymupdf`) as the reference and compare other methods to it.
    - Always use `heuristic_extract_name_course` only to parse hypothesis/reference strings into (name, course).
    - Generate synthetic PDFs if none are present.
    """
    # Ensure PDFs exist (generate synthetic if necessary)
    if not PDF_DIR.exists() or not any(PDF_DIR.iterdir()):
        print("Generating synthetic PDFs...")
        synthesize_pdfs([], [], [], [], n=num_pdfs)

    labels_map = load_labels_csv(labels_csv) if labels_csv is not None else {}

    # Build list of PDF paths to process: prefer filenames from labels_map (preserve CSV order)
    if labels_map:
        pdfs_to_process = []
        for fname in labels_map.keys():
            p = PDF_DIR / fname
            if p.exists():
                pdfs_to_process.append(p)
            else:
                print(f"Warning: labeled file listed in CSV not found: {p}")
        if not pdfs_to_process:
            print("No labeled PDF files found in PDF_DIR; falling back to globbing synth_*.pdf")
            pdfs_to_process = sorted(PDF_DIR.glob("synth_*.pdf"))
    else:
        pdfs_to_process = sorted(PDF_DIR.glob("synth_*.pdf"))

    rows = []

    for pdf in pdfs_to_process:
        print("Processing:", pdf)
        pdf_bytes = pdf.read_bytes()
        extracted = extract_all(pdf_bytes)

        # Determine reference: prefer labels_map (csv) if available, else PyMuPDF text-layer
        if labels_map and pdf.name in labels_map:
            ref_name, ref_course = labels_map[pdf.name]
        else:
            ref_text = extracted.get('pymupdf', ("", 0.0))[0]
            ref_name, ref_course = heuristic_extract_name_course(ref_text)

        def _best_line_match(ref: str, text: str) -> str:
            """Return the best-matching line from text for the provided ref string using CER (lower is better).

            If ref is empty or no lines, return empty string.
            """
            if not ref:
                return ""
            lines = [l.strip() for l in (text or "").splitlines() if l.strip()]
            if not lines:
                return ""
            best_line = ""
            best_score = float('inf')
            best_wer = float('inf')
            for l in lines:
                try:
                    sc = cer(ref, l)
                except Exception:
                    sc = 1.0
                if sc < best_score:
                    best_score = sc
                    # store wer as tie-breaker
                    try:
                        best_wer = wer(ref, l)
                    except Exception:
                        best_wer = 1.0
                    best_line = l
                elif sc == best_score:
                    # tie-breaker: choose lower WER
                    try:
                        w = wer(ref, l)
                    except Exception:
                        w = 1.0
                    if w < best_wer:
                        best_wer = w
                        best_line = l
            return best_line

        for method, (text, duration) in extracted.items():
            # If reference label exists, try to find best matching lines for name/course in the OCR text
            if labels_map and pdf.name in labels_map and (ref_name or ref_course):
                hyp_name = _best_line_match(ref_name, text) if ref_name else ""
                hyp_course = _best_line_match(ref_course, text) if ref_course else ""
                # if best-match failed, fallback to heuristic
                if not hyp_name and not hyp_course:
                    hyp_name, hyp_course = heuristic_extract_name_course(text)
                else:
                    # if one missing, attempt heuristic for the missing one
                    if not hyp_name:
                        hyp_name = heuristic_extract_name_course(text)[0]
                    if not hyp_course:
                        hyp_course = heuristic_extract_name_course(text)[1]
            else:
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

    # write CSV if we have results
    if rows:
        with open(RESULTS, 'w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
            writer.writeheader()
            writer.writerows(rows)
        print(f"Wrote results to: {RESULTS}")
    else:
        print("No results to write.")


def _parse_path(p: Optional[str]) -> Optional[Path]:
    if not p:
        return None
    pth = Path(p)
    return pth if pth.exists() else None


def main():
    parser = argparse.ArgumentParser(description="Run OCR benchmark")
    parser.add_argument('-n', '--num', type=int, default=10, help='number of synthetic PDFs to generate if missing')
    parser.add_argument('-l', '--labels', type=str, default=None, help='path to labels CSV (uses csv order as reference)')
    args = parser.parse_args()

    labels_path = _parse_path(args.labels) if args.labels else None
    labels_map = load_labels_csv(labels_path) if labels_path else {}
    if labels_path and not _validate_labels_map(labels_map):
        print(f"Warning: labels CSV provided but no usable name/course columns found: {labels_path}")

    run_benchmark(num_pdfs=args.num, labels_csv=labels_path)


if __name__ == '__main__':
    main()

import argparse
import csv
from pathlib import Path
from typing import Optional
from generate_pdfs import OUT_DIR, synthesize_pdfs
from extractors import extract_all
from metrics import heuristic_extract_name_course, summarize_metrics, cer, wer, bleu_score
try:
    from pythainlp.tokenize import word_tokenize as _thai_tokenize
except Exception:
    _thai_tokenize = None
import re


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


def run_benchmark(num_pdfs: int = 10, labels_csv: Optional[Path] = None, limit_pdfs: Optional[int] = None):
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

    # Apply limit if specified
    if limit_pdfs and limit_pdfs > 0:
        pdfs_to_process = pdfs_to_process[:limit_pdfs]
        print(f"Limited to first {len(pdfs_to_process)} PDFs")

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

        def _normalize_for_match(s: str) -> str:
            if not s:
                return ""
            s = s.strip()
            s = re.sub(r'\s+', ' ', s)
            s = re.sub(r'[“”"\'•·,;:!?()\[\]{}<>]', '', s)
            # strip common honorifics/titles to improve matching
            s = re.sub(r'\b(ดร|ผศ|ผศดร|นพ|นาย|นางสาว|นาง|เด็กชาย|เด็กหญิง|Dr|Assoc|Prof)\b', '', s, flags=re.IGNORECASE)
            return s.lower()


        def _best_line_match(ref: str, text: str) -> str:
            """Return the best-matching line from text for the provided ref string.

            Strategy:
            - normalize ref and candidate lines (strip honorifics/punctuation, lowercase)
            - fast-path: normalized substring match
            - fallback: choose line with smallest CER (WER tie-breaker)
            """
            if not ref:
                return ""
            ref_n = _normalize_for_match(ref)
            lines = [l.strip() for l in (text or "").splitlines() if l.strip()]
            if not lines:
                return ""

            # fast substring match
            for l in lines:
                if ref_n in _normalize_for_match(l):
                    return l

            best_line = ""
            best_score = float('inf')
            best_wer = float('inf')
            for l in lines:
                cand_n = _normalize_for_match(l)
                try:
                    sc = cer(ref_n, cand_n)
                except Exception:
                    sc = 1.0
                if sc < best_score:
                    best_score = sc
                    try:
                        best_wer = wer(ref_n, cand_n)
                    except Exception:
                        best_wer = 1.0
                    best_line = l
                elif sc == best_score:
                    try:
                        w = wer(ref_n, cand_n)
                    except Exception:
                        w = 1.0
                    if w < best_wer:
                        best_wer = w
                        best_line = l
            return best_line

        def _tokenize_for_metric(s: str) -> str:
            """Return a space-joined token string suitable for WER/BLEU computation.

            If Thai tokenizer available and text contains Thai characters, use it.
            Otherwise fall back to whitespace split.
            """
            if not s:
                return ""
            # use normalized form for tokenization
            s_n = _normalize_for_match(s)
            # detect Thai characters
            has_thai = any('\u0e00' <= ch <= '\u0e7f' for ch in s)
            if has_thai and _thai_tokenize:
                try:
                    toks = _thai_tokenize(s)
                    return ' '.join(toks)
                except Exception:
                    pass
            # fallback: split on whitespace
            return ' '.join(s_n.split())

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
            # compute metrics: CER on normalized raw strings; WER/BLEU on tokenized strings
            ref_name_norm = _normalize_for_match(ref_name) if ref_name else ""
            hyp_name_norm = _normalize_for_match(hyp_name) if hyp_name else ""
            ref_course_norm = _normalize_for_match(ref_course) if ref_course else ""
            hyp_course_norm = _normalize_for_match(hyp_course) if hyp_course else ""

            # tokenized forms for WER/BLEU
            ref_name_tok = _tokenize_for_metric(ref_name)
            hyp_name_tok = _tokenize_for_metric(hyp_name)
            ref_course_tok = _tokenize_for_metric(ref_course)
            hyp_course_tok = _tokenize_for_metric(hyp_course)

            name_cer = cer(ref_name_norm, hyp_name_norm)
            try:
                name_wer = wer(ref_name_tok, hyp_name_tok)
            except Exception:
                name_wer = 1.0
            try:
                name_bleu = bleu_score(ref_name_tok, hyp_name_tok)
            except Exception:
                name_bleu = 0.0
            name_exact = (ref_name_norm == hyp_name_norm)

            course_cer = cer(ref_course_norm, hyp_course_norm)
            try:
                course_wer = wer(ref_course_tok, hyp_course_tok)
            except Exception:
                course_wer = 1.0
            try:
                course_bleu = bleu_score(ref_course_tok, hyp_course_tok)
            except Exception:
                course_bleu = 0.0
            course_exact = (ref_course_norm == hyp_course_norm)

            rows.append({
                'pdf': pdf.name,
                'method': method,
                'time_s': round(duration, 3),
                'ref_name': ref_name,
                'hyp_name': hyp_name,
                'name_cer': name_cer,
                'name_wer': name_wer,
                'name_bleu': name_bleu,
                'name_exact': name_exact,
                'ref_course': ref_course,
                'hyp_course': hyp_course,
                'course_cer': course_cer,
                'course_wer': course_wer,
                'course_bleu': course_bleu,
                'course_exact': course_exact,
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
    parser.add_argument('--limit', type=int, default=None, help='limit number of PDFs to process (for quick testing)')
    args = parser.parse_args()

    # Auto-detect labels_varied.csv if no labels path specified
    labels_path = None
    if args.labels:
        labels_path = _parse_path(args.labels)
    else:
        # Try to find labels_varied.csv in current directory
        auto_labels = Path(__file__).parent / "labels_varied.csv"
        if auto_labels.exists():
            labels_path = auto_labels
            print(f"Auto-detected labels file: {auto_labels}")

    labels_map = load_labels_csv(labels_path) if labels_path else {}
    if labels_path and not _validate_labels_map(labels_map):
        print(f"Warning: labels CSV provided but no usable name/course columns found: {labels_path}")

    run_benchmark(num_pdfs=args.num, labels_csv=labels_path, limit_pdfs=args.limit)


if __name__ == '__main__':
    main()

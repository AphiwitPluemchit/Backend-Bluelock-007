import argparse
import csv
from pathlib import Path
from typing import Optional
from generate_pdfs import OUT_DIR, synthesize_pdfs
from extractors import extract_all
from metrics import cer, wer, bleu_score
try:
    from pythainlp.tokenize import word_tokenize as _thai_tokenize
except Exception:
    _thai_tokenize = None
import re


PDF_DIR = OUT_DIR
RESULTS = Path(__file__).parent / "benchmark_results.csv"

# Default number of decimal places for numeric metrics (can be overridden by
# passing `decimals` to run_benchmark or by editing this constant).
METRIC_DECIMALS = 3


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
            # prefer full_text column; labels should contain only dynamic body text
            full_text = (r.get('full_text') or r.get('ref_full_text') or '').strip()
            include_template = True if (r.get('include_template') or '').strip().lower() in ('1', 'true', 'yes', 'y') else False
            # If the labels CSV includes the full_text and include_template is True,
            # provide header_text/footer_text entries so the benchmark can remove
            # template header/footer lines before scoring. TEMPLATE_LINES is defined
            # later in the file but is available at runtime when this function is used.
            # Clean the provided full_text: remove any template/header/footer
            # lines so label entries contain only the dynamic body text.
            header_text = ''
            footer_text = ''
            try:
                def _strip_template_blocks(s: str) -> str:
                    if not s:
                        return ''
                    lines = [ln.strip() for ln in s.splitlines() if ln.strip()]
                    # remove any exact matches to TEMPLATE_LINES/HEADER_LINES/FOOTER_LINES
                    filtered = [ln for ln in lines if ln not in (TEMPLATE_LINES + HEADER_LINES + FOOTER_LINES)]
                    return "\n".join(filtered)

                # Always attempt to remove template/header/footer from CSV-provided full_text
                full_text = _strip_template_blocks(full_text)

                if include_template:
                    # expose header/footer labels for downstream stripping logic
                    header_text = "\n".join([t for t in TEMPLATE_LINES if t])
                    tpl_tail = [t for t in TEMPLATE_LINES if t][-2:]
                    footer_text = "\n".join(tpl_tail) if tpl_tail else ''
            except Exception:
                # conservative fallback: leave as-is
                pass

            mapping[key] = {
                'full_text': full_text,
                'include_template': include_template,
                'header_text': header_text,
                'footer_text': footer_text,
            }
    return mapping


def _validate_labels_map(mapping: dict) -> bool:
    """Return True if mapping looks valid (non-empty values for at least some entries).

    Mapping now stores dicts with keys: name, course, full_text, include_template.
    """
    if not mapping:
        return False
    for k, v in mapping.items():
        if not isinstance(v, dict):
            continue
        if v.get('full_text'):
            return True
    return False


# Template lines used for canonicalization (if include_template requested)
TEMPLATE_LINES = [
    "CERTIFICATE OF COMPLETION",
    "THAI MOOC",
    "Thailand Massive Open Online Courses",
    "THIS CERTIFICATE IS AWARDED TO",
    "",
    "Assoc.Prof.Dr. Thapanee Thammeter",
    "Director of Thailand Cyber University Project (TCU)",
]

# Explicit header/footer constants used when labels do not provide them.
HEADER_LINES = [
    "CERTIFICATE OF COMPLETION",
    "THAI MOOC",
    "Thailand Massive Open Online Courses",
    "THIS CERTIFICATE IS AWARDED TO",
]

FOOTER_LINES = [
    "Assoc.Prof.Dr. Thapanee Thammeter",
    "Director of Thailand Cyber University Project (TCU)",
]


def canonicalize_doc_text(text: str, include_template: bool = True) -> str:
    """Canonicalize multi-line document text into header -> body -> footer order.

    - Splits lines, trims and removes empty lines
    - Classifies lines into header/body/footer by keyword heuristics
    - If include_template True, ensure template lines are present at the top
    """
    if not text:
        return ""
    lines = [l.strip() for l in text.splitlines() if l.strip()]
    header_keys = ("CERTIFICATE", "THAI MOOC", "THAILAND MASSIVE OPEN ONLINE COURSES")
    footer_keys = ("ASSOC.PROF", "THAPANEE", "DIRECTOR", "TCU")
    headers, bodies, footers = [], [], []
    for l in lines:
        u = l.upper()
        if any(k in u for k in header_keys):
            headers.append(l)
        elif any(k in u for k in footer_keys):
            footers.append(l)
        else:
            bodies.append(l)

    ordered = []
    if include_template:
        for t in TEMPLATE_LINES:
            if t and t not in ordered:
                ordered.append(t)
    ordered += headers + bodies + footers

    # remove duplicates while preserving order
    seen = set()
    uniq = []
    for l in ordered:
        if l and l not in seen:
            seen.add(l)
            uniq.append(l)
    return "\n".join(uniq)


def split_doc_sections(text: str, include_template: bool = True):
    """Return (header, body, footer) strings from the provided document text.

    Uses same heuristics as canonicalize_doc_text. If include_template True,
    template lines are prepended to the header.
    """
    if not text:
        return "", "", ""
    lines = [l.strip() for l in text.splitlines() if l.strip()]
    header_keys = ("CERTIFICATE", "THAI MOOC", "THAILAND MASSIVE OPEN ONLINE COURSES")
    footer_keys = ("ASSOC.PROF", "THAPANEE", "DIRECTOR", "TCU")
    headers, bodies, footers = [], [], []
    for l in lines:
        u = l.upper()
        if any(k in u for k in header_keys):
            headers.append(l)
        elif any(k in u for k in footer_keys):
            footers.append(l)
        else:
            bodies.append(l)

    if include_template:
        # ensure template lines are included in header
        tpl = [t for t in TEMPLATE_LINES if t]
        # avoid duplicate appends
        for t in tpl:
            if t not in headers:
                headers.insert(0, t)

    header_text = "\n".join(headers)
    body_text = "\n".join(bodies)
    footer_text = "\n".join(footers)
    return header_text, body_text, footer_text


def run_benchmark(num_pdfs: int = 10, labels_csv: Optional[Path] = None, limit_pdfs: Optional[int] = None, decimals: Optional[int] = None, debug: bool = False, debug_limit: int = 2):
    """Run benchmark over PDFs.

    Behavior:
    - If `labels_csv` is provided and contains mappings, use it as the ground-truth (full document text for the body).
    - Otherwise, use the PyMuPDF text-layer (`pymupdf`) as the reference and compare other methods to it.
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

    dbg_count = 0
    for pdf in pdfs_to_process:
        print("Processing:", pdf)
        pdf_bytes = pdf.read_bytes()
        extracted = extract_all(pdf_bytes)

        # Determine reference: prefer labels_map (csv) if available, else PyMuPDF text-layer
        if labels_map and pdf.name in labels_map:
            lm = labels_map[pdf.name]
            # mapping entries are dicts with keys: full_text, include_template
            ref_full_text = lm.get('full_text') or ""
            full_include_template = bool(lm.get('include_template'))
        else:
            ref_full_text = extracted.get('pymupdf', ("", 0.0))[0] or ""
            full_include_template = False

        def _normalize_for_match(s: str) -> str:
            if not s:
                return ""
            s = s.strip()
            s = re.sub(r'\s+', ' ', s)
            s = re.sub(r'[“”"\'•·,;:!?()\[\]{}<>]', '', s)
            # strip common honorifics/titles to improve matching
            s = re.sub(r'\b(ดร|ผศ|ผศดร|นพ|นาย|นางสาว|นาง|เด็กชาย|เด็กหญิง|Dr|Assoc|Prof)\b', '', s, flags=re.IGNORECASE)
            return s.lower()


        # _best_line_match removed: not needed for body-only scoring

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
            # determine decimals to use for this run
            _dec = METRIC_DECIMALS if decimals is None else int(decimals)

            def _format_num(v):
                try:
                    if isinstance(v, float):
                        return round(v, _dec)
                    # attempt cast to float then round
                    fv = float(v)
                    return round(fv, _dec)
                except Exception:
                    return v
            # We treat the canonicalized body as the primary unit for scoring.
            # Determine full reference text (labels CSV preferred) then split to sections.
            # Use the reference full text determined earlier (labels preferred)
            full_ref_text = ref_full_text if 'ref_full_text' in locals() else ""
            if not full_ref_text:
                full_ref_text = extracted.get('pymupdf', ("", 0.0))[0] or ""

            full_hyp_text = text or ""

            # If debug mode requested, capture pre-strip snippets for the first
            # `debug_limit` PDFs to help troubleshooting.
            if debug and dbg_count < debug_limit:
                pre_strip_ref = (full_ref_text or '')[:1000]
                pre_strip_hyp = (full_hyp_text or '')[:1000]
            else:
                pre_strip_ref = pre_strip_hyp = None

            # canonicalize both (document-level)
            # If labels CSV provides explicit header/footer text, remove those
            # lines from both reference and hypothesis before computing body metrics.
            # We don't expose removed_header/removed_footer flags in the CSV to keep
            # output compact — benchmark simply ensures the body used for metrics
            # excludes header/footer when possible.
            # Prepare header/footer labels: prefer per-file labels from CSV, else fall
            # back to global constants. Use normalized substring matching to be robust
            # against small differences (punctuation/whitespace/case).
            lm = labels_map.get(pdf.name, {}) if labels_map else {}
            header_label_text = (lm.get('header') or lm.get('header_lines') or lm.get('header_text') or '')
            footer_label_text = (lm.get('footer') or lm.get('footer_lines') or lm.get('footer_text') or '')
            if not header_label_text:
                header_label_text = "\n".join(HEADER_LINES)
            if not footer_label_text:
                footer_label_text = "\n".join(FOOTER_LINES)

            def _normalize_for_substr(s: str) -> str:
                # reuse normalization but be a bit stricter for matching: remove
                # punctuation and collapse spaces
                if not s:
                    return ""
                s2 = _normalize_for_match(s)
                s2 = re.sub(r'[\p{P}\p{S}]', '', s2) if False else s2
                return re.sub(r'\s+', ' ', s2).strip()

            def _remove_label_lines_normalized(src_text: str, label_text: str):
                if not src_text or not label_text:
                    return src_text, False
                labs = [l for l in [ln.strip() for ln in label_text.splitlines()] if l]
                if not labs:
                    return src_text, False
                norm_labs = [_normalize_for_substr(l) for l in labs]
                out_lines = []
                removed_any = False
                for line in src_text.splitlines():
                    norm_line = _normalize_for_substr(line)
                    keep = True
                    for nl in norm_labs:
                        if nl and nl in norm_line:
                            keep = False
                            removed_any = True
                            break
                    if keep:
                        out_lines.append(line)
                return "\n".join(out_lines), removed_any

            full_ref_text, _ = _remove_label_lines_normalized(full_ref_text, header_label_text)
            full_hyp_text, _ = _remove_label_lines_normalized(full_hyp_text, header_label_text)
            full_ref_text, _ = _remove_label_lines_normalized(full_ref_text, footer_label_text)
            full_hyp_text, _ = _remove_label_lines_normalized(full_hyp_text, footer_label_text)

            if debug and dbg_count < debug_limit:
                post_strip_ref = (full_ref_text or '')[:1000]
                post_strip_hyp = (full_hyp_text or '')[:1000]
                # Print a clear PDF-level divider
                print("\n" + "=" * 80)
                print(f"DEBUG: PDF={pdf.name}  (sample {dbg_count+1}/{debug_limit})")
                print("=" * 80)
                # Print reference pre/post snippets
                print("-- Reference (pre-strip) --")
                print(pre_strip_ref or "<empty>")
                print("-- Reference (post-strip) --")
                print(post_strip_ref or "<empty>")
                print("-" * 60)
                # For hypotheses, print per-method labeled sections with durations
                print("-- Hypotheses (per method) --")
                for mtd, (mtext, mdur) in extracted.items():
                    # show only short snippets to keep output readable
                    m_pre = (mtext or '')[:800]
                    # attempt to remove header/footer from this method's text for post view
                    m_post_text, _ = _remove_label_lines_normalized(mtext or '', header_label_text)
                    m_post_text, _ = _remove_label_lines_normalized(m_post_text, footer_label_text)
                    m_post = (m_post_text or '')[:800]
                    print(f"[{mtd}] duration={round(mdur, 3)}s")
                    print("  pre: ")
                    print("    " + (m_pre.replace('\n', '\n    ') or '<empty>'))
                    print("  post: ")
                    print("    " + (m_post.replace('\n', '\n    ') or '<empty>'))
                    print("-" * 40)
                dbg_count += 1

            # After label-based stripping, split into sections and compute only body metrics
            hdr_ref, body_ref, ftr_ref = split_doc_sections(full_ref_text, include_template=False)
            hdr_hyp, body_hyp, ftr_hyp = split_doc_sections(full_hyp_text, include_template=False)

            # Canonicalized body (no template/header/footer lines) for inspection
            ref_body = canonicalize_doc_text(body_ref, include_template=False)
            hyp_body = canonicalize_doc_text(body_hyp, include_template=False)

            # Body-level normalization/tokenization for metrics
            ref_body_norm = _normalize_for_match(ref_body) if ref_body else ""
            hyp_body_norm = _normalize_for_match(hyp_body) if hyp_body else ""
            ref_body_tok = _tokenize_for_metric(ref_body)
            hyp_body_tok = _tokenize_for_metric(hyp_body)

            # Compute body-level metrics (CER / WER / BLEU / exact)
            try:
                body_cer = cer(ref_body_norm, hyp_body_norm)
            except Exception:
                body_cer = 1.0
            try:
                body_wer = wer(ref_body_tok, hyp_body_tok)
            except Exception:
                body_wer = 1.0
            try:
                body_bleu = bleu_score(ref_body_tok, hyp_body_tok)
            except Exception:
                body_bleu = 0.0
            body_exact = (ref_body_norm == hyp_body_norm)

            rows.append({
                'pdf': pdf.name,
                'method': method,
                'time_s': _format_num(duration),
                'ref_body': ref_body,
                'hyp_body': hyp_body,
                'body_cer': _format_num(body_cer),
                'body_wer': _format_num(body_wer),
                'body_bleu': _format_num(body_bleu),
                'body_exact': body_exact,
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
    parser.add_argument('--debug', action='store_true', help='print ref/hyp before and after header/footer stripping for first files')
    parser.add_argument('--debug-limit', type=int, default=2, help='number of files to print debug snippets for')
    parser.add_argument('--no-summary', action='store_true', help='skip automatic summary metrics display')
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

    run_benchmark(num_pdfs=args.num, labels_csv=labels_path, limit_pdfs=args.limit, debug=args.debug, debug_limit=args.debug_limit)
    
    # Auto-display summary metrics unless disabled
    if not args.no_summary and RESULTS.exists():
        try:
            print("\n")
            # Import and run summary metrics
            import summary_metrics
            summary_metrics.calculate_summary_metrics(str(RESULTS))
        except Exception as e:
            print(f"Note: Could not display summary metrics: {e}")
            print(f"Run 'python summary_metrics.py' to see the summary.")


if __name__ == '__main__':
    main()

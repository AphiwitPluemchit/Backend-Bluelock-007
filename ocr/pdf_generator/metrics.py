import math
from typing import Tuple

def levenshtein(a: str, b: str) -> int:
    """Compute Levenshtein distance between two strings."""
    if a == b:
        return 0
    la, lb = len(a), len(b)
    if la == 0:
        return lb
    if lb == 0:
        return la
    prev = list(range(lb + 1))
    for i, ca in enumerate(a, start=1):
        cur = [i] + [0] * lb
        for j, cb in enumerate(b, start=1):
            cost = 0 if ca == cb else 1
            cur[j] = min(prev[j] + 1, cur[j-1] + 1, prev[j-1] + cost)
        prev = cur
    return prev[lb]


def cer(ref: str, hyp: str) -> float:
    """Character Error Rate: levenshtein(chars)/len(ref_chars)"""
    ref_s = ref or ""
    hyp_s = hyp or ""
    if len(ref_s) == 0:
        return 0.0 if len(hyp_s) == 0 else 1.0
    dist = levenshtein(ref_s, hyp_s)
    return dist / max(1, len(ref_s))


def wer(ref: str, hyp: str) -> float:
    """Word Error Rate: levenshtein(words)/len(ref_words)"""
    r = (ref or "").split()
    h = (hyp or "").split()
    if len(r) == 0:
        return 0.0 if len(h) == 0 else 1.0
    # map words to characters to reuse levenshtein
    # build dict
    vocab = {}
    next_id = 0
    def ids(seq):
        nonlocal next_id
        out = []
        for w in seq:
            if w not in vocab:
                vocab[w] = chr(0x1000 + next_id)
                next_id += 1
            out.append(vocab[w])
        return ''.join(out)
    rid = ids(r)
    hid = ids(h)
    dist = levenshtein(rid, hid)
    return dist / max(1, len(r))


def exact_match(ref: str, hyp: str) -> bool:
    return (ref or "") == (hyp or "")


def bleu_score(ref: str, hyp: str) -> float:
    """Compute a simple BLEU score using unigram-bigram up to 4-gram with brevity penalty.
    If sacrebleu or nltk not available, fall back to a simple precision-based estimate.
    """
    try:
        import sacrebleu
        return sacrebleu.sentence_bleu(hyp, [ref]).score / 100.0
    except Exception:
        pass
    try:
        from nltk.translate.bleu_score import sentence_bleu, SmoothingFunction
        ref_t = (ref or "").split()
        hyp_t = (hyp or "").split()
        if len(ref_t) == 0:
            return 1.0 if len(hyp_t) == 0 else 0.0
        weights = (0.25, 0.25, 0.25, 0.25)
        sc = sentence_bleu([ref_t], hyp_t, weights=weights, smoothing_function=SmoothingFunction().method1)
        return float(sc)
    except Exception:
        # fallback: unigram precision with brevity penalty
        r = (ref or "").split()
        h = (hyp or "").split()
        if not r:
            return 1.0 if not h else 0.0
        # if hypothesis empty but reference not empty -> zero score
        if len(h) == 0:
            return 0.0
        # count overlap
        from collections import Counter
        rc = Counter(r)
        hc = Counter(h)
        overlap = sum(min(rc[w], hc.get(w, 0)) for w in rc)
        prec = overlap / max(1, len(h))
        bp = math.exp(1 - len(r) / len(h)) if len(h) < len(r) else 1.0
        return prec * bp


def summarize_metrics(ref: str, hyp: str) -> dict:
    return {
        'cer': cer(ref, hyp),
        'wer': wer(ref, hyp),
        'bleu': bleu_score(ref, hyp),
        'exact': exact_match(ref, hyp)
    }


def heuristic_extract_name_course(text: str) -> Tuple[str, str]:
    """Very small heuristic: return the largest line as name and the next non-empty as course.
    This is only for synthetic PDFs where name is prominent.
    """
    lines = [l.strip() for l in (text or "").splitlines() if l.strip()]
    if not lines:
        return "", ""
    # choose the longest line as name
    name = max(lines, key=len)
    # course: pick the longest remaining that is not the name
    rem = [l for l in lines if l != name]
    course = rem[0] if rem else ""
    return name, course

import re, unicodedata
from rapidfuzz import fuzz

PREFIXES = ["นาย","นางสาว","นาง","mr.","miss","mrs.","mr"]

def _map_thai_digits(s: str) -> str:
    thai = "๐๑๒๓๔๕๖๗๘๙"
    for i, d in enumerate(thai):
        s = s.replace(d, str(i))
    return s

def normalize_min(s: str) -> str:
    if not s: return ""
    # จัดการ whitespace แปลก ๆ
    s = s.replace("\u200B","").replace("\u2060","").replace("\u00A0"," ")
    # จัดรูป Unicode (สระ/วรรณยุกต์, ligature)
    s = unicodedata.normalize("NFKC", s)
    s = _map_thai_digits(s)
    s = s.lower()
    # บีบช่องว่าง
    s = re.sub(r"\s+", " ", s).strip()
    return s

def strip_prefix(name: str) -> str:
    n = name.lower()
    for p in PREFIXES:
        n = n.replace(p, "")
    return n.strip()

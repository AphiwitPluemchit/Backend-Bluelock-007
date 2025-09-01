import re, unicodedata
from rapidfuzz import fuzz


PREFIXES = ["นาย","นางสาว","นาง","mr.","miss","mrs.","mr","Mr.","Miss","Mrs.","Ms.","Ms"]

def _map_thai_digits(s: str) -> str:
    thai = "๐๑๒๓๔๕๖๗๘๙"
    for i, d in enumerate(thai):
        s = s.replace(d, str(i))
    return s
# -------- normalize แบบยืดหยุ่นเล็กน้อย --------
def normalize_min(
    s: str,
    *,
    remove_thai_internal_spaces: bool = True,  # ลบช่องว่างระหว่างอักขระไทยเท่านั้น
    remove_all_spaces: bool = False            # ลบช่องว่างทั้งหมด (ใช้เป็นทางเลือกเสริม)
) -> str:
    if not s:
        return ""
    # space พิเศษ + normalize unicode + ตัวเลขไทย -> อารบิก
    s = s.replace("\u200B","").replace("\u2060","").replace("\u00A0"," ")
    s = unicodedata.normalize("NFKC", s)
    s = _map_thai_digits(s)
    s = s.lower()

    # บีบ whitespace ทั่วไปก่อน
    s = re.sub(r"\s+", " ", s).strip()

    # ลบ "ช่องว่างคั่นตัวอักษรไทย" เช่น "ก า ร" -> "การ"
    if remove_thai_internal_spaces:
        s = re.sub(r'(?<=[\u0E00-\u0E7F])\s+(?=[\u0E00-\u0E7F])', '', s)

    # ถ้าต้องการลบทุกช่องว่างจริง ๆ
    if remove_all_spaces:
        s = s.replace(" ", "")

    return s

def strip_prefix(name: str) -> str:
    n = name.lower()
    for p in PREFIXES:
        n = n.replace(p, "")
    return n.strip()

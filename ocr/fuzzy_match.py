from rapidfuzz import fuzz

def best_score(needle: str, hay: str) -> int:
    # ใช้ค่าสูงสุดของ 3 metric ที่นิยม
    # rapidfuzz returns numbers that are already int/float; cast to int for parity
    return int(max(
        fuzz.partial_ratio(needle, hay),
        fuzz.token_set_ratio(needle, hay),
        fuzz.ratio(needle, hay),
    ))

from fuzzywuzzy import fuzz

def best_score(needle: str, hay: str) -> int:
    # ใช้ค่าสูงสุดของ 3 metric ที่นิยม
    return max(
        fuzz.partial_ratio(needle, hay),
        fuzz.token_set_ratio(needle, hay),
        fuzz.ratio(needle, hay),
    )

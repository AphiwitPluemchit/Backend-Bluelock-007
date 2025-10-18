from pathlib import Path
from extractors import extract_text_paddle

p = Path('synth_pdfs/varied_001.pdf')
b = p.read_bytes()

print("Testing PaddleOCR text structure...")
text, duration = extract_text_paddle(b, max_pages=1)

print(f"\nRaw output from PaddleOCR:")
print("="*60)
print(repr(text[:500]))
print("="*60)

print(f"\nFormatted output:")
print("="*60)
print(text)
print("="*60)

print(f"\nLines breakdown:")
lines = text.splitlines()
for i, line in enumerate(lines[:20]):
    print(f"{i:2d}: '{line}'")

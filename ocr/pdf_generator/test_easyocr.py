from pathlib import Path
import traceback

from extractors import extract_text_easyocr

p = Path('synth_pdfs/varied_001.pdf')
print('pdf path:', p.resolve())
print('exists:', p.exists())
if not p.exists():
    raise SystemExit('PDF not found')

b = p.read_bytes()
try:
    text, dur = extract_text_easyocr(b, lang_list=('th','en'), max_pages=1)
    print('duration:', dur)
    if text is None:
        print('text is None')
    else:
        print('text length:', len(text))
        print('sample repr:', repr(text[:800]))
except Exception as e:
    print('EXCEPTION:', e)
    traceback.print_exc()

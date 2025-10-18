from pathlib import Path
from extractors import extract_text_paddle

p = Path('synth_pdfs/varied_001.pdf')
print('Testing PaddleOCR extraction...')
print('PDF:', p.name)

b = p.read_bytes()
text, duration = extract_text_paddle(b, max_pages=1)

print(f'\nDuration: {duration:.2f}s')
print(f'Text length: {len(text)}')
print(f'\nExtracted text:')
print('-' * 60)
print(text)
print('-' * 60)

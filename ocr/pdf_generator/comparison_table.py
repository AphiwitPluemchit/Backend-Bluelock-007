import pandas as pd

# Read results
df = pd.read_csv('benchmark_results.csv')

# Create comparison table
comparison = []
for method in ['pymupdf', 'tesseract', 'easyocr', 'paddleocr']:
    method_data = df[df['method'] == method]
    if len(method_data) > 0:
        comparison.append({
            'OCR Method': method.upper(),
            'Speed (s)': f"{method_data['time_s'].mean():.2f}",
            'CER ↓': f"{method_data['body_cer'].mean():.3f}",
            'WER ↓': f"{method_data['body_wer'].mean():.3f}",
            'BLEU ↑': f"{method_data['body_bleu'].mean():.3f}",
            'Pros': '',
            'Cons': ''
        })

# Add pros/cons
comparison[0]['Pros'] = 'Perfect accuracy, Ultra fast'
comparison[0]['Cons'] = 'Requires text layer in PDF'

comparison[1]['Pros'] = 'Fast, Good accuracy, Mature'
comparison[1]['Cons'] = 'Less accurate than deep learning'

comparison[2]['Pros'] = 'Best accuracy for Thai text'
comparison[2]['Cons'] = 'Very slow (38s/page)'

comparison[3]['Pros'] = 'Balanced speed/accuracy'
comparison[3]['Cons'] = 'Lower accuracy than EasyOCR'

comp_df = pd.DataFrame(comparison)

print("\n" + "="*120)
print("OCR METHODS COMPARISON TABLE")
print("="*120)
print("\nNote: ↓ = Lower is better, ↑ = Higher is better")
print("-"*120)
print(comp_df.to_string(index=False))
print("-"*120)

print("\n" + "="*120)
print("QUICK DECISION GUIDE")
print("="*120)
print("""
📄 PDF has text layer (searchable PDF):
   → Use PYMUPDF (0.003s, perfect accuracy)

🖼️ Need to OCR scanned images/PDFs with Thai text:
   → Use EASYOCR (best accuracy: CER 0.186)
   → If speed matters: Use TESSERACT (fast: 1s, decent accuracy: CER 0.254)

⚡ Need balance of speed and accuracy:
   → Use PADDLEOCR (21s, CER 0.328)

💰 Production recommendation:
   1. Try PYMUPDF first (instant, free)
   2. Fallback to EASYOCR for scanned/image PDFs
   3. Use TESSERACT if processing large volumes (faster)
""")
print("="*120)

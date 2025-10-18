import pandas as pd

df = pd.read_csv('benchmark_results.csv')
print(df[['pdf', 'method', 'time_s', 'body_cer', 'body_wer', 'body_bleu']].to_string(index=False))

print('\n' + '='*80)
print('PaddleOCR Details')
print('='*80)

paddle_rows = df[df['method'] == 'paddleocr']
for idx, row in paddle_rows.iterrows():
    print(f"\nPDF: {row['pdf']}")
    print(f"Time: {row['time_s']}s")
    print(f"CER: {row['body_cer']}, WER: {row['body_wer']}, BLEU: {row['body_bleu']}")
    print(f"\nExtracted body text:")
    print(row['hyp_body'][:400])
    print('...\n')

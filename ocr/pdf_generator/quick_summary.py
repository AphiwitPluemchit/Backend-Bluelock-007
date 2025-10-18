"""
Quick summary view of OCR benchmark results in a compact table format.
"""
import pandas as pd
from pathlib import Path

def show_compact_summary(csv_path='benchmark_results.csv'):
    """Show a compact, easy-to-read summary table."""
    
    df = pd.read_csv(csv_path)
    
    # Calculate averages
    summary = df.groupby('method').agg({
        'time_s': 'mean',
        'body_cer': 'mean',
        'body_wer': 'mean',
        'body_bleu': 'mean',
    }).round(3)
    
    # Add exact match rate
    exact_match_rate = (df.groupby('method')['body_exact'].sum() / df.groupby('method').size() * 100).round(1)
    summary['exact_pct'] = exact_match_rate
    
    # Rename columns for display
    summary.columns = ['Avg Time(s)', 'Avg CER', 'Avg WER', 'Avg BLEU', 'Exact%']
    
    # Sort by CER (best OCR accuracy first)
    summary = summary.sort_values('Avg CER')
    
    # Add ranking indicators
    summary['Speed Rank'] = summary['Avg Time(s)'].rank().astype(int)
    summary['CER Rank'] = summary['Avg CER'].rank().astype(int)
    summary['BLEU Rank'] = summary['Avg BLEU'].rank(ascending=False).astype(int)
    
    print("="*100)
    print("OCR BENCHMARK SUMMARY - AVERAGE METRICS ACROSS ALL TEST PDFs")
    print("="*100)
    print("\nNote: Lower is better for CER/WER, Higher is better for BLEU")
    print("-"*100)
    print(summary.to_string())
    print("-"*100)
    
    # Best method recommendation
    print("\nRECOMMENDATIONS:")
    print("-"*100)
    
    best_speed = summary['Avg Time(s)'].idxmin()
    best_cer = summary['Avg CER'].idxmin()
    best_bleu = summary['Avg BLEU'].idxmax()
    
    print(f"  - Fastest:        {str(best_speed).upper():12s} ({summary.loc[best_speed, 'Avg Time(s)']:.3f}s)")
    print(f"  - Most Accurate:  {str(best_cer).upper():12s} (CER: {summary.loc[best_cer, 'Avg CER']:.3f})")
    print(f"  - Best Quality:   {str(best_bleu).upper():12s} (BLEU: {summary.loc[best_bleu, 'Avg BLEU']:.3f})")
    
    # Overall recommendation for OCR tasks
    ocr_methods = summary.drop('pymupdf', errors='ignore')
    if len(ocr_methods) > 0:
        best_ocr_cer = ocr_methods['Avg CER'].idxmin()
        print(f"\n  → For OCR tasks (excluding PyMuPDF): {str(best_ocr_cer).upper()}")
        print(f"    CER: {ocr_methods.loc[best_ocr_cer, 'Avg CER']:.3f}, "
              f"Time: {ocr_methods.loc[best_ocr_cer, 'Avg Time(s)']:.1f}s, "
              f"BLEU: {ocr_methods.loc[best_ocr_cer, 'Avg BLEU']:.3f}")
    
    print("="*100)
    
    return summary


if __name__ == '__main__':
    import sys
    csv_file = sys.argv[1] if len(sys.argv) > 1 else 'benchmark_results.csv'
    show_compact_summary(csv_file)

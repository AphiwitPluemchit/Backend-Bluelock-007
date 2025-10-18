import pandas as pd
import argparse
from pathlib import Path

def calculate_summary_metrics(csv_path='benchmark_results.csv'):
    """Calculate and display average metrics for each OCR method."""
    
    # Read the benchmark results
    df = pd.read_csv(csv_path)
    
    # Calculate average metrics grouped by method
    summary = df.groupby('method').agg({
        'time_s': ['mean', 'std', 'min', 'max'],
        'body_cer': ['mean', 'std', 'min', 'max'],
        'body_wer': ['mean', 'std', 'min', 'max'],
        'body_bleu': ['mean', 'std', 'min', 'max'],
    }).round(3)
    
    # Count exact matches
    exact_match_count = df.groupby('method')['body_exact'].sum()
    total_count = df.groupby('method').size()
    exact_match_rate = (exact_match_count / total_count * 100).round(1)
    
    # Create a simplified summary table with just averages
    simple_summary = pd.DataFrame({
        'Method': df.groupby('method').size().index,
        'Avg Time (s)': df.groupby('method')['time_s'].mean().round(3).values,
        'Avg CER': df.groupby('method')['body_cer'].mean().round(3).values,
        'Avg WER': df.groupby('method')['body_wer'].mean().round(3).values,
        'Avg BLEU': df.groupby('method')['body_bleu'].mean().round(3).values,
        'Exact Match (%)': exact_match_rate.values,
        'Total PDFs': total_count.values
    })
    
    # Sort by CER (lower is better)
    simple_summary = simple_summary.sort_values('Avg CER')
    
    print("="*90)
    print("OCR METHODS COMPARISON - AVERAGE METRICS")
    print("="*90)
    print("\nSimple Summary (sorted by CER - lower is better):")
    print("-"*90)
    print(simple_summary.to_string(index=False))
    
    print("\n" + "="*90)
    print("DETAILED STATISTICS")
    print("="*90)
    
    # Display detailed statistics
    methods = df['method'].unique()
    for method in methods:
        method_data = df[df['method'] == method]
        print(f"\n{method.upper()}")
        print("-"*50)
        print(f"  Time:      {method_data['time_s'].mean():.3f}s ± {method_data['time_s'].std():.3f}s "
              f"(min: {method_data['time_s'].min():.3f}s, max: {method_data['time_s'].max():.3f}s)")
        print(f"  CER:       {method_data['body_cer'].mean():.3f} ± {method_data['body_cer'].std():.3f} "
              f"(min: {method_data['body_cer'].min():.3f}, max: {method_data['body_cer'].max():.3f})")
        print(f"  WER:       {method_data['body_wer'].mean():.3f} ± {method_data['body_wer'].std():.3f} "
              f"(min: {method_data['body_wer'].min():.3f}, max: {method_data['body_wer'].max():.3f})")
        print(f"  BLEU:      {method_data['body_bleu'].mean():.3f} ± {method_data['body_bleu'].std():.3f} "
              f"(min: {method_data['body_bleu'].min():.3f}, max: {method_data['body_bleu'].max():.3f})")
        print(f"  Exact:     {method_data['body_exact'].sum()}/{len(method_data)} "
              f"({method_data['body_exact'].sum()/len(method_data)*100:.1f}%)")
    
    # Performance ranking
    print("\n" + "="*90)
    print("RANKING (Best to Worst)")
    print("="*90)
    
    rankings = {
        'By Speed (fastest)': simple_summary.sort_values('Avg Time (s)')[['Method', 'Avg Time (s)']],
        'By Accuracy CER (lowest)': simple_summary.sort_values('Avg CER')[['Method', 'Avg CER']],
        'By Accuracy WER (lowest)': simple_summary.sort_values('Avg WER')[['Method', 'Avg WER']],
        'By Quality BLEU (highest)': simple_summary.sort_values('Avg BLEU', ascending=False)[['Method', 'Avg BLEU']],
    }
    
    for title, ranking in rankings.items():
        print(f"\n{title}:")
        for idx, (_, row) in enumerate(ranking.iterrows(), 1):
            method = row['Method']
            value = row.iloc[1]
            # Use simple numbers instead of emojis for Windows compatibility
            rank_symbols = ['[1st]', '[2nd]', '[3rd]', '[4th]']
            symbol = rank_symbols[min(idx-1, 3)]
            print(f"  {symbol} {method:12s} - {value}")
    
    # Save summary to CSV
    summary_csv = Path(csv_path).parent / 'benchmark_summary.csv'
    simple_summary.to_csv(summary_csv, index=False)
    print(f"\n{'='*90}")
    print(f"Summary saved to: {summary_csv}")
    print("="*90)
    
    return simple_summary


def main():
    parser = argparse.ArgumentParser(description="Calculate summary metrics from benchmark results")
    parser.add_argument('-f', '--file', type=str, default='benchmark_results.csv', 
                        help='Path to benchmark results CSV file')
    args = parser.parse_args()
    
    calculate_summary_metrics(args.file)


if __name__ == '__main__':
    main()

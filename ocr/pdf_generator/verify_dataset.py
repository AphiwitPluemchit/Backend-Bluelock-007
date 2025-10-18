"""Quick verification script for generated dataset."""
import pandas as pd

df = pd.read_csv('labels_varied.csv')

print("=" * 60)
print("Dataset Verification Report")
print("=" * 60)
print(f"Total records: {len(df)}")
print()

print("First 3 records:")
print(df.head(3).to_string())
print()

print("Last 3 records:")
print(df.tail(3).to_string())
print()

print("=" * 60)
print("Variety Statistics")
print("=" * 60)
print(f"Unique Thai names: {df['name_th'].nunique()}")
print(f"Unique English names: {df['name_en'].nunique()}")
print(f"Unique Thai courses: {df['course_th'].nunique()}")
print(f"Unique English courses: {df['course_en'].nunique()}")
print()

# Check for variations (batch numbers)
batch_records = df[df['name_th'].str.contains(r'\(รุ่น', na=False)]
print(f"Records with batch variations: {len(batch_records)}")

# Check for long names
long_name_records = df[df['name_th'].str.contains('พิพัฒน์สมบูรณ์กิจ|จันทร์จำปา|เจริญพงศ์พัฒนากิจการ', na=False)]
print(f"Records with long names: {len(long_name_records)}")
print()

print("=" * 60)
print("✓ Dataset verification complete!")
print("=" * 60)

@echo off
REM Quick benchmark run - 10 PDFs only
echo Running benchmark on first 10 PDFs with labels...

REM Find Python executable
set PYTHON_EXE=
if exist "..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\.venv\Scripts\python.exe
) else if exist "..\..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\..\.venv\Scripts\python.exe
) else (
    set PYTHON_EXE=python
)

%PYTHON_EXE% benchmark.py --limit 10
echo.
echo Results saved to benchmark_results.csv
pause
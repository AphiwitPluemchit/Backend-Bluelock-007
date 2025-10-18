@echo off
REM Full benchmark run with all PDFs
echo Running full benchmark with labels...

REM Find Python executable
set PYTHON_EXE=
if exist "..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\.venv\Scripts\python.exe
) else if exist "..\..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\..\.venv\Scripts\python.exe
) else (
    set PYTHON_EXE=python
)

%PYTHON_EXE% benchmark.py -n 0
echo.
echo Results saved to benchmark_results.csv
pause
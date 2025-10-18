@echo off
REM Setup script for OCR benchmark dependencies
echo Installing benchmark dependencies...

REM Find Python executable (try common locations)
set PYTHON_EXE=
if exist "..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\.venv\Scripts\python.exe
) else if exist "..\..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\..\.venv\Scripts\python.exe
) else (
    set PYTHON_EXE=python
)

echo Using Python: %PYTHON_EXE%
%PYTHON_EXE% -m pip install -r requirements_benchmark.txt

echo.
echo Setup complete! You can now run:
echo   run_benchmark.bat
echo   run_benchmark_quick.bat (10 PDFs only)
pause
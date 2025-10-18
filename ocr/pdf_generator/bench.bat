@echo off
REM Simple benchmark runner (resolve venv relative to this script's folder)
set PYTHON_EXE=
set SCRIPT_DIR=%~dp0
REM check one and two levels up from this script directory
if exist "%SCRIPT_DIR%..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=%SCRIPT_DIR%..\.venv\Scripts\python.exe
) else if exist "%SCRIPT_DIR%..\..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=%SCRIPT_DIR%..\..\.venv\Scripts\python.exe
) else if exist "%SCRIPT_DIR%.venv\Scripts\python.exe" (
    set PYTHON_EXE=%SCRIPT_DIR%.venv\Scripts\python.exe
) else (
    set PYTHON_EXE=python
)

REM If no args provided, use a sensible default for quick debug runs
if "%*"=="" (
    set ARGS=--limit 2 --labels labels_varied.csv --debug
) else (
    set ARGS=%*
)

echo Using Python: %PYTHON_EXE%
echo Running: %PYTHON_EXE% "%SCRIPT_DIR%benchmark.py" %ARGS%

REM Change working directory to the script directory so relative paths inside
REM benchmark.py (synth_pdfs, labels_varied.csv, etc.) resolve correctly.
pushd "%SCRIPT_DIR%" >nul 2>&1
%PYTHON_EXE% "%SCRIPT_DIR%benchmark.py" %ARGS%
set EXIT_CODE=%ERRORLEVEL%
popd >nul 2>&1
exit /b %EXIT_CODE%
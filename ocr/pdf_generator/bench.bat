@echo off
REM Simple benchmark runner
set PYTHON_EXE=
if exist "..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\.venv\Scripts\python.exe
) else if exist "..\..\.venv\Scripts\python.exe" (
    set PYTHON_EXE=..\..\.venv\Scripts\python.exe
) else (
    set PYTHON_EXE=python
)

%PYTHON_EXE% benchmark.py %*
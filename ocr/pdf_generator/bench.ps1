# Simple benchmark runner (PowerShell)
$pythonExe = "python"
if (Test-Path "..\.venv\Scripts\python.exe") {
    $pythonExe = "..\.venv\Scripts\python.exe"
} elseif (Test-Path "..\..\.venv\Scripts\python.exe") {
    $pythonExe = "..\..\.venv\Scripts\python.exe"
}

& $pythonExe benchmark.py @args
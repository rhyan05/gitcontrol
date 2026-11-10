$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root ".env"
if (-not (Test-Path -LiteralPath $envFile)) {
    throw ".env não encontrado em $envFile"
}

$vars = @{}
Get-Content -LiteralPath $envFile | ForEach-Object {
    if ($_ -match '^\s*([^#=]+?)\s*=\s*(.*)\s*$') {
        $vars[$matches[1]] = $matches[2]
    }
}

if (-not $vars.ContainsKey("password") -or [string]::IsNullOrWhiteSpace($vars["password"])) {
    throw ".env precisa de password (token do PyPI)"
}

$env:TWINE_USERNAME = if ($vars.ContainsKey("username")) { $vars["username"] } else { "__token__" }
$env:TWINE_PASSWORD = $vars["password"]

Push-Location $root
try {
    python scripts/build_pypi.py
    if ($LASTEXITCODE -ne 0) { throw "build_pypi.py falhou" }

    python -m build --wheel
    if ($LASTEXITCODE -ne 0) { throw "python -m build falhou" }

    python -m twine check dist/*.whl
    if ($LASTEXITCODE -ne 0) { throw "twine check falhou" }

    python -m twine upload --non-interactive --repository pypi dist/*.whl
    if ($LASTEXITCODE -ne 0) { throw "twine upload falhou" }
} finally {
    Pop-Location
}

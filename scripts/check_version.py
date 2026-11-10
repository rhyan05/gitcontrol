#!/usr/bin/env python3
"""Garante que a versão seja a mesma em todos os manifests."""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def main() -> int:
    go = re.search(r'var version = "([^"]+)"', read("cmd/gitcontrol/main.go"))
    pyproject = re.search(r'^version = "([^"]+)"', read("pyproject.toml"), re.M)
    init = re.search(r'__version__ = "([^"]+)"', read("packaging/pypi/gitcontrol/__init__.py"))
    package = json.loads(read("desktop/package.json"))["version"]
    tauri = json.loads(read("desktop/src-tauri/tauri.conf.json"))["version"]
    cargo = re.search(r'^version = "([^"]+)"', read("desktop/src-tauri/Cargo.toml"), re.M)

    versions = {
        "cmd/gitcontrol/main.go": go.group(1) if go else "",
        "pyproject.toml": pyproject.group(1) if pyproject else "",
        "packaging/pypi/gitcontrol/__init__.py": init.group(1) if init else "",
        "desktop/package.json": package,
        "desktop/src-tauri/tauri.conf.json": tauri,
        "desktop/src-tauri/Cargo.toml": cargo.group(1) if cargo else "",
    }

    expected = versions["cmd/gitcontrol/main.go"]
    if not expected:
        print("versão não encontrada em cmd/gitcontrol/main.go", file=sys.stderr)
        return 1

    bad = {path: value for path, value in versions.items() if value != expected}
    if bad:
        print(f"versões divergentes (esperado {expected}):", file=sys.stderr)
        for path, value in bad.items():
            print(f"  {path}: {value!r}", file=sys.stderr)
        return 1

    print(f"versão consistente: {expected}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

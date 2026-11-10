#!/usr/bin/env python3
"""Compila os binários Go para o wheel universal do PyPI."""

import os
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BIN_DIR = ROOT / "packaging" / "pypi" / "gitcontrol" / "bin"

TARGETS = [
    ("windows", "amd64"),
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
]


def main() -> None:
    version = os.environ.get("VERSION", "0.1.0")
    shutil.rmtree(BIN_DIR, ignore_errors=True)
    BIN_DIR.mkdir(parents=True, exist_ok=True)

    for goos, goarch in TARGETS:
        ext = ".exe" if goos == "windows" else ""
        output = BIN_DIR / f"gitcontrol-{goos}-{goarch}{ext}"
        env = os.environ.copy()
        env.update(GOOS=goos, GOARCH=goarch, CGO_ENABLED="0")
        subprocess.run(
            [
                "go",
                "build",
                "-trimpath",
                "-ldflags",
                f"-s -w -X main.version={version}",
                "-o",
                str(output),
                "./cmd/gitcontrol",
            ],
            cwd=ROOT,
            env=env,
            check=True,
        )
        print(f"[gitcontrol] {output.relative_to(ROOT)}")


if __name__ == "__main__":
    main()

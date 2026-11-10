"""Ponto de entrada: executa o binário Go correspondente à plataforma."""

import os
import platform
import stat
import subprocess
import sys
from pathlib import Path

_BIN_DIR = Path(__file__).resolve().with_name("bin")


def target_binary() -> Path:
    """Mapeia plataforma/arquitetura para o binário embutido no wheel."""
    if sys.platform.startswith("win"):
        os_name, ext = "windows", ".exe"
    elif sys.platform == "darwin":
        os_name, ext = "darwin", ""
    else:
        os_name, ext = "linux", ""

    machine = platform.machine().lower()
    arch = "arm64" if machine in ("arm64", "aarch64") else "amd64"
    return _BIN_DIR / f"gitcontrol-{os_name}-{arch}{ext}"


def main() -> int:
    binary = target_binary()
    if not binary.exists():
        print(
            f"gitcontrol: binário não encontrado para esta plataforma: {binary.name}",
            file=sys.stderr,
        )
        return 1

    if os.name != "nt":
        mode = binary.stat().st_mode
        binary.chmod(mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
        os.execv(str(binary), [str(binary), *sys.argv[1:]])
        return 127

    return subprocess.call([str(binary), *sys.argv[1:]])


if __name__ == "__main__":
    raise SystemExit(main())

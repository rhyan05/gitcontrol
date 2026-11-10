// Compila o CLI Go como sidecar do Tauri.
//
// O Tauri exige o sufixo do target triple no nome do binário:
//   src-tauri/binaries/gitcontrol-cli-x86_64-pc-windows-msvc.exe
// O `externalBin` no tauri.conf.json aponta para "binaries/gitcontrol-cli" e o Tauri
// resolve o sufixo sozinho em dev e no bundle.
//
// O nome tem "gitcontrol-cli" (e não "gitcontrol") porque o binário do app também se chama
// gitcontrol.exe: o sidecar é resolvido ao lado do executável e os dois colidiriam.
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { homedir } from "node:os";
import { join, resolve } from "node:path";

const desktopDir = resolve(import.meta.dirname, "..");
const repoRoot = resolve(desktopDir, "..");

// rustc pode não estar no PATH (instalação via rustup no Windows).
const exe = process.platform === "win32" ? ".exe" : "";
const rustupRustc = join(homedir(), ".cargo", "bin", `rustc${exe}`);
const rustc = process.env.RUSTC ?? (existsSync(rustupRustc) ? rustupRustc : "rustc");

const target = execFileSync(rustc, ["--print", "host-tuple"], {
  encoding: "utf8",
}).trim();
if (!target) {
  throw new Error("não foi possível obter o host triple do rustc");
}

const outDir = join(desktopDir, "src-tauri", "binaries");
mkdirSync(outDir, { recursive: true });

const out = join(outDir, `gitcontrol-cli-${target}${exe}`);
console.log(`[gitcontrol] compilando sidecar -> ${out}`);
execFileSync("go", ["build", "-o", out, "./cmd/gitcontrol"], {
  cwd: repoRoot,
  stdio: "inherit",
});

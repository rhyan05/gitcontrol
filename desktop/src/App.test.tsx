import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { ToastProvider } from "./components/ui/Toast";
import { runGitControl } from "./lib/gitcontrol";
import type { Branch, Commit, StatusResult } from "./lib/gitcontrol";

vi.mock("./lib/gitcontrol", () => ({ runGitControl: vi.fn() }));
vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));

const runMock = vi.mocked(runGitControl);

const branches: Branch[] = [{ name: "master", head: true }];
const commits: Commit[] = [
  {
    hash: "abcdef1234567890",
    author: "Rhyan Souza",
    time: new Date().toISOString(),
    subject: "primeiro commit",
  },
];

function statusWith(extra: Partial<StatusResult> = {}): StatusResult {
  return {
    branch: "master",
    head_oid: "abcdef1234567890",
    files: [
      {
        path: "src/App.tsx",
        index_xy: ".",
        worktree_xy: "M",
      },
    ],
    ...extra,
  };
}

function mockRepo(status: StatusResult) {
  runMock.mockImplementation(
    (async (_repo: string, ...args: string[]) => {
      switch (args[0]) {
        case "status":
          return status;
        case "branches":
          return branches;
        case "log":
          return commits;
        case "graph":
          return commits.map((commit) => ({ graph: "*", commit }));
        case "show":
          return { hash: commits[0].hash, diff: "+linha nova" };
        case "diff":
          return { path: args[1], diff: "+linha nova" };
        default:
          return null;
      }
    }) as typeof runGitControl,
  );
}

function renderApp() {
  return render(
    <ToastProvider>
      <App />
    </ToastProvider>,
  );
}

beforeEach(() => {
  localStorage.clear();
  runMock.mockReset();
});

describe("App", () => {
  it("renderiza os três painéis com dados reais", async () => {
    mockRepo(statusWith());
    renderApp();

    expect(await screen.findByText("primeiro commit")).toBeInTheDocument();
    expect(screen.getByText("src/App.tsx")).toBeInTheDocument();
    expect(screen.getAllByText("master").length).toBeGreaterThan(0);
  });

  it("faz stage do arquivo selecionado", async () => {
    const user = userEvent.setup();
    mockRepo(statusWith());
    renderApp();

    const stageButton = await screen.findByRole("button", {
      name: "Stage src/App.tsx",
    });
    await user.click(stageButton);

    await waitFor(() => {
      expect(runMock).toHaveBeenCalledWith(
        expect.any(String),
        "stage",
        "src/App.tsx",
      );
    });
    expect(await screen.findByRole("status")).toHaveTextContent(
      "stage: src/App.tsx",
    );
  });

  it("cria commit quando há arquivo staged", async () => {
    const user = userEvent.setup();
    mockRepo(
      statusWith({
        files: [{ path: "src/App.tsx", index_xy: "A", worktree_xy: "." }],
      }),
    );
    renderApp();

    const input = await screen.findByLabelText("Mensagem do commit");
    await user.type(input, "nova feature");
    await user.click(screen.getByRole("button", { name: "Commit" }));

    await waitFor(() => {
      expect(runMock).toHaveBeenCalledWith(
        expect.any(String),
        "commit",
        "nova feature",
      );
    });
  });

  it("confirma cherry-pick antes de executar", async () => {
    const user = userEvent.setup();
    mockRepo(statusWith());
    renderApp();

    await user.click(
      await screen.findByRole("button", { name: /primeiro commit/i }),
    );
    await user.click(screen.getByRole("button", { name: "Cherry-pick" }));
    await user.click(screen.getByRole("button", { name: "Aplicar commit" }));

    await waitFor(() => {
      expect(runMock).toHaveBeenCalledWith(
        expect.any(String),
        "cherry-pick",
        "abcdef1234567890",
      );
    });
  });

  it("abre o diff do arquivo ao clicar no status", async () => {
    const user = userEvent.setup();
    mockRepo(statusWith());
    renderApp();

    await user.click(
      await screen.findByRole("button", { name: "src/App.tsx" }),
    );

    await waitFor(() => {
      expect(runMock).toHaveBeenCalledWith(
        expect.any(String),
        "diff",
        "src/App.tsx",
      );
    });
    expect(await screen.findByText("+linha nova")).toBeInTheDocument();
  });

  it("continua merge em conflito", async () => {
    const user = userEvent.setup();
    mockRepo(
      statusWith({
        operation: "merge",
        files: [
          { path: "a.txt", index_xy: "U", worktree_xy: "U", unmerged: true },
        ],
      }),
    );
    renderApp();

    expect(await screen.findByText("merge em andamento")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    await waitFor(() => {
      expect(
        screen.getAllByRole("button", { name: "Continuar" }),
      ).toHaveLength(2);
    });
    const buttons = screen.getAllByRole("button", { name: "Continuar" });
    await user.click(buttons[buttons.length - 1]);

    await waitFor(() => {
      expect(runMock).toHaveBeenCalledWith(expect.any(String), "continue");
    });
  });

  it("filtra os painéis pela busca", async () => {
    const user = userEvent.setup();
    mockRepo(statusWith());
    renderApp();

    await screen.findByText("primeiro commit");
    await user.type(screen.getByLabelText("Filtrar branches, commits e arquivos"), "xyz");

    expect(screen.getByText("Nenhum commit encontrado")).toBeInTheDocument();
  });
});

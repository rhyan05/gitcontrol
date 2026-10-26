import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { ToastProvider } from "./components/ui/Toast";
import { gg } from "./lib/gg";
import type { Branch, Commit, StatusResult } from "./lib/gg";

vi.mock("./lib/gg", () => ({ gg: vi.fn() }));
vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));

const ggMock = vi.mocked(gg);

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
  ggMock.mockImplementation(
    (async (_repo: string, ...args: string[]) => {
      switch (args[0]) {
        case "status":
          return status;
        case "branches":
          return branches;
        case "log":
          return commits;
        case "show":
          return { hash: commits[0].hash, diff: "+linha nova" };
        default:
          return null;
      }
    }) as typeof gg,
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
  ggMock.mockReset();
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
      expect(ggMock).toHaveBeenCalledWith(
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
      expect(ggMock).toHaveBeenCalledWith(
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
      expect(ggMock).toHaveBeenCalledWith(
        expect.any(String),
        "cherry-pick",
        "abcdef1234567890",
      );
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

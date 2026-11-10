import { useCallback, useEffect, useRef, useState } from "react";
import {
  runGitControl,
  type Branch,
  type Commit,
  type GraphRow,
  type StatusResult,
} from "../lib/gitcontrol";

const POLL_MS = 3000;

export type RepoData = {
  status: StatusResult | null;
  branches: Branch[];
  commits: Commit[];
  graph: GraphRow[];
};

const emptyData: RepoData = {
  status: null,
  branches: [],
  commits: [],
  graph: [],
};

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

// Carrega status/branches/commits/grafo e mantém a tela viva com polling curto.
// O refresh silencioso não acende o skeleton.
export function useRepo(repo: string) {
  const [data, setData] = useState<RepoData>(emptyData);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const requestId = useRef(0);

  const refresh = useCallback(
    async (silent = false) => {
      const id = ++requestId.current;
      if (silent) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }

      try {
        const [status, branches, commits, graph] = await Promise.all([
          runGitControl<StatusResult>(repo, "status"),
          runGitControl<Branch[]>(repo, "branches"),
          runGitControl<Commit[]>(repo, "log"),
          runGitControl<GraphRow[]>(repo, "graph"),
        ]);
        if (id !== requestId.current) return;

        setData({ status, branches, commits, graph });
        setError("");
      } catch (err) {
        if (id === requestId.current) {
          setError(errorMessage(err));
        }
      } finally {
        if (id === requestId.current) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [repo],
  );

  useEffect(() => {
    void refresh(false);
  }, [refresh]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!document.hidden) void refresh(true);
    }, POLL_MS);

    return () => window.clearInterval(timer);
  }, [refresh]);

  return {
    ...data,
    loading,
    refreshing,
    error,
    refresh,
  };
}

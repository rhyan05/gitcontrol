import { useCallback, useEffect, useRef, useState } from "react";
import {
  gg,
  type Branch,
  type Commit,
  type StatusResult,
} from "../lib/gg";

const POLL_MS = 3000;

export type RepoData = {
  status: StatusResult | null;
  branches: Branch[];
  commits: Commit[];
};

const emptyData: RepoData = {
  status: null,
  branches: [],
  commits: [],
};

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

// Carrega status/branches/commits e mantém a tela viva com polling curto.
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
        const [status, branches, commits] = await Promise.all([
          gg<StatusResult>(repo, "status"),
          gg<Branch[]>(repo, "branches"),
          gg<Commit[]>(repo, "log"),
        ]);
        if (id !== requestId.current) return;

        setData({ status, branches, commits });
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

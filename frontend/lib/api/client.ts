import { getAccessToken, setAccessToken, clearSession } from "../auth";

export const BASE_URL = process.env.NEXT_PUBLIC_API_URL!;

export interface ApiError {
  error: {
    code: string;
    message: string;
    field?: string | null;
  };
}

export interface PaginationMeta {
  total: number;
  page: number;
  per_page: number;
}

export interface ListResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

let refreshPromise: Promise<string | null> | null = null;

async function doRefresh(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      const res = await fetch(`${BASE_URL}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
      if (!res.ok) {
        clearSession();
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
        return null;
      }
      const data: { access_token: string } = await res.json();
      setAccessToken(data.access_token);
      return data.access_token;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
  retried = false
): Promise<T> {
  const token = getAccessToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };

  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    credentials: "include",
    headers,
  });

  if (res.status === 401 && !retried) {
    const newToken = await doRefresh();
    if (newToken) {
      return apiFetch<T>(path, options, true);
    }
    throw { error: { code: "UNAUTHORIZED", message: "Session expired. Please log in again." } } as ApiError;
  }

  if (!res.ok) {
    const error: ApiError = await res.json().catch(() => ({
      error: { code: "UNKNOWN_ERROR", message: res.statusText },
    }));
    throw error;
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  if (!text.trim()) {
    return undefined as T;
  }
  return JSON.parse(text) as T;
}

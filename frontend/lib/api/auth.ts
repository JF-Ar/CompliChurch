import { apiFetch, BASE_URL, type ApiError } from "./client";
import type { Member } from "./members";
import type { Church } from "./churches";

export { setSession } from "../auth";

export interface LoginResponse {
  access_token: string;
  member: Member;
  church: Church;
}

export interface RegisterRequest {
  church_name: string;
  pastor_name: string;
  email: string;
  password: string;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const error: ApiError = await res.json().catch(() => ({
      error: { code: "UNKNOWN_ERROR", message: res.statusText },
    }));
    throw error;
  }
  return res.json();
}

export async function register(body: RegisterRequest): Promise<LoginResponse> {
  const res = await fetch(`${BASE_URL}/auth/register`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const error: ApiError = await res.json().catch(() => ({
      error: { code: "UNKNOWN_ERROR", message: res.statusText },
    }));
    throw error;
  }
  return res.json();
}

export async function logout(): Promise<void> {
  return apiFetch("/auth/logout", { method: "POST" });
}

export async function logoutAll(): Promise<void> {
  return apiFetch("/auth/logout-all", { method: "POST" });
}

import { apiFetch } from "./client";

export interface Church {
  id: string;
  parent_church_id?: string | null;
  name: string;
  denomination_name?: string | null;
  cnpj?: string | null;
  address?: string | null;
  is_autonomous: boolean;
  plan_tier: "free" | "basic" | "growth" | "enterprise";
  member_count_cache: number;
  created_at: string;
}

export async function getMyChurch(): Promise<Church> {
  return apiFetch("/churches/me");
}

export async function listCongregations(): Promise<{ data: Church[] }> {
  return apiFetch("/churches/me/congregations");
}

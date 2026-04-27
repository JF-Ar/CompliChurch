import { apiFetch, ListResponse } from "./client";
import type { RoleSummary } from "./roles";

export interface MemberSummary {
  id: string;
  name: string;
  email: string;
  is_active: boolean;
}

export interface MemberInstrument {
  id: string;
  instrument_id: string;
  instrument_name: string;
  is_primary: boolean;
}

export interface Member {
  id: string;
  name: string;
  email: string;
  phone?: string | null;
  birth_date?: string | null;
  avatar_url?: string | null;
  is_active: boolean;
  roles: RoleSummary[];
  instruments: MemberInstrument[];
  created_at: string;
}

export interface MemberCreate {
  name: string;
  email: string;
  phone?: string | null;
  birth_date?: string | null;
  role_ids?: string[];
}

export interface MemberUpdate {
  name?: string;
  phone?: string | null;
  birth_date?: string | null;
}

export interface MemberInstrumentAdd {
  instrument_id: string;
  is_primary?: boolean;
}

export async function getMe(): Promise<Member> {
  return apiFetch("/members/me");
}

export async function updateMe(data: MemberUpdate): Promise<Member> {
  return apiFetch("/members/me", { method: "PUT", body: JSON.stringify(data) });
}

export async function listMembers(params?: {
  page?: number;
  per_page?: number;
  search?: string;
  role?: string;
  is_active?: boolean;
}): Promise<ListResponse<Member>> {
  const qs = new URLSearchParams();
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  if (params?.search) qs.set("search", params.search);
  if (params?.role) qs.set("role", params.role);
  if (params?.is_active !== undefined) qs.set("is_active", String(params.is_active));
  return apiFetch(`/members?${qs}`);
}

export async function getMember(id: string): Promise<Member> {
  return apiFetch(`/members/${id}`);
}

export async function createMember(data: MemberCreate): Promise<Member> {
  return apiFetch("/members", { method: "POST", body: JSON.stringify(data) });
}

export async function updateMember(id: string, data: MemberUpdate): Promise<Member> {
  return apiFetch(`/members/${id}`, { method: "PUT", body: JSON.stringify(data) });
}

export async function deactivateMember(id: string): Promise<void> {
  return apiFetch(`/members/${id}`, { method: "DELETE" });
}

export async function getMyInstruments(): Promise<{ data: MemberInstrument[] }> {
  return apiFetch("/members/me/instruments");
}

export async function addMyInstrument(data: MemberInstrumentAdd): Promise<MemberInstrument> {
  return apiFetch("/members/me/instruments", { method: "POST", body: JSON.stringify(data) });
}

export async function removeMyInstrument(instrumentId: string): Promise<void> {
  return apiFetch(`/members/me/instruments/${instrumentId}`, { method: "DELETE" });
}

export async function getMemberInstruments(
  memberId: string
): Promise<{ data: MemberInstrument[] }> {
  return apiFetch(`/members/${memberId}/instruments`);
}

export async function addMemberInstrument(
  memberId: string,
  data: MemberInstrumentAdd
): Promise<MemberInstrument> {
  return apiFetch(`/members/${memberId}/instruments`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function removeMemberInstrument(
  memberId: string,
  instrumentId: string
): Promise<void> {
  return apiFetch(`/members/${memberId}/instruments/${instrumentId}`, { method: "DELETE" });
}

export async function assignRole(memberId: string, roleId: string): Promise<void> {
  return apiFetch(`/members/${memberId}/roles`, {
    method: "POST",
    body: JSON.stringify({ role_id: roleId }),
  });
}

export async function removeRole(memberId: string, roleId: string): Promise<void> {
  return apiFetch(`/members/${memberId}/roles/${roleId}`, { method: "DELETE" });
}

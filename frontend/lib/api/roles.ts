import { apiFetch } from "./client";

export interface RoleSummary {
  id: string;
  name: string;
  base_profile: "pastor" | "leadership" | "musician" | "member";
}

export interface Role extends RoleSummary {
  church_id?: string | null;
  is_system: boolean;
}

export async function listRoles(): Promise<{ data: Role[] }> {
  return apiFetch("/roles");
}

export async function createRole(data: { name: string; base_profile: string }): Promise<Role> {
  return apiFetch("/roles", { method: "POST", body: JSON.stringify(data) });
}

export async function updateRole(
  id: string,
  data: { name: string; base_profile: string }
): Promise<Role> {
  return apiFetch(`/roles/${id}`, { method: "PUT", body: JSON.stringify(data) });
}

export async function deleteRole(id: string): Promise<void> {
  return apiFetch(`/roles/${id}`, { method: "DELETE" });
}

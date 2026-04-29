import { apiFetch, ListResponse } from "./client";
import type { MemberSummary } from "./members";

export interface ItemCategory {
  id: string;
  church_id: string;
  name: string;
  icon?: string | null;
}

export interface Item {
  id: string;
  church_id: string;
  category?: ItemCategory | null;
  item_type: "asset" | "consumable";
  name: string;
  description?: string | null;
  asset_number?: string | null;
  photo_url?: string | null;
  location: string;
  status: "available" | "on_loan" | "maintenance" | "damaged";
  quantity: number;
  qty_min_alert?: number | null;
  serial_number?: string | null;
  notes?: string | null;
  deleted_at?: string | null;
  deletion_reason?: "donated" | "discarded" | null;
  created_at: string;
  updated_at: string;
}

export interface ItemCreate {
  item_type: "asset" | "consumable";
  name: string;
  description?: string | null;
  category_id?: string | null;
  asset_number?: string | null;
  location: string;
  quantity?: number;
  qty_min_alert?: number | null;
  serial_number?: string | null;
  notes?: string | null;
}

export interface ItemUpdate {
  name?: string;
  description?: string | null;
  category_id?: string | null;
  location?: string;
  status?: "available" | "on_loan" | "maintenance";
  quantity?: number;
  qty_min_alert?: number | null;
  serial_number?: string | null;
  notes?: string | null;
}

export interface Loan {
  id: string;
  item: Item;
  requested_by: MemberSummary;
  approved_by?: MemberSummary | null;
  loan_to_type: "church" | "member";
  loan_to_id: string;
  loan_to_name: string;
  status: "pending" | "active" | "returned" | "returned_with_issue" | "rejected";
  expected_return_date?: string | null;
  actual_return_date?: string | null;
  return_condition?: "good" | "damaged" | "lost" | null;
  return_notes?: string | null;
  created_at: string;
  returned_at?: string | null;
}

export interface LoanCreate {
  item_id: string;
  loan_to_type: "church" | "member";
  loan_to_id: string;
  expected_return_date?: string | null;
}

export interface LoanReturn {
  return_condition: "good" | "damaged" | "lost";
  return_notes?: string | null;
}

export async function listCategories(): Promise<{ data: ItemCategory[] }> {
  return apiFetch("/inventory/categories");
}

export async function createCategory(data: {
  name: string;
  icon?: string | null;
}): Promise<ItemCategory> {
  return apiFetch("/inventory/categories", { method: "POST", body: JSON.stringify(data) });
}

export async function updateCategory(
  id: string,
  data: { name: string; icon?: string | null }
): Promise<ItemCategory> {
  return apiFetch(`/inventory/categories/${id}`, { method: "PUT", body: JSON.stringify(data) });
}

export async function deleteCategory(id: string): Promise<void> {
  return apiFetch(`/inventory/categories/${id}`, { method: "DELETE" });
}

export async function listItems(params?: {
  page?: number;
  per_page?: number;
  search?: string;
  category_id?: string;
  status?: string;
  item_type?: "asset" | "consumable";
  include_deleted?: boolean;
}): Promise<ListResponse<Item>> {
  const qs = new URLSearchParams();
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  if (params?.search) qs.set("search", params.search);
  if (params?.category_id) qs.set("category_id", params.category_id);
  if (params?.status) qs.set("status", params.status);
  if (params?.item_type) qs.set("item_type", params.item_type);
  if (params?.include_deleted) qs.set("include_deleted", "true");
  return apiFetch(`/inventory/items?${qs}`);
}

export async function createItem(data: ItemCreate): Promise<Item> {
  return apiFetch("/inventory/items", { method: "POST", body: JSON.stringify(data) });
}

export async function getItem(id: string): Promise<Item> {
  return apiFetch(`/inventory/items/${id}`);
}

export async function updateItem(id: string, data: ItemUpdate): Promise<Item> {
  return apiFetch(`/inventory/items/${id}`, { method: "PUT", body: JSON.stringify(data) });
}

export async function uploadItemPhoto(id: string, file: File): Promise<{ photo_url: string }> {
  const form = new FormData();
  form.append("photo", file);
  return apiFetch(`/inventory/items/${id}/photo`, { method: "POST", body: form });
}

export async function discardItem(id: string): Promise<void> {
  return apiFetch(`/inventory/items/${id}/discard`, {
    method: "POST",
    body: JSON.stringify({ deletion_reason: "discarded" }),
  });
}

export async function donateItem(id: string): Promise<void> {
  return apiFetch(`/inventory/items/${id}/donate`, {
    method: "POST",
    body: JSON.stringify({ deletion_reason: "donated" }),
  });
}

export async function listLoans(params?: {
  status?: string;
  page?: number;
  per_page?: number;
}): Promise<ListResponse<Loan>> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set("status", params.status);
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  return apiFetch(`/inventory/loans?${qs}`);
}

export async function createLoan(data: LoanCreate): Promise<Loan> {
  return apiFetch("/inventory/loans", { method: "POST", body: JSON.stringify(data) });
}

export async function getLoan(id: string): Promise<Loan> {
  return apiFetch(`/inventory/loans/${id}`);
}

export async function approveLoan(id: string): Promise<Loan> {
  return apiFetch(`/inventory/loans/${id}/approve`, { method: "POST" });
}

export async function rejectLoan(id: string): Promise<Loan> {
  return apiFetch(`/inventory/loans/${id}/reject`, { method: "POST" });
}

export async function returnLoan(id: string, data: LoanReturn): Promise<Loan> {
  return apiFetch(`/inventory/loans/${id}/return`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

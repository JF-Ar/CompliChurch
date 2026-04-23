import { getAccessToken, setSession, clearSession } from "./auth";

const BASE_URL = process.env.NEXT_PUBLIC_API_URL!;

// ── Types (aligned with openapi.yaml) ────────────────────────────────────────

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

export interface MemberSummary {
  id: string;
  name: string;
  email: string;
  is_active: boolean;
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

export interface RoleSummary {
  id: string;
  name: string;
  base_profile: "pastor" | "leadership" | "musician" | "member";
}

export interface Role extends RoleSummary {
  church_id?: string | null;
  is_system: boolean;
}

export interface MemberInstrument {
  id: string;
  instrument_id: string;
  instrument_name: string;
  is_primary: boolean;
}

export interface Instrument {
  id: string;
  church_id?: string | null;
  name: string;
  is_system: boolean;
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

export interface ApiError {
  error: {
    code: string;
    message: string;
    field?: string | null;
  };
}

export interface LoginResponse {
  access_token: string;
  member: Member;
  church: Church;
}

export interface Schedule {
  id: string;
  church_id: string;
  sunday_date: string;
  status: "draft" | "published" | "cancelled";
  created_by: MemberSummary;
  approved_by?: MemberSummary | null;
  notes?: string | null;
  published_at?: string | null;
  slots: ScheduleSlot[];
  created_at: string;
}

export interface ScheduleSummary {
  id: string;
  sunday_date: string;
  status: "draft" | "published" | "cancelled";
  slot_count: number;
  published_at?: string | null;
}

export interface ScheduleSlot {
  id: string;
  schedule_id: string;
  member: MemberSummary;
  instrument?: Instrument | null;
  function_in_scale: string;
  confirmed: boolean;
  notified_at?: string | null;
}

export interface ScheduleSuggestion {
  sunday_date: string;
  suggested_slots: Array<{
    member_id: string;
    member_name: string;
    instrument_id: string;
    instrument_name: string;
    warning: "consecutive_sunday" | null;
  }>;
  available_members: MemberSummary[];
  unavailable_members: Array<{
    member: MemberSummary;
    reason?: string | null;
  }>;
}

export interface AvailabilityException {
  id: string;
  member_id: string;
  church_id: string;
  unavailable_date: string;
  reason?: string | null;
  created_at: string;
}

export interface EventSummary {
  id: string;
  title: string;
  event_type: "pastoral_visit" | "counseling" | "leadership_meeting" | "block" | "other";
  status: "requested" | "confirmed" | "declined" | "cancelled";
  starts_at: string;
  ends_at: string;
}

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
  status: "available" | "on_loan" | "maintenance";
  quantity: number;
  qty_min_alert?: number | null;
  serial_number?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
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

// ── Internal fetch wrapper ────────────────────────────────────────────────────

async function doRefresh(): Promise<string | null> {
  const res = await fetch(`${BASE_URL}/auth/refresh`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    clearSession();
    return null;
  }
  const data: { access_token: string } = await res.json();
  return data.access_token;
}

async function apiFetch<T>(
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

  return res.json();
}

// ── Auth ──────────────────────────────────────────────────────────────────────

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

export async function logout(): Promise<void> {
  return apiFetch("/auth/logout", { method: "POST" });
}

export async function logoutAll(): Promise<void> {
  return apiFetch("/auth/logout-all", { method: "POST" });
}

// ── Churches ─────────────────────────────────────────────────────────────────

export async function getMyChurch(): Promise<Church> {
  return apiFetch("/churches/me");
}

// ── Members ───────────────────────────────────────────────────────────────────

export async function getMe(): Promise<Member> {
  return apiFetch("/members/me");
}

export async function updateMe(data: MemberUpdate): Promise<Member> {
  return apiFetch("/members/me", {
    method: "PUT",
    body: JSON.stringify(data),
  });
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

// ── Schedules ─────────────────────────────────────────────────────────────────

export async function listSchedules(params?: {
  page?: number;
  per_page?: number;
  status?: "draft" | "published" | "cancelled";
}): Promise<ListResponse<ScheduleSummary>> {
  const qs = new URLSearchParams();
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  if (params?.status) qs.set("status", params.status);
  return apiFetch(`/schedules?${qs}`);
}

export async function getSchedule(id: string): Promise<Schedule> {
  return apiFetch(`/schedules/${id}`);
}

export async function createSchedule(data: {
  sunday_date: string;
  notes?: string | null;
}): Promise<Schedule> {
  return apiFetch("/schedules", { method: "POST", body: JSON.stringify(data) });
}

export async function getScheduleSuggestion(sundayDate: string): Promise<ScheduleSuggestion> {
  return apiFetch(`/schedules/suggest/${sundayDate}`);
}

export async function publishSchedule(id: string): Promise<Schedule> {
  return apiFetch(`/schedules/${id}/publish`, { method: "POST" });
}

export async function addScheduleSlot(
  scheduleId: string,
  data: { member_id: string; instrument_id?: string | null; function_in_scale: string }
): Promise<ScheduleSlot> {
  return apiFetch(`/schedules/${scheduleId}/slots`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function removeScheduleSlot(
  scheduleId: string,
  slotId: string
): Promise<void> {
  return apiFetch(`/schedules/${scheduleId}/slots/${slotId}`, { method: "DELETE" });
}

export async function confirmScheduleSlot(
  scheduleId: string,
  slotId: string
): Promise<ScheduleSlot> {
  return apiFetch(`/schedules/${scheduleId}/slots/${slotId}/confirm`, { method: "POST" });
}

// ── Availability ──────────────────────────────────────────────────────────────

export async function listMyExceptions(month?: string): Promise<{ data: AvailabilityException[] }> {
  const qs = month ? `?month=${month}` : "";
  return apiFetch(`/availability/exceptions${qs}`);
}

export async function createException(data: {
  unavailable_date: string;
  reason?: string | null;
}): Promise<AvailabilityException> {
  return apiFetch("/availability/exceptions", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteException(id: string): Promise<void> {
  return apiFetch(`/availability/exceptions/${id}`, { method: "DELETE" });
}

// ── Agenda (Events) ───────────────────────────────────────────────────────────

export async function listEvents(params?: {
  status?: string;
  from?: string;
  to?: string;
  page?: number;
  per_page?: number;
}): Promise<ListResponse<EventSummary>> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set("status", params.status);
  if (params?.from) qs.set("from", params.from);
  if (params?.to) qs.set("to", params.to);
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  return apiFetch(`/agenda/events?${qs}`);
}

// ── Inventory ─────────────────────────────────────────────────────────────────

export async function listCategories(): Promise<{ data: ItemCategory[] }> {
  return apiFetch("/inventory/categories");
}

export async function listItems(params?: {
  page?: number;
  per_page?: number;
  search?: string;
  category_id?: string;
  status?: string;
  item_type?: "asset" | "consumable";
}): Promise<ListResponse<Item>> {
  const qs = new URLSearchParams();
  if (params?.page) qs.set("page", String(params.page));
  if (params?.per_page) qs.set("per_page", String(params.per_page));
  if (params?.search) qs.set("search", params.search);
  if (params?.category_id) qs.set("category_id", params.category_id);
  if (params?.status) qs.set("status", params.status);
  if (params?.item_type) qs.set("item_type", params.item_type);
  return apiFetch(`/inventory/items?${qs}`);
}

export async function getItem(id: string): Promise<Item> {
  return apiFetch(`/inventory/items/${id}`);
}

export async function uploadItemPhoto(id: string, file: File): Promise<{ photo_url: string }> {
  const form = new FormData();
  form.append("photo", file);
  return apiFetch(`/inventory/items/${id}/photo`, { method: "POST", body: form });
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

// ── Roles & Instruments ────────────────────────────────────────────────────────

export async function listRoles(): Promise<{ data: Role[] }> {
  return apiFetch("/roles");
}

export async function listInstruments(): Promise<{ data: Instrument[] }> {
  return apiFetch("/instruments");
}

// Re-export setSession so callers can store the session after login
export { setSession } from "./auth";

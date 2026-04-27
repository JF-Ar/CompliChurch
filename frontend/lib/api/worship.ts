import { apiFetch, ListResponse } from "./client";
import type { MemberSummary } from "./members";
import type { Instrument } from "./instruments";

export interface ScheduleSlot {
  id: string;
  schedule_id: string;
  member: MemberSummary;
  instrument?: Instrument | null;
  function_in_scale: string;
  confirmed: boolean;
  notified_at?: string | null;
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

export async function removeScheduleSlot(scheduleId: string, slotId: string): Promise<void> {
  return apiFetch(`/schedules/${scheduleId}/slots/${slotId}`, { method: "DELETE" });
}

export async function confirmScheduleSlot(
  scheduleId: string,
  slotId: string
): Promise<ScheduleSlot> {
  return apiFetch(`/schedules/${scheduleId}/slots/${slotId}/confirm`, { method: "POST" });
}

export async function listMyExceptions(
  month?: string
): Promise<{ data: AvailabilityException[] }> {
  const qs = month ? `?month=${month}` : "";
  return apiFetch(`/availability/exceptions${qs}`);
}

export async function createException(data: {
  unavailable_date: string;
  reason?: string | null;
}): Promise<AvailabilityException> {
  return apiFetch("/availability/exceptions", { method: "POST", body: JSON.stringify(data) });
}

export async function deleteException(id: string): Promise<void> {
  return apiFetch(`/availability/exceptions/${id}`, { method: "DELETE" });
}

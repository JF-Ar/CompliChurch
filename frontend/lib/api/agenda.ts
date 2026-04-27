import { apiFetch, ListResponse } from "./client";

export interface EventSummary {
  id: string;
  title: string;
  event_type: "pastoral_visit" | "counseling" | "leadership_meeting" | "block" | "other";
  status: "requested" | "confirmed" | "declined" | "cancelled";
  starts_at: string;
  ends_at: string;
}

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

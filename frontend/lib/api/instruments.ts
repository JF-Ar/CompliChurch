import { apiFetch } from "./client";

export interface Instrument {
  id: string;
  church_id?: string | null;
  name: string;
  is_system: boolean;
}

export async function listInstruments(): Promise<{ data: Instrument[] }> {
  return apiFetch("/instruments");
}

export async function createInstrument(data: { name: string }): Promise<Instrument> {
  return apiFetch("/instruments", { method: "POST", body: JSON.stringify(data) });
}

export async function deleteInstrument(id: string): Promise<void> {
  return apiFetch(`/instruments/${id}`, { method: "DELETE" });
}

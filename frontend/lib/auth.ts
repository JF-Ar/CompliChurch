import type { Member, Church } from "./api";

let accessToken: string | null = null;
let currentMember: Member | null = null;
let currentChurch: Church | null = null;

export function getAccessToken(): string | null {
  return accessToken;
}

export function setSession(token: string, member: Member, church: Church): void {
  accessToken = token;
  currentMember = member;
  currentChurch = church;
}

export function getSession(): { member: Member; church: Church } | null {
  if (!accessToken || !currentMember || !currentChurch) return null;
  return { member: currentMember, church: currentChurch };
}

export function setAccessToken(token: string): void {
  accessToken = token;
}

export function clearSession(): void {
  accessToken = null;
  currentMember = null;
  currentChurch = null;
}

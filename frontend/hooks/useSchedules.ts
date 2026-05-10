"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listSchedules,
  getSchedule,
  createSchedule,
  publishSchedule,
  addScheduleSlot,
  removeScheduleSlot,
  confirmScheduleSlot,
  getScheduleSuggestion,
  listMyExceptions,
  createException,
  deleteException,
  listAllExceptions,
} from "@/lib/api";

export const scheduleKeys = {
  all: ["schedules"] as const,
  list: (params?: object) => ["schedules", "list", params] as const,
  detail: (id: string) => ["schedules", id] as const,
  slots: (id: string) => ["schedules", id, "slots"] as const,
  suggestion: (date: string) => ["schedules", "suggest", date] as const,
};

export const exceptionKeys = {
  all: ["exceptions"] as const,
  mine: (month?: string) => ["exceptions", "mine", month] as const,
  church: (month: string) => ["exceptions", "church", month] as const,
};

export function useSchedules(params?: {
  page?: number;
  per_page?: number;
  status?: "draft" | "published" | "cancelled";
}) {
  return useQuery({
    queryKey: scheduleKeys.list(params),
    queryFn: () => listSchedules(params),
  });
}

export function useSchedule(id: string) {
  return useQuery({
    queryKey: scheduleKeys.detail(id),
    queryFn: () => getSchedule(id),
    enabled: !!id,
  });
}

export function useCreateSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createSchedule,
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduleKeys.all }),
  });
}

export function usePublishSchedule(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => publishSchedule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduleKeys.detail(id) }),
  });
}

export function useAddSlot(scheduleId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      member_id: string;
      instrument_id?: string | null;
      function_in_scale: string;
    }) => addScheduleSlot(scheduleId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: scheduleKeys.slots(scheduleId) });
      qc.invalidateQueries({ queryKey: scheduleKeys.detail(scheduleId) });
    },
  });
}

export function useRemoveSlot(scheduleId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slotId: string) => removeScheduleSlot(scheduleId, slotId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: scheduleKeys.slots(scheduleId) });
      qc.invalidateQueries({ queryKey: scheduleKeys.detail(scheduleId) });
    },
  });
}

export function useConfirmSlot(scheduleId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slotId: string) => confirmScheduleSlot(scheduleId, slotId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: scheduleKeys.slots(scheduleId) });
      qc.invalidateQueries({ queryKey: scheduleKeys.detail(scheduleId) });
    },
  });
}

export function useScheduleSuggestion(date: string) {
  return useQuery({
    queryKey: scheduleKeys.suggestion(date),
    queryFn: () => getScheduleSuggestion(date),
    enabled: !!date,
  });
}

export function useMyExceptions(month?: string) {
  return useQuery({
    queryKey: exceptionKeys.mine(month),
    queryFn: () => listMyExceptions(month),
  });
}

export function useAddException() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createException,
    onSuccess: () => qc.invalidateQueries({ queryKey: exceptionKeys.all }),
  });
}

export function useRemoveException() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteException(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: exceptionKeys.all }),
  });
}

export function useAllExceptions(month: string) {
  return useQuery({
    queryKey: exceptionKeys.church(month),
    queryFn: () => listAllExceptions(month),
    enabled: !!month,
  });
}

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listMembers,
  getMember,
  createMember,
  updateMember,
  deactivateMember,
  listRoles,
  assignRole,
  removeRole,
} from "@/lib/api";
import type { MemberCreate, MemberUpdate } from "@/lib/api";

export const memberKeys = {
  all: ["members"] as const,
  list: (params?: object) => ["members", "list", params] as const,
  detail: (id: string) => ["members", id] as const,
};

export const roleKeys = {
  all: ["roles"] as const,
};

export function useMembers(params?: {
  page?: number;
  per_page?: number;
  search?: string;
  role?: string;
  is_active?: boolean;
}) {
  return useQuery({
    queryKey: memberKeys.list(params),
    queryFn: () => listMembers(params),
  });
}

export function useMember(id: string) {
  return useQuery({
    queryKey: memberKeys.detail(id),
    queryFn: () => getMember(id),
    enabled: !!id,
  });
}

export function useRoles() {
  return useQuery({
    queryKey: roleKeys.all,
    queryFn: listRoles,
  });
}

export function useCreateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: MemberCreate) => createMember(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}

export function useUpdateMember(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: MemberUpdate) => updateMember(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: memberKeys.detail(id) });
      qc.invalidateQueries({ queryKey: memberKeys.all });
    },
  });
}

export function useDeactivateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deactivateMember,
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}

export function useAssignRole(memberId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (roleId: string) => assignRole(memberId, roleId),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.detail(memberId) }),
  });
}

export function useRemoveRole(memberId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (roleId: string) => removeRole(memberId, roleId),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.detail(memberId) }),
  });
}

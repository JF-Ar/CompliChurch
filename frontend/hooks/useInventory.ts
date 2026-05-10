"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  listCategories,
  createCategory,
  updateCategory,
  deleteCategory,
  listItems,
  createItem,
  getItem,
  updateItem,
  uploadItemPhoto,
  discardItem,
  donateItem,
  importItems,
  listLoans,
  createLoan,
  getLoan,
  approveLoan,
  rejectLoan,
  returnLoan,
  listCongregations,
} from "@/lib/api";
import type { ApiError } from "@/lib/api/client";
import type { ItemCreate, ItemUpdate, LoanCreate, LoanReturn } from "@/lib/api";

export const categoryKeys = {
  all: ["inventory", "categories"] as const,
};

export const itemKeys = {
  all: ["inventory", "items"] as const,
  list: (params?: object) => ["inventory", "items", "list", params] as const,
  detail: (id: string) => ["inventory", "items", id] as const,
};

export const loanKeys = {
  all: ["inventory", "loans"] as const,
  list: (params?: object) => ["inventory", "loans", "list", params] as const,
  detail: (id: string) => ["inventory", "loans", id] as const,
};

export const congregationKeys = {
  all: ["churches", "congregations"] as const,
};

// ── Categories ────────────────────────────────────────────────────────────────

export function useCategories() {
  return useQuery({
    queryKey: categoryKeys.all,
    queryFn: listCategories,
  });
}

export function useCreateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createCategory,
    onSuccess: () => qc.invalidateQueries({ queryKey: categoryKeys.all }),
  });
}

export function useUpdateCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name: string; icon?: string | null } }) =>
      updateCategory(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: categoryKeys.all }),
  });
}

export function useDeleteCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deleteCategory,
    onSuccess: () => qc.invalidateQueries({ queryKey: categoryKeys.all }),
  });
}

// ── Items ─────────────────────────────────────────────────────────────────────

export function useItems(params?: {
  page?: number;
  per_page?: number;
  search?: string;
  category_id?: string;
  status?: string;
  item_type?: "asset" | "consumable";
  include_deleted?: boolean;
}) {
  return useQuery({
    queryKey: itemKeys.list(params),
    queryFn: () => listItems(params),
  });
}

export function useItem(id: string) {
  return useQuery({
    queryKey: itemKeys.detail(id),
    queryFn: () => getItem(id),
    enabled: !!id,
  });
}

export function useCreateItem() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ItemCreate) => createItem(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: itemKeys.all }),
  });
}

export function useUpdateItem(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ItemUpdate) => updateItem(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: itemKeys.detail(id) });
      qc.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

export function useUploadItemPhoto(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => uploadItemPhoto(id, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: itemKeys.detail(id) }),
  });
}

export function useDiscardItem(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => discardItem(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: itemKeys.detail(id) });
      qc.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

export function useDonateItem(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => donateItem(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: itemKeys.detail(id) });
      qc.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

// ── Loans ─────────────────────────────────────────────────────────────────────

export function useLoans(params?: {
  status?: string;
  page?: number;
  per_page?: number;
}) {
  return useQuery({
    queryKey: loanKeys.list(params),
    queryFn: () => listLoans(params),
  });
}

export function useLoan(id: string) {
  return useQuery({
    queryKey: loanKeys.detail(id),
    queryFn: () => getLoan(id),
    enabled: !!id,
  });
}

export function useCreateLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: LoanCreate) => createLoan(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useApproveLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => approveLoan(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useRejectLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => rejectLoan(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useReturnLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: LoanReturn }) => returnLoan(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useImportItems() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => importItems(file),
    onSuccess: () => qc.invalidateQueries({ queryKey: itemKeys.all }),
    onError: (err) => {
      const e = err as unknown as ApiError;
      toast.error(e?.error?.message ?? "Erro ao importar planilha. Tente novamente.");
    },
  });
}

// ── Congregations (used in loan form) ────────────────────────────────────────

export function useCongregations() {
  return useQuery({
    queryKey: congregationKeys.all,
    queryFn: listCongregations,
  });
}

"use client";

import { useState } from "react";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { useLoans, useApproveLoan, useRejectLoan, useReturnLoan } from "@/hooks/useInventory";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import type { ApiError, Loan } from "@/lib/api";

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDate(date: string | null | undefined): string {
  if (!date) return "—";
  const [year, month, day] = date.split("-");
  return `${day}/${month}/${year}`;
}

const LOAN_STATUS_LABELS: Record<string, string> = {
  pending: "Pendente",
  active: "Ativo",
  returned: "Devolvido",
  returned_with_issue: "Devolvido c/ problema",
  rejected: "Rejeitado",
};

const LOAN_STATUS_VARIANTS: Record<string, "warning" | "success" | "muted" | "destructive"> = {
  pending: "warning",
  active: "success",
  returned: "muted",
  returned_with_issue: "destructive",
  rejected: "destructive",
};

// ── Return schema ─────────────────────────────────────────────────────────────

const returnSchema = z.object({
  return_condition: z.enum(["good", "damaged", "lost"]),
  return_notes: z.string().optional(),
});

type ReturnValues = z.infer<typeof returnSchema>;

// ── Page ─────────────────────────────────────────────────────────────────────

export default function LoansPage() {
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(1);
  const [confirmRejectId, setConfirmRejectId] = useState<string | null>(null);
  const [returnLoan, setReturnLoan] = useState<Loan | null>(null);
  const [returnError, setReturnError] = useState<string | null>(null);

  const { data, isLoading, error } = useLoans({
    status: statusFilter || undefined,
    page,
    per_page: 20,
  });

  const { mutateAsync: approveLoan, isPending: isApproving } = useApproveLoan();
  const { mutateAsync: rejectLoan, isPending: isRejecting } = useRejectLoan();
  const { mutateAsync: doReturnLoan, isPending: isReturning } = useReturnLoan();

  const loans = data?.data ?? [];
  const meta = data?.meta;
  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 1;

  const returnForm = useForm<ReturnValues>({
    resolver: zodResolver(returnSchema),
    defaultValues: { return_condition: "good", return_notes: "" },
  });

  async function handleApprove(id: string) {
    try {
      await approveLoan(id);
      toast.success("Empréstimo aprovado.");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível aprovar. Tente novamente.");
    }
  }

  async function handleReject(id: string) {
    try {
      await rejectLoan(id);
      toast.success("Empréstimo rejeitado.");
      setConfirmRejectId(null);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível rejeitar. Tente novamente.");
    }
  }

  async function onReturnSubmit(values: ReturnValues) {
    if (!returnLoan) return;
    setReturnError(null);
    try {
      await doReturnLoan({
        id: returnLoan.id,
        data: {
          return_condition: values.return_condition,
          return_notes: values.return_notes || null,
        },
      });
      toast.success("Devolução registrada com sucesso.");
      setReturnLoan(null);
      returnForm.reset({ return_condition: "good", return_notes: "" });
    } catch (err) {
      const e = err as ApiError;
      setReturnError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-3">
          <Link href="/inventory" className="text-sm text-muted-foreground hover:text-foreground">
            ← Patrimônio
          </Link>
          <h1 className="text-xl font-semibold">Empréstimos</h1>
        </div>
      </div>

      {/* Status filter */}
      <select
        aria-label="Filtrar por status"
        className="h-10 self-start min-w-[160px] rounded-md border border-input bg-background px-3 py-2 text-sm"
        value={statusFilter}
        onChange={(e) => {
          setStatusFilter(e.target.value);
          setPage(1);
        }}
      >
        <option value="">Todos os status</option>
        <option value="pending">Pendente</option>
        <option value="active">Ativo</option>
        <option value="returned">Devolvido</option>
        <option value="returned_with_issue">Devolvido c/ problema</option>
        <option value="rejected">Rejeitado</option>
      </select>

      {/* Loading */}
      {isLoading && (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Error */}
      {error && !isLoading && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar os empréstimos. Tente novamente.
        </div>
      )}

      {/* Empty */}
      {!isLoading && !error && loans.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <span className="text-5xl">🤝</span>
          <p className="text-sm text-muted-foreground">
            {statusFilter
              ? "Nenhum empréstimo com esse status."
              : "Nenhum empréstimo registrado ainda."}
          </p>
        </div>
      )}

      {/* Loans list — cards on mobile, table on desktop */}
      {!isLoading && !error && loans.length > 0 && (
        <>
          {/* Mobile cards */}
          <div className="flex flex-col gap-3 md:hidden">
            {loans.map((loan) => (
              <LoanCard
                key={loan.id}
                loan={loan}
                onApprove={handleApprove}
                onReject={(id) => setConfirmRejectId(id)}
                onReturn={(l) => {
                  returnForm.reset({ return_condition: "good", return_notes: "" });
                  setReturnError(null);
                  setReturnLoan(l);
                }}
                isApproving={isApproving}
              />
            ))}
          </div>

          {/* Desktop table */}
          <div className="hidden md:block overflow-x-auto rounded-lg border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="text-left px-4 py-3 font-medium">Item</th>
                  <th className="text-left px-4 py-3 font-medium">Solicitado por</th>
                  <th className="text-left px-4 py-3 font-medium">Destinatário</th>
                  <th className="text-left px-4 py-3 font-medium">Status</th>
                  <th className="text-left px-4 py-3 font-medium">Devolução prevista</th>
                  <th className="text-left px-4 py-3 font-medium">Ações</th>
                </tr>
              </thead>
              <tbody>
                {loans.map((loan) => (
                  <tr key={loan.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <Link
                        href={`/inventory/${loan.item.id}`}
                        className="font-medium hover:underline"
                      >
                        {loan.item.name}
                      </Link>
                      {loan.item.asset_number && (
                        <div className="text-xs text-muted-foreground">
                          {loan.item.asset_number}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{loan.requested_by.name}</td>
                    <td className="px-4 py-3">{loan.loan_to_name}</td>
                    <td className="px-4 py-3">
                      <Badge variant={LOAN_STATUS_VARIANTS[loan.status]}>
                        {LOAN_STATUS_LABELS[loan.status]}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {formatDate(loan.expected_return_date)}
                    </td>
                    <td className="px-4 py-3">
                      <LoanActions
                        loan={loan}
                        onApprove={handleApprove}
                        onReject={(id) => setConfirmRejectId(id)}
                        onReturn={(l) => {
                          returnForm.reset({ return_condition: "good", return_notes: "" });
                          setReturnError(null);
                          setReturnLoan(l);
                        }}
                        isApproving={isApproving}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Pagination */}
      {meta && meta.total > meta.per_page && (
        <div className="flex items-center justify-between pt-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Anterior
          </Button>
          <span className="text-xs text-muted-foreground">
            Página {page} de {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Próxima
          </Button>
        </div>
      )}

      {/* Reject confirmation dialog */}
      <Dialog open={!!confirmRejectId} onOpenChange={() => setConfirmRejectId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rejeitar empréstimo?</DialogTitle>
            <DialogDescription>
              O solicitante será notificado. Esta ação não pode ser desfeita.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button
              variant="destructive"
              disabled={isRejecting}
              onClick={() => confirmRejectId && handleReject(confirmRejectId)}
            >
              {isRejecting ? "Rejeitando…" : "Rejeitar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Return modal */}
      <Dialog open={!!returnLoan} onOpenChange={(open) => !open && setReturnLoan(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Registrar devolução</DialogTitle>
            <DialogDescription>
              {returnLoan && (
                <>
                  Registre a devolução de <strong>{returnLoan.item.name}</strong> por{" "}
                  <strong>{returnLoan.loan_to_name}</strong>.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={returnForm.handleSubmit(onReturnSubmit)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="return-condition">Condição de devolução *</Label>
              <select
                id="return-condition"
                {...returnForm.register("return_condition")}
                className="h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="good">Bom estado</option>
                <option value="damaged">Danificado</option>
                <option value="lost">Perdido</option>
              </select>
              {returnForm.formState.errors.return_condition && (
                <p className="text-xs text-destructive">
                  {returnForm.formState.errors.return_condition.message}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="return-notes">Observações</Label>
              <textarea
                id="return-notes"
                {...returnForm.register("return_notes")}
                rows={3}
                className="rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>

            {returnError && (
              <p className="text-sm text-destructive text-center">{returnError}</p>
            )}

            <DialogFooter>
              <DialogClose asChild>
                <Button type="button" variant="outline">
                  Cancelar
                </Button>
              </DialogClose>
              <Button type="submit" disabled={isReturning}>
                {isReturning ? "Registrando…" : "Confirmar devolução"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ── Loan action buttons ───────────────────────────────────────────────────────

function LoanActions({
  loan,
  onApprove,
  onReject,
  onReturn,
  isApproving,
}: {
  loan: Loan;
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
  onReturn: (loan: Loan) => void;
  isApproving: boolean;
}) {
  if (loan.status === "pending") {
    return (
      <div className="flex gap-2">
        <Button
          size="sm"
          disabled={isApproving}
          onClick={() => onApprove(loan.id)}
        >
          {isApproving ? "…" : "Aprovar"}
        </Button>
        <Button size="sm" variant="outline" onClick={() => onReject(loan.id)}>
          Rejeitar
        </Button>
      </div>
    );
  }
  if (loan.status === "active") {
    return (
      <Button size="sm" variant="outline" onClick={() => onReturn(loan)}>
        Registrar devolução
      </Button>
    );
  }
  return <span className="text-xs text-muted-foreground">—</span>;
}

// ── Mobile loan card ──────────────────────────────────────────────────────────

function LoanCard({
  loan,
  onApprove,
  onReject,
  onReturn,
  isApproving,
}: {
  loan: Loan;
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
  onReturn: (loan: Loan) => void;
  isApproving: boolean;
}) {
  return (
    <div className="rounded-lg border p-4 flex flex-col gap-2">
      <div className="flex items-start justify-between gap-2">
        <Link href={`/inventory/${loan.item.id}`} className="text-sm font-medium hover:underline">
          {loan.item.name}
        </Link>
        <Badge variant={LOAN_STATUS_VARIANTS[loan.status]}>
          {LOAN_STATUS_LABELS[loan.status]}
        </Badge>
      </div>
      <div className="flex flex-col gap-0.5 text-xs text-muted-foreground">
        <span>Destinatário: {loan.loan_to_name}</span>
        <span>Solicitado por: {loan.requested_by.name}</span>
        {loan.expected_return_date && (
          <span>Devolução prevista: {formatDate(loan.expected_return_date)}</span>
        )}
      </div>
      <LoanActions
        loan={loan}
        onApprove={onApprove}
        onReject={onReject}
        onReturn={onReturn}
        isApproving={isApproving}
      />
    </div>
  );
}

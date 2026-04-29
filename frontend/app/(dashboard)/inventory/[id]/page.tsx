"use client";

import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import type { Resolver } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import Link from "next/link";
import { useMe, useMembers } from "@/hooks/useMembers";
import {
  useItem,
  useUpdateItem,
  useUploadItemPhoto,
  useDiscardItem,
  useDonateItem,
  useLoans,
  useCreateLoan,
  useReturnLoan,
  useCongregations,
  itemKeys,
} from "@/hooks/useInventory";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import type { ApiError } from "@/lib/api";

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDate(date: string | null | undefined): string {
  if (!date) return "—";
  const [year, month, day] = date.split("-");
  return `${day}/${month}/${year}`;
}

function formatTimestamp(ts: string): string {
  return new Date(ts).toLocaleDateString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const STATUS_LABELS: Record<string, string> = {
  available: "Disponível",
  on_loan: "Emprestado",
  damaged: "Com dano",
  maintenance: "Em manutenção",
};

const STATUS_VARIANTS: Record<string, "success" | "warning" | "orange" | "destructive"> = {
  available: "success",
  on_loan: "warning",
  damaged: "orange",
  maintenance: "destructive",
};

function getItemBadge(item: { status: string; deletion_reason?: "donated" | "discarded" | null }) {
  if (item.deletion_reason === "donated") return { label: "Doado", variant: "muted" as const };
  if (item.deletion_reason === "discarded") return { label: "Descartado", variant: "destructive" as const };
  return { label: STATUS_LABELS[item.status] ?? item.status, variant: STATUS_VARIANTS[item.status] ?? "muted" as const };
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
  returned: "success",
  returned_with_issue: "destructive",
  rejected: "destructive",
};

const RETURN_CONDITION_LABELS: Record<string, string> = {
  good: "Bom estado",
  damaged: "Com dano",
  lost: "Perdido",
};

// ── Schemas ───────────────────────────────────────────────────────────────────

const editSchema = z.object({
  name: z.string().min(1, "Nome é obrigatório"),
  location: z.string().min(1, "Localização é obrigatória"),
  status: z.enum(["available", "on_loan", "maintenance", "damaged"]),
  description: z.string().optional(),
  notes: z.string().optional(),
  qty_min_alert: z.coerce.number().int().min(0).nullable().optional(),
});

const loanSchema = z.object({
  loan_to_type: z.enum(["church", "member"]),
  loan_to_id: z.string().min(1, "Selecione o destinatário"),
  expected_return_date: z.string().optional(),
});

const returnSchema = z.object({
  return_condition: z.enum(["good", "damaged", "lost"]),
  return_notes: z.string().optional(),
});

type EditValues = z.infer<typeof editSchema>;
type LoanValues = z.infer<typeof loanSchema>;
type ReturnValues = z.infer<typeof returnSchema>;

// ── Page ─────────────────────────────────────────────────────────────────────

export default function InventoryItemPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const [showEdit, setShowEdit] = useState(false);
  const [showDiscardDialog, setShowDiscardDialog] = useState(false);
  const [showDonateDialog, setShowDonateDialog] = useState(false);
  const [showLoanModal, setShowLoanModal] = useState(false);
  const [showReturnModal, setShowReturnModal] = useState(false);
  const [activeLoanId, setActiveLoanId] = useState<string | null>(null);
  const [editError, setEditError] = useState<string | null>(null);
  const [loanError, setLoanError] = useState<string | null>(null);
  const [returnError, setReturnError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: meData } = useMe();
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

  const { data: item, isLoading, error } = useItem(id);
  const { mutateAsync: updateItem, isPending: isUpdating } = useUpdateItem(id);
  const { mutateAsync: uploadPhoto, isPending: isUploading } = useUploadItemPhoto(id);
  const { mutateAsync: discardItem, isPending: isDiscarding } = useDiscardItem(id);
  const { mutateAsync: donateItem, isPending: isDonating } = useDonateItem(id);
  const { mutateAsync: createLoan, isPending: isCreatingLoan } = useCreateLoan();
  const { mutateAsync: doReturnLoan, isPending: isReturning } = useReturnLoan();
  const queryClient = useQueryClient();

  // Fetch all loans and filter for this item client-side
  const { data: loansData } = useLoans({ per_page: 100 });
  const itemLoans = loansData?.data.filter((l) => l.item.id === id) ?? [];

  // Member and congregation data for the loan form
  const { data: membersData } = useMembers({ per_page: 200 });
  const { data: congregationsData } = useCongregations();

  // ── Edit form ───────────────────────────────────────────────────────────────

  const editForm = useForm<EditValues>({
    resolver: zodResolver(editSchema) as Resolver<EditValues>,
    defaultValues: {
      name: item?.name ?? "",
      location: item?.location ?? "",
      status: item?.status ?? "available",
      description: item?.description ?? "",
      notes: item?.notes ?? "",
      qty_min_alert: item?.qty_min_alert ?? null,
    },
  });

  async function onEditSubmit(values: EditValues) {
    setEditError(null);
    try {
      await updateItem({
        name: values.name,
        location: values.location,
        status: values.status,
        description: values.description || null,
        notes: values.notes || null,
        qty_min_alert: values.qty_min_alert ?? null,
      });
      toast.success("Item atualizado com sucesso.");
      setShowEdit(false);
    } catch (err) {
      const e = err as ApiError;
      setEditError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
    }
  }

  // ── Photo upload ────────────────────────────────────────────────────────────

  async function handlePhotoChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await uploadPhoto(file);
      toast.success("Foto atualizada com sucesso.");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível fazer o upload. Tente novamente.");
    }
    // Reset input so same file can be re-selected
    e.target.value = "";
  }

  // ── Discard / Donate ────────────────────────────────────────────────────────

  async function handleDiscard() {
    try {
      await discardItem();
      toast.success("Item descartado.");
      setShowDiscardDialog(false);
      router.push("/inventory");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível descartar. Tente novamente.");
    }
  }

  async function handleDonate() {
    try {
      await donateItem();
      toast.success("Item marcado como doado.");
      setShowDonateDialog(false);
      router.push("/inventory");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível registrar a doação. Tente novamente.");
    }
  }

  // ── Loan form ───────────────────────────────────────────────────────────────

  const loanForm = useForm<LoanValues>({
    resolver: zodResolver(loanSchema),
    defaultValues: { loan_to_type: "member", loan_to_id: "", expected_return_date: "" },
  });

  const returnForm = useForm<ReturnValues>({
    resolver: zodResolver(returnSchema),
    defaultValues: { return_condition: "good", return_notes: "" },
  });

  const loanToType = loanForm.watch("loan_to_type");

  async function onLoanSubmit(values: LoanValues) {
    setLoanError(null);
    try {
      await createLoan({
        item_id: id,
        loan_to_type: values.loan_to_type,
        loan_to_id: values.loan_to_id,
        expected_return_date: values.expected_return_date || null,
      });
      toast.success("Empréstimo registrado com sucesso.");
      setShowLoanModal(false);
      loanForm.reset({ loan_to_type: "member", loan_to_id: "", expected_return_date: "" });
    } catch (err) {
      const e = err as ApiError;
      setLoanError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
    }
  }

  async function onReturnSubmit(values: ReturnValues) {
    if (!activeLoanId) return;
    setReturnError(null);
    try {
      await doReturnLoan({
        id: activeLoanId,
        data: {
          return_condition: values.return_condition,
          return_notes: values.return_notes || null,
        },
      });
      toast.success("Devolução registrada com sucesso.");
      setShowReturnModal(false);
      setActiveLoanId(null);
      returnForm.reset({ return_condition: "good", return_notes: "" });
      queryClient.invalidateQueries({ queryKey: itemKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: itemKeys.all });
    } catch (err) {
      const e = err as ApiError;
      setReturnError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 p-4 max-w-2xl mx-auto">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-48 w-full rounded-lg" />
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </div>
    );
  }

  if (error || !item) {
    return (
      <div className="flex flex-col gap-4 p-4 max-w-2xl mx-auto">
        <Link href="/inventory" className="text-sm text-muted-foreground hover:text-foreground">
          ← Patrimônio
        </Link>
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Item não encontrado ou não foi possível carregar.
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
      <Link href="/inventory" className="text-sm text-muted-foreground hover:text-foreground w-fit">
        ← Patrimônio
      </Link>

      {/* Photo */}
      <div className="relative w-full rounded-lg overflow-hidden bg-muted aspect-video flex items-center justify-center">
        {item.photo_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={item.photo_url}
            alt={item.name}
            className="w-full h-full object-cover"
          />
        ) : (
          <span className="text-6xl" aria-hidden="true">📦</span>
        )}
        {isLeadership && (
          <div className="absolute bottom-2 right-2">
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handlePhotoChange}
            />
            <Button
              size="sm"
              variant="outline"
              className="bg-background/90"
              disabled={isUploading}
              onClick={() => fileInputRef.current?.click()}
            >
              {isUploading ? "Enviando…" : item.photo_url ? "Trocar foto" : "Adicionar foto"}
            </Button>
          </div>
        )}
      </div>

      {/* Header */}
      <div className="flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <h1 className="text-xl font-semibold">{item.name}</h1>
          <Badge variant={getItemBadge(item).variant}>{getItemBadge(item).label}</Badge>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {item.asset_number && (
            <Badge variant="outline">{item.asset_number}</Badge>
          )}
          {item.category && (
            <Badge variant="secondary">{item.category.name}</Badge>
          )}
          <Badge variant="outline">
            {item.item_type === "asset" ? "Bem permanente" : "Consumível"}
          </Badge>
        </div>
      </div>

      {/* Details */}
      {!showEdit && (
        <div className="flex flex-col gap-3 rounded-lg border p-4">
          <Row label="Localização" value={item.location} />
          {item.description && <Row label="Descrição" value={item.description} />}
          <Row label="Quantidade" value={String(item.quantity)} />
          {item.qty_min_alert != null && (
            <Row label="Alerta de estoque mínimo" value={String(item.qty_min_alert)} />
          )}
          {item.serial_number && <Row label="Número de série" value={item.serial_number} />}
          {item.notes && <Row label="Observações" value={item.notes} />}
          <Row label="Cadastrado em" value={formatTimestamp(item.created_at)} />
          <Row label="Atualizado em" value={formatTimestamp(item.updated_at)} />
        </div>
      )}

      {/* Edit form (inline) */}
      {showEdit && (
        <form
          onSubmit={editForm.handleSubmit(onEditSubmit)}
          className="flex flex-col gap-4 rounded-lg border p-4"
        >
          <p className="text-sm font-semibold">Editar item</p>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-name">Nome *</Label>
            <Input
              id="edit-name"
              defaultValue={item.name}
              {...editForm.register("name")}
            />
            {editForm.formState.errors.name && (
              <p className="text-xs text-destructive">
                {editForm.formState.errors.name.message}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-location">Localização *</Label>
            <Input
              id="edit-location"
              defaultValue={item.location}
              {...editForm.register("location")}
            />
            {editForm.formState.errors.location && (
              <p className="text-xs text-destructive">
                {editForm.formState.errors.location.message}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-status">Status</Label>
            <select
              id="edit-status"
              defaultValue={item.status}
              {...editForm.register("status")}
              className="h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="available">Disponível</option>
              <option value="on_loan">Emprestado</option>
              <option value="damaged">Com dano</option>
              <option value="maintenance">Em manutenção</option>
            </select>
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-description">Descrição</Label>
            <textarea
              id="edit-description"
              defaultValue={item.description ?? ""}
              {...editForm.register("description")}
              rows={3}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-notes">Observações</Label>
            <textarea
              id="edit-notes"
              defaultValue={item.notes ?? ""}
              {...editForm.register("notes")}
              rows={3}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          {item.item_type === "consumable" && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="edit-qty-alert">Alerta de quantidade mínima</Label>
              <Input
                id="edit-qty-alert"
                type="number"
                min={0}
                defaultValue={item.qty_min_alert ?? ""}
                {...editForm.register("qty_min_alert")}
              />
            </div>
          )}

          {editError && (
            <p className="text-sm text-destructive text-center">{editError}</p>
          )}

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              className="flex-1"
              onClick={() => {
                setShowEdit(false);
                editForm.reset();
                setEditError(null);
              }}
            >
              Cancelar
            </Button>
            <Button
              type="submit"
              className="flex-1"
              disabled={isUpdating}
            >
              {isUpdating ? "Salvando…" : "Salvar"}
            </Button>
          </div>
        </form>
      )}

      {/* Leadership actions */}
      {isLeadership && !showEdit && (
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={() => {
              editForm.reset({
                name: item.name,
                location: item.location,
                status: item.status,
                description: item.description ?? "",
                notes: item.notes ?? "",
                qty_min_alert: item.qty_min_alert ?? null,
              });
              setEditError(null);
              setShowEdit(true);
            }}
          >
            Editar
          </Button>
          <Button
            variant="outline"
            className="text-destructive hover:text-destructive"
            onClick={() => setShowDiscardDialog(true)}
          >
            Descartar
          </Button>
          <Button
            variant="outline"
            onClick={() => setShowDonateDialog(true)}
          >
            Registrar doação
          </Button>
        </div>
      )}

      {/* Loans section */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Empréstimos</h2>
          {isLeadership && (
            <Button size="sm" onClick={() => setShowLoanModal(true)}>
              + Novo empréstimo
            </Button>
          )}
        </div>

        {itemLoans.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            Nenhum empréstimo registrado para este item.
          </p>
        ) : (
          <div className="flex flex-col gap-2">
            {itemLoans.map((loan) => {
              const canReturn =
                loan.status === "active" &&
                (meData?.id === loan.requested_by.id || isLeadership);
              const isReturned =
                loan.status === "returned" || loan.status === "returned_with_issue";
              return (
                <div key={loan.id} className="rounded-lg border p-3 flex flex-col gap-1.5">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-medium">{loan.loan_to_name}</span>
                    <Badge variant={LOAN_STATUS_VARIANTS[loan.status]}>
                      {LOAN_STATUS_LABELS[loan.status]}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    Solicitado por: {loan.requested_by.name}
                  </span>
                  {loan.expected_return_date && (
                    <span className="text-xs text-muted-foreground">
                      Devolução prevista: {formatDate(loan.expected_return_date)}
                    </span>
                  )}
                  {isReturned && loan.return_condition && (
                    <span className="text-xs text-muted-foreground">
                      Condição: {RETURN_CONDITION_LABELS[loan.return_condition] ?? loan.return_condition}
                    </span>
                  )}
                  {isReturned && loan.return_notes && (
                    <span className="text-xs text-muted-foreground">
                      Obs: {loan.return_notes}
                    </span>
                  )}
                  {isReturned && loan.actual_return_date && (
                    <span className="text-xs text-muted-foreground">
                      Devolvido em: {formatDate(loan.actual_return_date)}
                    </span>
                  )}
                  {canReturn && (
                    <div className="pt-1">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setActiveLoanId(loan.id);
                          returnForm.reset({ return_condition: "good", return_notes: "" });
                          setReturnError(null);
                          setShowReturnModal(true);
                        }}
                      >
                        Registrar devolução
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Discard dialog */}
      <Dialog open={showDiscardDialog} onOpenChange={setShowDiscardDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Descartar {item.name}?</DialogTitle>
            <DialogDescription>
              O item será marcado como descartado e removido do inventário ativo.
              Esta ação não pode ser desfeita.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button
              variant="destructive"
              disabled={isDiscarding}
              onClick={handleDiscard}
            >
              {isDiscarding ? "Descartando…" : "Descartar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Donate dialog */}
      <Dialog open={showDonateDialog} onOpenChange={setShowDonateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Registrar doação de {item.name}?</DialogTitle>
            <DialogDescription>
              O item será marcado como doado e removido do inventário ativo.
              Esta ação não pode ser desfeita.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button
              disabled={isDonating}
              onClick={handleDonate}
            >
              {isDonating ? "Registrando…" : "Confirmar doação"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New loan modal */}
      <Dialog open={showLoanModal} onOpenChange={setShowLoanModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Novo empréstimo</DialogTitle>
            <DialogDescription>
              Registre a saída de <strong>{item.name}</strong>.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={loanForm.handleSubmit(onLoanSubmit)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label>Tipo de destinatário</Label>
              <div className="flex gap-4">
                <label className="flex items-center gap-2 cursor-pointer min-h-[48px]">
                  <input
                    type="radio"
                    value="member"
                    {...loanForm.register("loan_to_type")}
                    className="h-4 w-4"
                  />
                  <span className="text-sm">Membro</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer min-h-[48px]">
                  <input
                    type="radio"
                    value="church"
                    {...loanForm.register("loan_to_type")}
                    className="h-4 w-4"
                  />
                  <span className="text-sm">Igreja / Congregação</span>
                </label>
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="loan-to-id">
                {loanToType === "member" ? "Membro" : "Congregação"} *
              </Label>
              {loanToType === "member" ? (
                <select
                  id="loan-to-id"
                  {...loanForm.register("loan_to_id")}
                  className="h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  <option value="">Selecione um membro…</option>
                  {membersData?.data.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name}
                    </option>
                  ))}
                </select>
              ) : (
                <select
                  id="loan-to-id"
                  {...loanForm.register("loan_to_id")}
                  className="h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  <option value="">Selecione uma congregação…</option>
                  {congregationsData?.data.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              )}
              {loanForm.formState.errors.loan_to_id && (
                <p className="text-xs text-destructive">
                  {loanForm.formState.errors.loan_to_id.message}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="loan-return-date">Previsão de devolução</Label>
              <Input
                id="loan-return-date"
                type="date"
                {...loanForm.register("expected_return_date")}
              />
            </div>

            {loanError && (
              <p className="text-sm text-destructive text-center">{loanError}</p>
            )}

            <DialogFooter>
              <DialogClose asChild>
                <Button type="button" variant="outline">
                  Cancelar
                </Button>
              </DialogClose>
              <Button type="submit" disabled={isCreatingLoan}>
                {isCreatingLoan ? "Registrando…" : "Registrar empréstimo"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      {/* Return modal */}
      <Dialog
        open={showReturnModal}
        onOpenChange={(open) => {
          if (!open) {
            setShowReturnModal(false);
            setActiveLoanId(null);
            setReturnError(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Registrar devolução</DialogTitle>
            <DialogDescription>
              Informe as condições de devolução do item.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={returnForm.handleSubmit(onReturnSubmit)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label>Condição de devolução *</Label>
              <div className="flex flex-col gap-2">
                {(["good", "damaged", "lost"] as const).map((value) => (
                  <label key={value} className="flex items-center gap-2 cursor-pointer min-h-[48px]">
                    <input
                      type="radio"
                      value={value}
                      {...returnForm.register("return_condition")}
                      className="h-4 w-4"
                    />
                    <span className="text-sm">{RETURN_CONDITION_LABELS[value]}</span>
                  </label>
                ))}
              </div>
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
                placeholder="Observações sobre a devolução"
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
                {isReturning ? "Registrando…" : "Confirmar"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm">{value}</span>
    </div>
  );
}

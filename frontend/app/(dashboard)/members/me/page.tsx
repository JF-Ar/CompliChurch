"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Music, X, Pencil } from "lucide-react";
import {
  useMe,
  useUpdateMe,
  useMyInstruments,
  useAddInstrument,
  useRemoveInstrument,
  useInstruments,
} from "@/hooks/useMembers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { RoleBadge } from "@/components/features/members/RoleBadge";
import type { ApiError } from "@/lib/api";

const editSchema = z.object({
  name: z.string().min(2, "Nome deve ter ao menos 2 caracteres"),
  phone: z.string().optional(),
  birth_date: z.string().optional(),
});

type EditValues = z.infer<typeof editSchema>;

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return "—";
  const [year, month, day] = dateStr.split("-");
  return `${day}/${month}/${year}`;
}

export default function MyProfilePage() {
  const [isEditing, setIsEditing] = useState(false);
  const [selectedInstrumentId, setSelectedInstrumentId] = useState("");

  const { data: member, isLoading, isError } = useMe();
  const { data: myInstrumentsData, isLoading: instrLoading } = useMyInstruments();
  const { data: catalogData } = useInstruments();
  const updateMe = useUpdateMe();
  const addInstrument = useAddInstrument();
  const removeInstrument = useRemoveInstrument();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<EditValues>({ resolver: zodResolver(editSchema) });

  function startEdit() {
    if (!member) return;
    reset({
      name: member.name,
      phone: member.phone ?? "",
      birth_date: member.birth_date ?? "",
    });
    setIsEditing(true);
  }

  async function onEdit(values: EditValues) {
    try {
      await updateMe.mutateAsync({
        name: values.name,
        phone: values.phone?.trim() || null,
        birth_date: values.birth_date || null,
      });
      toast.success("Perfil atualizado.");
      setIsEditing(false);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível atualizar. Tente novamente.");
    }
  }

  async function handleAddInstrument() {
    if (!selectedInstrumentId) return;
    try {
      await addInstrument.mutateAsync({ instrument_id: selectedInstrumentId });
      toast.success("Instrumento adicionado.");
      setSelectedInstrumentId("");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível adicionar o instrumento.");
    }
  }

  async function handleRemoveInstrument(instrumentId: string, instrumentName: string) {
    try {
      await removeInstrument.mutateAsync(instrumentId);
      toast.success(`"${instrumentName}" removido.`);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível remover o instrumento.");
    }
  }

  // ── Loading ────────────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="px-4 py-6 max-w-2xl mx-auto space-y-5">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-36 w-full rounded-lg" />
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-24 w-full rounded-lg" />
      </div>
    );
  }

  // ── Error ──────────────────────────────────────────────────────────────────
  if (isError || !member) {
    return (
      <div className="px-4 py-6 max-w-2xl mx-auto">
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar seu perfil. Tente novamente.
        </div>
      </div>
    );
  }

  const myInstruments = myInstrumentsData?.data ?? [];
  const myInstrumentIds = new Set(myInstruments.map((i) => i.instrument_id));
  const availableInstruments = (catalogData?.data ?? []).filter(
    (i) => !myInstrumentIds.has(i.id)
  );

  return (
    <div className="px-4 py-6 max-w-2xl mx-auto space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold truncate">{member.name}</h1>
          <p className="text-sm text-muted-foreground truncate">{member.email}</p>
        </div>
        <Badge variant={member.is_active ? "success" : "muted"} className="shrink-0 mt-0.5">
          {member.is_active ? "Ativo" : "Inativo"}
        </Badge>
      </div>

      {/* Info */}
      <div className="rounded-lg border bg-card p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold">Informações</h2>
          {!isEditing && (
            <Button
              variant="ghost"
              size="sm"
              onClick={startEdit}
              className="h-8 px-2 text-xs"
            >
              <Pencil className="h-3.5 w-3.5 mr-1" />
              Editar
            </Button>
          )}
        </div>

        {isEditing ? (
          <form onSubmit={handleSubmit(onEdit)} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="name">Nome</Label>
              <Input id="name" {...register("name")} />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="phone">Telefone</Label>
              <Input
                id="phone"
                type="tel"
                placeholder="+55 31 99999-0000"
                {...register("phone")}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="birth_date">Data de nascimento</Label>
              <Input id="birth_date" type="date" {...register("birth_date")} />
            </div>
            <div className="flex gap-2">
              <Button type="submit" size="sm" disabled={isSubmitting} className="min-h-[44px]">
                {isSubmitting ? "Salvando…" : "Salvar"}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setIsEditing(false)}
                className="min-h-[44px]"
              >
                Cancelar
              </Button>
            </div>
          </form>
        ) : (
          <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
            <div>
              <dt className="text-xs text-muted-foreground mb-0.5">Telefone</dt>
              <dd>{member.phone ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground mb-0.5">Nascimento</dt>
              <dd>{formatDate(member.birth_date)}</dd>
            </div>
            <div className="col-span-2">
              <dt className="text-xs text-muted-foreground mb-0.5">Membro desde</dt>
              <dd>{new Date(member.created_at).toLocaleDateString("pt-BR")}</dd>
            </div>
          </dl>
        )}
      </div>

      {/* Roles — read-only on own profile */}
      <div className="rounded-lg border bg-card p-4 space-y-3">
        <h2 className="text-base font-semibold">Funções</h2>
        {member.roles.length === 0 ? (
          <p className="text-sm text-muted-foreground">Nenhuma função atribuída.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {member.roles.map((role) => (
              <RoleBadge key={role.id} role={role} />
            ))}
          </div>
        )}
      </div>

      {/* Instruments — full management on own profile */}
      <div className="rounded-lg border bg-card p-4 space-y-3">
        <h2 className="text-base font-semibold">Instrumentos</h2>

        {instrLoading ? (
          <Skeleton className="h-8 w-full" />
        ) : myInstruments.length === 0 ? (
          <p className="text-sm text-muted-foreground">Nenhum instrumento cadastrado.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {myInstruments.map((inst) => (
              <div
                key={inst.id}
                className="flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm"
              >
                <Music className="h-3 w-3 text-muted-foreground" />
                <span>{inst.instrument_name}</span>
                {inst.is_primary && (
                  <span className="text-xs text-muted-foreground">(principal)</span>
                )}
                <button
                  onClick={() =>
                    handleRemoveInstrument(inst.instrument_id, inst.instrument_name)
                  }
                  disabled={removeInstrument.isPending}
                  className="ml-1 rounded-full p-0.5 text-muted-foreground hover:text-destructive disabled:opacity-40 transition-colors"
                  aria-label={`Remover ${inst.instrument_name}`}
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        )}

        {availableInstruments.length > 0 && (
          <div className="flex gap-2 pt-1">
            <select
              value={selectedInstrumentId}
              onChange={(e) => setSelectedInstrumentId(e.target.value)}
              className="flex-1 h-9 rounded-md border border-input bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              aria-label="Selecionar instrumento para adicionar"
            >
              <option value="">Selecionar instrumento…</option>
              {availableInstruments.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
            </select>
            <Button
              size="sm"
              onClick={handleAddInstrument}
              disabled={!selectedInstrumentId || addInstrument.isPending}
              className="min-h-[44px]"
            >
              {addInstrument.isPending ? "Adicionando…" : "Adicionar"}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

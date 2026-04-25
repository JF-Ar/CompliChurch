"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ChevronLeft, Music, X, UserMinus, Pencil } from "lucide-react";
import {
  useMember,
  useUpdateMember,
  useDeactivateMember,
  useAssignRole,
  useRemoveRole,
  useRoles,
} from "@/hooks/useMembers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { RoleBadge } from "@/components/features/members/RoleBadge";
import { DeactivateDialog } from "@/components/features/members/DeactivateDialog";
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

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [showDeactivate, setShowDeactivate] = useState(false);
  const [selectedRoleId, setSelectedRoleId] = useState("");
  const [isEditing, setIsEditing] = useState(false);

  const { data: member, isLoading, isError } = useMember(id);
  const { data: rolesData } = useRoles();
  const updateMember = useUpdateMember(id);
  const deactivate = useDeactivateMember();
  const assignRole = useAssignRole(id);
  const removeRole = useRemoveRole(id);

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
      await updateMember.mutateAsync({
        name: values.name,
        phone: values.phone?.trim() || null,
        birth_date: values.birth_date || null,
      });
      toast.success("Informações atualizadas.");
      setIsEditing(false);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível atualizar. Tente novamente.");
    }
  }

  async function handleDeactivate() {
    try {
      await deactivate.mutateAsync(id);
      toast.success("Membro desativado com sucesso.");
      setShowDeactivate(false);
      router.push("/members");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível desativar. Tente novamente.");
    }
  }

  async function handleAssignRole() {
    if (!selectedRoleId) return;
    try {
      await assignRole.mutateAsync(selectedRoleId);
      toast.success("Função atribuída com sucesso.");
      setSelectedRoleId("");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível atribuir a função.");
    }
  }

  async function handleRemoveRole(roleId: string, roleName: string) {
    try {
      await removeRole.mutateAsync(roleId);
      toast.success(`Função "${roleName}" removida.`);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível remover a função.");
    }
  }

  // ── Loading ────────────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="px-4 py-6 max-w-2xl mx-auto space-y-5">
        <Skeleton className="h-5 w-24" />
        <div className="space-y-1">
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-28 w-full rounded-lg" />
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-20 w-full rounded-lg" />
      </div>
    );
  }

  // ── Error / not found ──────────────────────────────────────────────────────
  if (isError || !member) {
    return (
      <div className="px-4 py-6 max-w-2xl mx-auto">
        <Link
          href="/members"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground mb-4"
        >
          <ChevronLeft className="h-4 w-4" />
          Membros
        </Link>
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Membro não encontrado ou você não tem permissão para visualizar.
        </div>
      </div>
    );
  }

  const assignedRoleIds = new Set(member.roles.map((r) => r.id));
  const availableRoles = (rolesData?.data ?? []).filter((r) => !assignedRoleIds.has(r.id));

  return (
    <>
      <div className="px-4 py-6 max-w-2xl mx-auto space-y-5">
        {/* Back */}
        <Link
          href="/members"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="h-4 w-4" />
          Membros
        </Link>

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
                <Label htmlFor="edit-name">Nome</Label>
                <Input id="edit-name" {...register("name")} />
                {errors.name && (
                  <p className="text-xs text-destructive">{errors.name.message}</p>
                )}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="edit-phone">Telefone</Label>
                <Input
                  id="edit-phone"
                  type="tel"
                  placeholder="+55 31 99999-0000"
                  {...register("phone")}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="edit-birth_date">Data de nascimento</Label>
                <Input id="edit-birth_date" type="date" {...register("birth_date")} />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  size="sm"
                  disabled={isSubmitting}
                  className="min-h-[44px]"
                >
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

        {/* Roles */}
        <div className="rounded-lg border bg-card p-4 space-y-3">
          <h2 className="text-base font-semibold">Funções</h2>

          {member.roles.length === 0 ? (
            <p className="text-sm text-muted-foreground">Nenhuma função atribuída.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {member.roles.map((role) => (
                <div key={role.id} className="flex items-center gap-1">
                  <RoleBadge role={role} />
                  <button
                    onClick={() => handleRemoveRole(role.id, role.name)}
                    disabled={removeRole.isPending}
                    className="rounded-full p-0.5 text-muted-foreground hover:text-destructive disabled:opacity-40 transition-colors"
                    aria-label={`Remover função ${role.name}`}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
            </div>
          )}

          {availableRoles.length > 0 && (
            <div className="flex gap-2 pt-1">
              <select
                value={selectedRoleId}
                onChange={(e) => setSelectedRoleId(e.target.value)}
                className="flex-1 h-9 rounded-md border border-input bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                aria-label="Selecionar função para atribuir"
              >
                <option value="">Selecionar função…</option>
                {availableRoles.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name}
                  </option>
                ))}
              </select>
              <Button
                size="sm"
                onClick={handleAssignRole}
                disabled={!selectedRoleId || assignRole.isPending}
                className="min-h-[44px]"
              >
                {assignRole.isPending ? "Adicionando…" : "Adicionar"}
              </Button>
            </div>
          )}
        </div>

        {/* Instruments — read-only on other member's profile */}
        <div className="rounded-lg border bg-card p-4 space-y-3">
          <h2 className="text-base font-semibold">Instrumentos</h2>
          {member.instruments.length === 0 ? (
            <p className="text-sm text-muted-foreground">Nenhum instrumento cadastrado.</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {member.instruments.map((inst) => (
                <div
                  key={inst.id}
                  className="flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm"
                >
                  <Music className="h-3 w-3 text-muted-foreground" />
                  <span>{inst.instrument_name}</span>
                  {inst.is_primary && (
                    <span className="text-xs text-muted-foreground">(principal)</span>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Danger zone */}
        {member.is_active && (
          <div className="rounded-lg border border-destructive/30 p-4 space-y-2">
            <h2 className="text-base font-semibold text-destructive">Zona de risco</h2>
            <p className="text-sm text-muted-foreground">
              Desativar remove o acesso ao sistema. O cadastro é mantido.
            </p>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setShowDeactivate(true)}
              className="min-h-[48px] w-full sm:w-auto"
            >
              <UserMinus className="h-4 w-4 mr-2" />
              Desativar membro
            </Button>
          </div>
        )}
      </div>

      <DeactivateDialog
        memberName={member.name}
        open={showDeactivate}
        onOpenChange={setShowDeactivate}
        onConfirm={handleDeactivate}
        isLoading={deactivate.isPending}
      />
    </>
  );
}

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ChevronLeft, ChevronDown, ChevronUp, Plus } from "lucide-react";
import { useCreateMember, useRoles } from "@/hooks/useMembers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import type { ApiError } from "@/lib/api";

const PROFILE_VARIANT: Record<string, "default" | "secondary" | "warning" | "muted"> = {
  pastor: "default",
  leadership: "secondary",
  musician: "warning",
  member: "muted",
};

const PROFILE_LABEL: Record<string, string> = {
  pastor: "Pastor",
  leadership: "Liderança",
  musician: "Músico",
  member: "Membro",
};

const schema = z.object({
  name: z.string().min(2, "Nome deve ter ao menos 2 caracteres"),
  email: z.string().email("E-mail inválido"),
  phone: z.string().optional(),
  birth_date: z.string().optional(),
  role_ids: z.array(z.string()).optional(),
});

type FormValues = z.infer<typeof schema>;

export default function NewMemberPage() {
  const router = useRouter();
  const [rolesOpen, setRolesOpen] = useState(false);
  const createMember = useCreateMember();
  const { data: rolesData, isLoading: rolesLoading } = useRoles();

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { role_ids: [] },
  });

  const selectedRoleIds = watch("role_ids") ?? [];

  function toggleRole(id: string) {
    if (selectedRoleIds.includes(id)) {
      setValue("role_ids", selectedRoleIds.filter((r) => r !== id));
    } else {
      setValue("role_ids", [...selectedRoleIds, id]);
    }
  }

  async function onSubmit(values: FormValues) {
    try {
      await createMember.mutateAsync({
        name: values.name,
        email: values.email,
        phone: values.phone?.trim() || null,
        birth_date: values.birth_date || null,
        role_ids: values.role_ids?.length ? values.role_ids : undefined,
      });
      toast.success("Membro cadastrado. Um convite foi enviado por e-mail.");
      router.push("/members");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível cadastrar o membro. Tente novamente.");
    }
  }

  const roles = rolesData?.data ?? [];

  return (
    <div className="px-4 py-6 max-w-lg mx-auto">
      <Link
        href="/members"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-6"
      >
        <ChevronLeft className="h-4 w-4" />
        Membros
      </Link>

      <h1 className="text-xl font-semibold mb-2">Adicionar membro</h1>
      <p className="text-xs text-muted-foreground mb-6">Campos com * são obrigatórios</p>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        <div className="space-y-1.5">
          <Label htmlFor="name">Nome *</Label>
          <Input
            id="name"
            placeholder="João Silva"
            autoComplete="name"
            {...register("name")}
          />
          {errors.name && (
            <p className="text-xs text-destructive">{errors.name.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="email">E-mail *</Label>
          <Input
            id="email"
            type="email"
            placeholder="joao@exemplo.com"
            autoComplete="off"
            {...register("email")}
          />
          {errors.email && (
            <p className="text-xs text-destructive">{errors.email.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="phone">Telefone</Label>
          <Input
            id="phone"
            type="tel"
            placeholder="+55 31 99999-0000"
            autoComplete="tel"
            {...register("phone")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="birth_date">Data de nascimento</Label>
          <Input id="birth_date" type="date" {...register("birth_date")} />
          <p className="text-xs text-muted-foreground">Selecione a data no campo acima</p>
        </div>

        {/* Role selector — collapsed by default */}
        <div className="space-y-2">
          <button
            type="button"
            onClick={() => setRolesOpen((o) => !o)}
            className="inline-flex items-center gap-1.5 text-sm font-medium text-foreground hover:text-foreground/80 transition-colors"
          >
            {rolesOpen ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Adicionar funções
            {selectedRoleIds.length > 0 && (
              <Badge variant="secondary" className="ml-1 text-xs">
                {selectedRoleIds.length} selecionada{selectedRoleIds.length !== 1 ? "s" : ""}
              </Badge>
            )}
          </button>

          {rolesOpen && (
            <div className="rounded-md border bg-card p-3 space-y-1">
              {rolesLoading ? (
                <p className="text-sm text-muted-foreground py-2">Carregando funções…</p>
              ) : roles.length === 0 ? (
                <p className="text-sm text-muted-foreground py-2">Nenhuma função disponível.</p>
              ) : (
                roles.map((role) => {
                  const checked = selectedRoleIds.includes(role.id);
                  return (
                    <label
                      key={role.id}
                      className="flex items-center gap-3 min-h-[48px] px-2 rounded-md cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleRole(role.id)}
                        className="h-4 w-4 shrink-0 rounded border-input accent-primary"
                      />
                      <span className="flex-1 text-sm">{role.name}</span>
                      <Badge
                        variant={PROFILE_VARIANT[role.base_profile] ?? "secondary"}
                        className="text-xs shrink-0"
                      >
                        {PROFILE_LABEL[role.base_profile] ?? role.base_profile}
                      </Badge>
                    </label>
                  );
                })
              )}
            </div>
          )}
        </div>

        <Button type="submit" className="w-full min-h-[48px]" disabled={isSubmitting}>
          {isSubmitting ? "Cadastrando…" : "Cadastrar membro"}
        </Button>
      </form>
    </div>
  );
}

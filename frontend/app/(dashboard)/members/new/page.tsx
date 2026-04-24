"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ChevronLeft } from "lucide-react";
import { useCreateMember } from "@/hooks/useMembers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ApiError } from "@/lib/api";

const schema = z.object({
  name: z.string().min(2, "Nome deve ter ao menos 2 caracteres"),
  email: z.string().email("E-mail inválido"),
  phone: z.string().optional(),
  birth_date: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

export default function NewMemberPage() {
  const router = useRouter();
  const createMember = useCreateMember();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  async function onSubmit(values: FormValues) {
    try {
      await createMember.mutateAsync({
        name: values.name,
        email: values.email,
        phone: values.phone?.trim() || null,
        birth_date: values.birth_date || null,
      });
      toast.success("Membro cadastrado. Um convite foi enviado por e-mail.");
      router.push("/members");
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível cadastrar o membro. Tente novamente.");
    }
  }

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

        <Button type="submit" className="w-full min-h-[48px]" disabled={isSubmitting}>
          {isSubmitting ? "Cadastrando…" : "Cadastrar membro"}
        </Button>
      </form>
    </div>
  );
}

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { register as registerUser, setSession, type ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const schema = z
  .object({
    church_name: z.string().min(1, "Nome da igreja obrigatório"),
    pastor_name: z.string().min(1, "Seu nome é obrigatório"),
    email: z.string().email("E-mail inválido"),
    password: z.string().min(8, "Senha deve ter no mínimo 8 caracteres"),
    confirm_password: z.string().min(1, "Confirmação de senha obrigatória"),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "As senhas não coincidem",
    path: ["confirm_password"],
  });

type FormValues = z.infer<typeof schema>;

export default function RegisterPage() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
  });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      const data = await registerUser({
        church_name: values.church_name,
        pastor_name: values.pastor_name,
        email: values.email,
        password: values.password,
      });
      setSession(data.access_token, data.member, data.church);
      router.push("/dashboard");
    } catch (err) {
      const apiErr = err as ApiError;
      const code = apiErr?.error?.code;
      const field = apiErr?.error?.field;

      if (code === "CONFLICT" || (typeof code === "string" && code.includes("409"))) {
        setError("email", { message: "E-mail já cadastrado" });
        return;
      }

      if (field) {
        const fieldMap: Record<string, keyof FormValues> = {
          church_name: "church_name",
          pastor_name: "pastor_name",
          email: "email",
          password: "password",
        };
        const formField = fieldMap[field];
        if (formField) {
          setError(formField, { message: apiErr.error.message });
          return;
        }
      }

      setServerError(apiErr?.error?.message ?? "Erro ao criar conta. Tente novamente.");
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 px-4 py-8">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-bold tracking-tight">Igreja Organizada</h1>
          <p className="text-sm text-muted-foreground">Cadastre sua igreja</p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1">
            <Label htmlFor="church_name">Nome da igreja</Label>
            <Input
              id="church_name"
              type="text"
              autoComplete="organization"
              placeholder="Igreja Batista Central"
              {...register("church_name")}
            />
            {errors.church_name && (
              <p className="text-xs text-destructive">{errors.church_name.message}</p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="pastor_name">Seu nome</Label>
            <Input
              id="pastor_name"
              type="text"
              autoComplete="name"
              placeholder="João da Silva"
              {...register("pastor_name")}
            />
            {errors.pastor_name && (
              <p className="text-xs text-destructive">{errors.pastor_name.message}</p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="email">E-mail</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="voce@exemplo.com"
              {...register("email")}
            />
            {errors.email && (
              <p className="text-xs text-destructive">{errors.email.message}</p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="password">Senha</Label>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              placeholder="Mínimo 8 caracteres"
              {...register("password")}
            />
            {errors.password && (
              <p className="text-xs text-destructive">{errors.password.message}</p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="confirm_password">Confirmar senha</Label>
            <Input
              id="confirm_password"
              type="password"
              autoComplete="new-password"
              placeholder="••••••••"
              {...register("confirm_password")}
            />
            {errors.confirm_password && (
              <p className="text-xs text-destructive">{errors.confirm_password.message}</p>
            )}
          </div>

          {serverError && (
            <p className="text-sm text-destructive text-center">{serverError}</p>
          )}

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? "Criando conta…" : "Criar conta"}
          </Button>
        </form>

        <p className="text-sm text-center text-muted-foreground">
          Já tem uma conta?{" "}
          <Link href="/login" className="font-medium text-foreground underline-offset-4 hover:underline">
            Entrar
          </Link>
        </p>
      </div>
    </div>
  );
}

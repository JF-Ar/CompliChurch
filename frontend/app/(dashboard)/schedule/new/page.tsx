"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import Link from "next/link";
import type { Resolver } from "react-hook-form";
import { useCreateSchedule } from "@/hooks/useSchedules";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ApiError } from "@/lib/api/client";

const schema = z.object({
  sunday_date: z
    .string()
    .min(1, "Data obrigatória")
    .refine((v) => new Date(v + "T12:00:00").getDay() === 0, {
      message: "Deve ser um domingo",
    }),
  notes: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

export default function NewSchedulePage() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const { mutateAsync: doCreate } = useCreateSchedule();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema) as Resolver<FormValues>,
  });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      const schedule = await doCreate({
        sunday_date: values.sunday_date,
        notes: values.notes || null,
      });
      router.push(`/schedule/${schedule.id}`);
    } catch (err) {
      const e = err as ApiError;
      if (e?.error?.code === "SCHEDULE_ALREADY_EXISTS" || e?.error?.code === "CONFLICT") {
        setServerError("Já existe uma escala para este domingo.");
      } else {
        setServerError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
      }
    }
  }

  return (
    <div className="flex flex-col gap-6 p-4 pb-24 md:pb-6 max-w-lg mx-auto">
      <div className="flex items-center gap-2">
        <Button asChild variant="ghost" size="sm">
          <Link href="/schedule">← Escalas</Link>
        </Button>
        <h1 className="text-xl font-semibold">Nova escala</h1>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <p className="text-xs text-muted-foreground">Campos com * são obrigatórios</p>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sunday_date">Data do culto *</Label>
          <Input id="sunday_date" type="date" {...register("sunday_date")} />
          {errors.sunday_date && (
            <p className="text-xs text-destructive">{errors.sunday_date.message}</p>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="notes">Observações</Label>
          <textarea
            id="notes"
            rows={3}
            className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 resize-none"
            placeholder="Tema do culto, informações especiais…"
            {...register("notes")}
          />
        </div>

        {serverError && (
          <p className="text-sm text-destructive text-center">{serverError}</p>
        )}

        <Button type="submit" disabled={isSubmitting} className="w-full mt-2">
          {isSubmitting ? "Criando…" : "Criar escala"}
        </Button>
      </form>
    </div>
  );
}

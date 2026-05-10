"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useMyExceptions, useAddException, useRemoveException } from "@/hooks/useSchedules";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { ApiError } from "@/lib/api/client";

function getSundaysOfMonth(year: number, month: number): Date[] {
  const sundays: Date[] = [];
  const d = new Date(year, month, 1);
  while (d.getDay() !== 0) d.setDate(d.getDate() + 1);
  while (d.getMonth() === month) {
    sundays.push(new Date(d));
    d.setDate(d.getDate() + 7);
  }
  return sundays;
}

function toDateStr(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function formatDate(d: string) {
  const [, m, day] = d.split("-");
  return `${day}/${m}`;
}

export default function AvailabilityPage() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth());

  const monthStr = `${year}-${String(month + 1).padStart(2, "0")}`;

  const { data: exceptionsData, isLoading, error } = useMyExceptions(monthStr);
  const { mutateAsync: doAdd, isPending: isAdding } = useAddException();
  const { mutateAsync: doRemove, isPending: isRemoving } = useRemoveException();

  const exceptions = exceptionsData?.data ?? [];
  const sundays = getSundaysOfMonth(year, month);

  const monthLabel = new Date(year, month, 1).toLocaleDateString("pt-BR", {
    month: "long",
    year: "numeric",
  });

  function prevMonth() {
    if (month === 0) {
      setYear((y) => y - 1);
      setMonth(11);
    } else {
      setMonth((m) => m - 1);
    }
  }

  function nextMonth() {
    if (month === 11) {
      setYear((y) => y + 1);
      setMonth(0);
    } else {
      setMonth((m) => m + 1);
    }
  }

  async function handleAdd(date: string) {
    try {
      await doAdd({ unavailable_date: date });
      toast.success("Domingo marcado como indisponível.");
    } catch (err) {
      const e = err as ApiError;
      if (e?.error?.code === "CONFLICT" || e?.error?.code === "AVAILABILITY_EXCEPTION_EXISTS") {
        toast.error("Você já marcou este domingo como indisponível.");
      } else {
        toast.error(e?.error?.message ?? "Erro inesperado. Tente novamente.");
      }
    }
  }

  async function handleRemove(id: string) {
    try {
      await doRemove(id);
      toast.success("Disponibilidade restaurada.");
    } catch {
      toast.error("Não foi possível remover. Tente novamente.");
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-lg mx-auto">
      <h1 className="text-xl font-semibold">Minha disponibilidade</h1>

      {/* Month navigation */}
      <div className="flex items-center justify-between gap-2">
        <Button variant="outline" size="sm" onClick={prevMonth}>
          ‹ Anterior
        </Button>
        <span className="text-sm font-medium capitalize">{monthLabel}</span>
        <Button variant="outline" size="sm" onClick={nextMonth}>
          Próximo ›
        </Button>
      </div>

      {/* Error */}
      {error && !isLoading && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar. Tente novamente.
        </div>
      )}

      {/* Loading */}
      {isLoading && (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Sunday list */}
      {!isLoading && sundays.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-8">
          Nenhum domingo encontrado.
        </p>
      )}

      {!isLoading && sundays.length > 0 && (
        <div className="flex flex-col gap-3">
          {sundays.map((sunday) => {
            const dateStr = toDateStr(sunday);
            const exception = exceptions.find((e) => e.unavailable_date === dateStr);
            return (
              <div
                key={dateStr}
                className="flex items-center justify-between gap-3 rounded-lg border bg-card p-4 shadow-sm"
              >
                <div className="flex flex-col gap-0.5">
                  <span className="text-sm font-medium">
                    {formatDate(dateStr)} · Domingo
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {sunday.toLocaleDateString("pt-BR", { month: "long" })}
                  </span>
                </div>
                {exception ? (
                  <div className="flex items-center gap-2">
                    <Badge variant="warning">Indisponível</Badge>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={isRemoving}
                      onClick={() => handleRemove(exception.id)}
                    >
                      Remover
                    </Button>
                  </div>
                ) : (
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={isAdding}
                    onClick={() => handleAdd(dateStr)}
                  >
                    Indisponível
                  </Button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

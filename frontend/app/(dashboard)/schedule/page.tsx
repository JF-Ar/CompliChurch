"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useMe } from "@/hooks/useMembers";
import { useSchedules } from "@/hooks/useSchedules";
import { Button } from "@/components/ui/button";
import { ScheduleCard, ScheduleCardSkeleton } from "@/components/features/schedule/ScheduleCard";
import { cn } from "@/lib/utils";

type StatusFilter = "" | "draft" | "published" | "cancelled";

const FILTER_TABS: { value: StatusFilter; label: string }[] = [
  { value: "", label: "Todas" },
  { value: "draft", label: "Rascunho" },
  { value: "published", label: "Publicada" },
  { value: "cancelled", label: "Cancelada" },
];

export default function ScheduleListPage() {
  const router = useRouter();
  const [status, setStatus] = useState<StatusFilter>("");
  const [page, setPage] = useState(1);

  const { data: meData } = useMe();
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

  const { data, isLoading, error } = useSchedules({
    status: status || undefined,
    page,
    per_page: 12,
  });

  const schedules = data?.data ?? [];
  const meta = data?.meta;
  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 1;

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <h1 className="text-xl font-semibold">Escalas</h1>
        <div className="flex flex-col gap-2 md:flex-row md:items-center">
          <Button variant="outline" asChild className="w-full md:w-auto">
            <Link href="/availability">Minha disponibilidade</Link>
          </Button>
          {isLeadership && (
            <Button size="sm" className="w-full md:w-auto" onClick={() => router.push("/schedule/new")}>
              + Nova escala
            </Button>
          )}
        </div>
      </div>

      {/* Status filter tabs */}
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {FILTER_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => {
              setStatus(tab.value);
              setPage(1);
            }}
            className={cn(
              "shrink-0 rounded-full px-4 py-1.5 text-sm transition-colors",
              status === tab.value
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <ScheduleCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {error && !isLoading && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar as escalas. Tente novamente.
        </div>
      )}

      {/* Empty */}
      {!isLoading && !error && schedules.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <span className="text-5xl" aria-hidden="true">🎵</span>
          <p className="text-sm text-muted-foreground">Nenhuma escala encontrada.</p>
          {isLeadership && !status && (
            <Button size="sm" onClick={() => router.push("/schedule/new")}>
              Criar primeira escala
            </Button>
          )}
        </div>
      )}

      {/* List */}
      {!isLoading && !error && schedules.length > 0 && (
        <div className="flex flex-col gap-3">
          {schedules.map((s) => (
            <ScheduleCard key={s.id} schedule={s} />
          ))}
        </div>
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
    </div>
  );
}

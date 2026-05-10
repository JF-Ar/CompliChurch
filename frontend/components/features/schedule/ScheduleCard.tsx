import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import type { ScheduleSummary } from "@/lib/api";

function formatDate(d: string) {
  const [y, m, day] = d.split("-");
  return `${day}/${m}/${y}`;
}

const STATUS_LABELS: Record<string, string> = {
  draft: "Rascunho",
  published: "Publicada",
  cancelled: "Cancelada",
};

const STATUS_VARIANTS: Record<string, "secondary" | "success" | "destructive"> = {
  draft: "secondary",
  published: "success",
  cancelled: "destructive",
};

export function ScheduleCard({ schedule }: { schedule: ScheduleSummary }) {
  return (
    <Link
      href={`/schedule/${schedule.id}`}
      className="flex flex-col gap-2 rounded-lg border bg-card p-4 shadow-sm hover:bg-accent/50 transition-colors active:scale-[0.99] min-h-20"
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-sm font-medium">
          Domingo, {formatDate(schedule.sunday_date)}
        </span>
        <Badge variant={STATUS_VARIANTS[schedule.status] ?? "secondary"}>
          {STATUS_LABELS[schedule.status] ?? schedule.status}
        </Badge>
      </div>
      <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
        <span>
          {schedule.slot_count} músico{schedule.slot_count !== 1 ? "s" : ""} escalado
          {schedule.slot_count !== 1 ? "s" : ""}
        </span>
        {schedule.published_at && (
          <span>· Publicada em {formatDate(schedule.published_at.split("T")[0])}</span>
        )}
      </div>
    </Link>
  );
}

export function ScheduleCardSkeleton() {
  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-card p-4 shadow-sm min-h-20 animate-pulse">
      <div className="flex items-start justify-between gap-2">
        <div className="h-4 w-44 bg-muted rounded" />
        <div className="h-5 w-20 bg-muted rounded-full" />
      </div>
      <div className="h-3 w-28 bg-muted rounded" />
    </div>
  );
}

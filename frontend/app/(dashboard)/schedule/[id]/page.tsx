"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useMe, useMembers, useInstruments } from "@/hooks/useMembers";
import {
  useSchedule,
  usePublishSchedule,
  useAddSlot,
  useRemoveSlot,
  useConfirmSlot,
  useScheduleSuggestion,
} from "@/hooks/useSchedules";
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
import type { ApiError } from "@/lib/api/client";

function formatDate(d: string | null | undefined) {
  if (!d) return "—";
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

export default function ScheduleDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [showPublishDialog, setShowPublishDialog] = useState(false);
  const [fetchSuggestion, setFetchSuggestion] = useState(false);
  const [memberSearch, setMemberSearch] = useState("");
  const [addMemberId, setAddMemberId] = useState("");
  const [addInstrumentId, setAddInstrumentId] = useState("");
  const [addFunction, setAddFunction] = useState("");
  const [addError, setAddError] = useState<string | null>(null);

  const { data: meData } = useMe();
  const currentMemberId = meData?.id;
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

  const { data: schedule, isLoading, error } = useSchedule(id);
  const { data: membersData } = useMembers({ per_page: 200 });
  const { data: instrumentsData } = useInstruments();
  const { data: suggestion, isLoading: isSuggesting } = useScheduleSuggestion(
    fetchSuggestion ? (schedule?.sunday_date ?? "") : ""
  );

  const { mutateAsync: doPublish, isPending: isPublishing } = usePublishSchedule(id);
  const { mutateAsync: doAddSlot, isPending: isAddingSlot } = useAddSlot(id);
  const { mutateAsync: doRemoveSlot } = useRemoveSlot(id);
  const { mutateAsync: doConfirmSlot, isPending: isConfirming } = useConfirmSlot(id);

  const members = membersData?.data ?? [];
  const instruments = instrumentsData?.data ?? [];

  const filteredMembers = memberSearch
    ? members.filter((m) => m.name.toLowerCase().includes(memberSearch.toLowerCase()))
    : members;

  async function handlePublish() {
    try {
      await doPublish();
      toast.success("Escala publicada com sucesso.");
      setShowPublishDialog(false);
    } catch (err) {
      const e = err as ApiError;
      toast.error(e?.error?.message ?? "Não foi possível publicar. Tente novamente.");
      setShowPublishDialog(false);
    }
  }

  async function handleAddSlot() {
    if (!addMemberId || !addFunction.trim()) return;
    setAddError(null);
    try {
      await doAddSlot({
        member_id: addMemberId,
        instrument_id: addInstrumentId || null,
        function_in_scale: addFunction.trim(),
      });
      setAddMemberId("");
      setAddInstrumentId("");
      setAddFunction("");
      setMemberSearch("");
      toast.success("Músico adicionado à escala.");
    } catch (err) {
      const e = err as ApiError;
      if (e?.error?.code === "CONFLICT" || e?.error?.code === "SCHEDULE_SLOT_EXISTS") {
        setAddError("Este membro já está nesta escala.");
      } else {
        setAddError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
      }
    }
  }

  async function handleRemoveSlot(slotId: string) {
    try {
      await doRemoveSlot(slotId);
      toast.success("Músico removido da escala.");
    } catch {
      toast.error("Não foi possível remover. Tente novamente.");
    }
  }

  async function handleConfirmSlot(slotId: string) {
    try {
      await doConfirmSlot(slotId);
      toast.success("Presença confirmada.");
    } catch {
      toast.error("Não foi possível confirmar. Tente novamente.");
    }
  }

  async function handleAddFromSuggestion(suggested: {
    member_id: string;
    instrument_id: string;
    instrument_name: string;
  }) {
    try {
      await doAddSlot({
        member_id: suggested.member_id,
        instrument_id: suggested.instrument_id || null,
        function_in_scale: suggested.instrument_name,
      });
      toast.success("Músico adicionado à escala.");
    } catch (err) {
      const e = err as ApiError;
      if (e?.error?.code === "CONFLICT" || e?.error?.code === "SCHEDULE_SLOT_EXISTS") {
        toast.error("Este membro já está nesta escala.");
      } else {
        toast.error(e?.error?.message ?? "Erro inesperado.");
      }
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-6 w-24 rounded-full" />
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  if (error || !schedule) {
    return (
      <div className="flex flex-col gap-4 p-4 max-w-2xl mx-auto">
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar a escala. Tente novamente.
        </div>
        <Button asChild variant="outline" size="sm" className="w-fit">
          <Link href="/schedule">← Escalas</Link>
        </Button>
      </div>
    );
  }

  const isDraft = schedule.status === "draft";

  return (
    <div className="flex flex-col gap-6 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
      <Button asChild variant="ghost" size="sm" className="w-fit">
        <Link href="/schedule">← Escalas</Link>
      </Button>

      {/* A) Header */}
      <div className="flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2 flex-wrap">
          <h1 className="text-xl font-semibold">
            Domingo, {formatDate(schedule.sunday_date)}
          </h1>
          <Badge variant={STATUS_VARIANTS[schedule.status] ?? "secondary"}>
            {STATUS_LABELS[schedule.status] ?? schedule.status}
          </Badge>
        </div>
        {schedule.notes && (
          <p className="text-sm text-muted-foreground">{schedule.notes}</p>
        )}
        {isLeadership && isDraft && (
          <Button
            size="sm"
            className="w-fit mt-1"
            onClick={() => setShowPublishDialog(true)}
          >
            Publicar escala
          </Button>
        )}
      </div>

      {/* B) Slots section */}
      <section className="flex flex-col gap-3">
        <h2 className="text-base font-semibold">Músicos escalados</h2>
        {schedule.slots.length === 0 ? (
          <p className="text-sm text-muted-foreground">Nenhum músico adicionado ainda.</p>
        ) : (
          <div className="flex flex-col gap-2">
            {schedule.slots.map((slot) => (
              <div
                key={slot.id}
                className="flex items-center gap-3 rounded-lg border bg-card p-4 shadow-sm flex-wrap"
              >
                <div className="flex-1 flex flex-col gap-0.5 min-w-0">
                  <span className="text-sm font-medium">{slot.member.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {slot.function_in_scale}
                    {slot.instrument ? ` · ${slot.instrument.name}` : ""}
                  </span>
                </div>
                <div className="flex items-center gap-2 flex-wrap">
                  <Badge variant={slot.confirmed ? "success" : "warning"}>
                    {slot.confirmed ? "Confirmado" : "Aguardando"}
                  </Badge>
                  {slot.member.id === currentMemberId && !slot.confirmed && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={isConfirming}
                      onClick={() => handleConfirmSlot(slot.id)}
                    >
                      Confirmar presença
                    </Button>
                  )}
                  {isLeadership && isDraft && (
                    <button
                      aria-label="Remover músico da escala"
                      className="flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors text-lg"
                      onClick={() => handleRemoveSlot(slot.id)}
                    >
                      ×
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* C) Add slot form */}
      {isLeadership && isDraft && (
        <section className="flex flex-col gap-3 rounded-lg border bg-card p-4 shadow-sm">
          <h2 className="text-base font-semibold">Adicionar músico</h2>

          <div className="flex flex-col gap-1.5">
            <Label>Filtrar membro</Label>
            <Input
              placeholder="Buscar por nome…"
              value={memberSearch}
              onChange={(e) => setMemberSearch(e.target.value)}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Membro *</Label>
            <select
              aria-label="Selecionar membro"
              value={addMemberId}
              onChange={(e) => setAddMemberId(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">Selecione um membro</option>
              {filteredMembers
                .filter((m) => m.is_active)
                .map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Instrumento</Label>
            <select
              aria-label="Selecionar instrumento"
              value={addInstrumentId}
              onChange={(e) => setAddInstrumentId(e.target.value)}
              className="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">Sem instrumento</option>
              {instruments.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Função na escala *</Label>
            <Input
              placeholder="Ex: Violão, Vocal principal…"
              value={addFunction}
              onChange={(e) => setAddFunction(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleAddSlot();
              }}
            />
          </div>

          {addError && <p className="text-sm text-destructive">{addError}</p>}

          <Button
            disabled={isAddingSlot || !addMemberId || !addFunction.trim()}
            onClick={handleAddSlot}
          >
            {isAddingSlot ? "Adicionando…" : "Adicionar"}
          </Button>
        </section>
      )}

      {/* D) Suggestion panel */}
      {isLeadership && isDraft && (
        <section className="flex flex-col gap-3 rounded-lg border bg-card p-4 shadow-sm">
          <h2 className="text-base font-semibold">Sugestão automática</h2>
          {!fetchSuggestion && (
            <Button variant="outline" onClick={() => setFetchSuggestion(true)}>
              Sugerir escala automaticamente
            </Button>
          )}
          {fetchSuggestion && isSuggesting && (
            <Button variant="outline" disabled>
              Carregando sugestão…
            </Button>
          )}
          {fetchSuggestion && !isSuggesting && suggestion && (
            <div className="flex flex-col gap-4">
              {suggestion.suggested_slots.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Nenhuma sugestão disponível para este domingo.
                </p>
              ) : (
                <div className="flex flex-col gap-2">
                  <p className="text-xs text-muted-foreground">
                    Músicos sugeridos para este domingo:
                  </p>
                  {suggestion.suggested_slots.map((s) => (
                    <div
                      key={s.member_id}
                      className="flex items-center gap-3 rounded-md border p-3 flex-wrap"
                    >
                      <div className="flex-1 flex flex-col gap-0.5 min-w-0">
                        <span className="text-sm font-medium">{s.member_name}</span>
                        <span className="text-xs text-muted-foreground">
                          {s.instrument_name}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 flex-wrap">
                        {s.warning === "consecutive_sunday" && (
                          <Badge variant="warning">Domingo consecutivo</Badge>
                        )}
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() =>
                            handleAddFromSuggestion({
                              member_id: s.member_id,
                              instrument_id: s.instrument_id,
                              instrument_name: s.instrument_name,
                            })
                          }
                        >
                          + Adicionar
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {suggestion.unavailable_members.length > 0 && (
                <div className="flex flex-col gap-1.5">
                  <p className="text-xs font-medium text-muted-foreground">
                    Indisponíveis neste domingo:
                  </p>
                  {suggestion.unavailable_members.map((u) => (
                    <div key={u.member.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{u.member.name}</span>
                      {u.reason && <span>· {u.reason}</span>}
                    </div>
                  ))}
                </div>
              )}
              <Button
                variant="ghost"
                size="sm"
                className="w-fit"
                onClick={() => setFetchSuggestion(false)}
              >
                Ocultar sugestão
              </Button>
            </div>
          )}
        </section>
      )}

      {/* Publish confirmation dialog */}
      <Dialog open={showPublishDialog} onOpenChange={setShowPublishDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Publicar escala?</DialogTitle>
            <DialogDescription>
              Publicar enviará um e-mail para todos os músicos escalados. Esta
              ação não pode ser desfeita.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button disabled={isPublishing} onClick={handlePublish}>
              {isPublishing ? "Publicando…" : "Publicar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

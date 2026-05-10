"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import * as XLSX from "xlsx";
import { toast } from "sonner";
import { useMe } from "@/hooks/useMembers";
import { useItems, useCategories, useImportItems } from "@/hooks/useInventory";
import { useDebounce } from "@/hooks/useDebounce";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

const STATUS_LABELS: Record<string, string> = {
  available: "Disponível",
  on_loan: "Emprestado",
  damaged: "Com dano",
  maintenance: "Em manutenção",
};

const STATUS_VARIANTS: Record<string, "success" | "warning" | "orange" | "destructive"> = {
  available: "success",
  on_loan: "warning",
  damaged: "orange",
  maintenance: "destructive",
};

function getItemBadge(item: { status: string; deletion_reason?: "donated" | "discarded" | null }) {
  if (item.deletion_reason === "donated") return { label: "Doado", variant: "muted" as const };
  if (item.deletion_reason === "discarded") return { label: "Descartado", variant: "destructive" as const };
  return { label: STATUS_LABELS[item.status] ?? item.status, variant: STATUS_VARIANTS[item.status] ?? "muted" as const };
}

export default function InventoryPage() {
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState("");
  const [itemType, setItemType] = useState("");
  const [includeDeleted, setIncludeDeleted] = useState(false);
  const [page, setPage] = useState(1);
  const [importErrors, setImportErrors] = useState<Array<{ row: number; reason: string }> | null>(null);
  const [categoryWarnings, setCategoryWarnings] = useState<Array<{ row: number; informed_name: string; matched_name: string }>>([]);
  const [showCategoryWarnings, setShowCategoryWarnings] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const debouncedSearch = useDebounce(search, 400);

  const { data: meData } = useMe();
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

  const { mutateAsync: doImport, isPending: isImporting } = useImportItems();

  const { data: categoriesData } = useCategories();

  const { data, isLoading, error } = useItems({
    search: debouncedSearch || undefined,
    category_id: categoryId || undefined,
    status: status || undefined,
    item_type: (itemType || undefined) as "asset" | "consumable" | undefined,
    include_deleted: includeDeleted || undefined,
    page,
    per_page: 20,
  });

  const items = data?.data ?? [];
  const meta = data?.meta;
  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 1;

  const hasFilters = !!(debouncedSearch || categoryId || status || itemType);

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = "";
    try {
      const result = await doImport(file);
      if (result.errors.length === 0) {
        toast.success(`${result.created} itens importados com sucesso`);
      } else {
        toast.success(`${result.created} importados · ${result.errors.length} com erro`);
        setImportErrors(result.errors);
      }
      if (result.category_warnings.length > 0) {
        setCategoryWarnings(result.category_warnings);
        setShowCategoryWarnings(true);
      }
    } catch {
      // network error already handled by hook's onError
    }
  }

  function downloadTemplate() {
    const headers = [
      "name", "item_type", "location", "category", "description",
      "asset_number", "quantity", "qty_min_alert", "serial_number", "notes",
    ];
    const example = [
      "Violão Yamaha", "asset", "Sala Principal", "Instrumentos",
      "Violão acústico", "", "1", "", "", "",
    ];
    const ws = XLSX.utils.aoa_to_sheet([headers, example]);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, "Itens");
    XLSX.writeFile(wb, "modelo_patrimonio.xlsx");
  }

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-xl font-semibold">Patrimônio</h1>
        <div className="flex flex-wrap gap-2 justify-end">
          {isLeadership && (
            <Button asChild variant="outline" size="sm">
              <Link href="/inventory/loans">Empréstimos</Link>
            </Button>
          )}
          {isLeadership && (
            <Button variant="outline" size="sm" onClick={downloadTemplate}>
              Baixar modelo
            </Button>
          )}
          {isLeadership && (
            <Button
              variant="outline"
              size="sm"
              disabled={isImporting}
              onClick={() => fileInputRef.current?.click()}
            >
              {isImporting ? "Importando…" : "Importar planilha"}
            </Button>
          )}
          {isLeadership && (
            <Button asChild size="sm">
              <Link href="/inventory/new">+ Novo item</Link>
            </Button>
          )}
        </div>
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept=".xlsx"
        className="hidden"
        onChange={handleFileSelected}
      />

      {/* Filters */}
      <div className="flex flex-col gap-2">
        <Input
          placeholder="Buscar por nome…"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
        />
        <div className="flex flex-wrap gap-2">
          <select
            aria-label="Filtrar por categoria"
            className="h-10 flex-1 min-w-[140px] rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={categoryId}
            onChange={(e) => {
              setCategoryId(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Todas as categorias</option>
            {categoriesData?.data.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <select
            aria-label="Filtrar por status"
            className="h-10 flex-1 min-w-[140px] rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Todos os status</option>
            <option value="available">Disponível</option>
            <option value="on_loan">Emprestado</option>
            <option value="damaged">Com dano</option>
            <option value="maintenance">Em manutenção</option>
          </select>
          <select
            aria-label="Filtrar por tipo"
            className="h-10 flex-1 min-w-[140px] rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={itemType}
            onChange={(e) => {
              setItemType(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Todos os tipos</option>
            <option value="asset">Bem permanente</option>
            <option value="consumable">Consumível</option>
          </select>
        </div>
        {isLeadership && (
          <label className="flex items-center gap-2 text-sm cursor-pointer w-fit">
            <input
              type="checkbox"
              checked={includeDeleted}
              onChange={(e) => {
                setIncludeDeleted(e.target.checked);
                setPage(1);
              }}
              className="h-4 w-4 rounded border-input"
            />
            Mostrar descartados e doados
          </label>
        )}
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Error */}
      {error && !isLoading && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar o patrimônio. Tente novamente.
        </div>
      )}

      {/* Empty */}
      {!isLoading && !error && items.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <span className="text-5xl">📦</span>
          <p className="text-sm text-muted-foreground">
            {hasFilters
              ? "Nenhum item encontrado para os filtros selecionados."
              : "Nenhum item cadastrado ainda."}
          </p>
          {isLeadership && !hasFilters && (
            <Button asChild size="sm">
              <Link href="/inventory/new">Adicionar primeiro item</Link>
            </Button>
          )}
        </div>
      )}

      {/* Item cards */}
      {!isLoading && !error && items.length > 0 && (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <Link
              key={item.id}
              href={`/inventory/${item.id}`}
              className="flex gap-3 rounded-lg border bg-card p-4 shadow-sm hover:bg-accent/50 transition-colors active:scale-[0.99]"
            >
              <div className="h-14 w-14 shrink-0 rounded-md bg-muted flex items-center justify-center overflow-hidden">
                {item.photo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={item.photo_url}
                    alt={item.name}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <span className="text-2xl" aria-hidden="true">
                    📦
                  </span>
                )}
              </div>
              <div className="flex flex-col gap-1 min-w-0 flex-1">
                <div className="flex items-baseline gap-2 flex-wrap">
                  <span className="text-sm font-medium">{item.name}</span>
                  {item.asset_number && (
                    <span className="text-xs text-muted-foreground">{item.asset_number}</span>
                  )}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  <Badge variant={getItemBadge(item).variant}>
                    {getItemBadge(item).label}
                  </Badge>
                  {item.category && (
                    <Badge variant="outline">{item.category.name}</Badge>
                  )}
                </div>
                <span className="text-xs text-muted-foreground truncate">{item.location}</span>
              </div>
            </Link>
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

      {/* Import errors dialog */}
      <Dialog open={!!importErrors} onOpenChange={(open) => { if (!open) setImportErrors(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Erros na importação</DialogTitle>
            <DialogDescription>
              Os itens abaixo não foram importados. Corrija a planilha e importe novamente.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5 max-h-64 overflow-y-auto">
            {importErrors?.map((e) => (
              <p key={e.row} className="text-sm">
                <span className="font-medium">Linha {e.row}:</span> {e.reason}
              </p>
            ))}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Fechar</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Category warnings dialog */}
      <Dialog open={showCategoryWarnings} onOpenChange={(open) => { if (!open) setShowCategoryWarnings(false); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Categorias ajustadas automaticamente</DialogTitle>
            <DialogDescription>
              As categorias abaixo foram aproximadas automaticamente. Verifique se o agrupamento está correto.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5 max-h-64 overflow-y-auto">
            {categoryWarnings.map((w) => (
              <div key={w.row} className="flex items-center flex-wrap gap-1 text-sm">
                <span className="font-medium">Linha {w.row}:</span>
                <span>&ldquo;{w.informed_name}&rdquo;</span>
                <Badge variant="warning">→</Badge>
                <span>&ldquo;{w.matched_name}&rdquo;</span>
              </div>
            ))}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Entendi</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

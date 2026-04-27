"use client";

import { useState } from "react";
import Link from "next/link";
import { useMe } from "@/hooks/useMembers";
import { useItems, useCategories } from "@/hooks/useInventory";
import { useDebounce } from "@/hooks/useDebounce";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

const STATUS_LABELS: Record<string, string> = {
  available: "Disponível",
  on_loan: "Emprestado",
  maintenance: "Manutenção",
};

const STATUS_VARIANTS: Record<string, "success" | "warning" | "destructive"> = {
  available: "success",
  on_loan: "warning",
  maintenance: "destructive",
};

export default function InventoryPage() {
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState("");
  const [itemType, setItemType] = useState("");
  const [includeDeleted, setIncludeDeleted] = useState(false);
  const [page, setPage] = useState(1);

  const debouncedSearch = useDebounce(search, 400);

  const { data: meData } = useMe();
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

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

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Patrimônio</h1>
        <div className="flex gap-2">
          {isLeadership && (
            <Button asChild variant="outline" size="sm">
              <Link href="/inventory/loans">Empréstimos</Link>
            </Button>
          )}
          {isLeadership && (
            <Button asChild size="sm">
              <Link href="/inventory/new">+ Novo item</Link>
            </Button>
          )}
        </div>
      </div>

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
            <option value="maintenance">Manutenção</option>
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
                  <Badge variant={STATUS_VARIANTS[item.status]}>
                    {STATUS_LABELS[item.status]}
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
    </div>
  );
}

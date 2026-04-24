"use client";

import { useState } from "react";
import Link from "next/link";
import { UserPlus, Search, Users } from "lucide-react";
import { useMembers } from "@/hooks/useMembers";
import { useDebounce } from "@/hooks/useDebounce";
import { MemberCard } from "@/components/features/members/MemberCard";
import { MemberCardSkeleton } from "@/components/features/members/MemberCardSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const ROLE_OPTIONS = [
  { value: "", label: "Todos os perfis" },
  { value: "pastor", label: "Pastor" },
  { value: "leadership", label: "Liderança" },
  { value: "musician", label: "Músico" },
  { value: "member", label: "Membro" },
];

export default function MembersPage() {
  const [search, setSearch] = useState("");
  const [role, setRole] = useState("");
  const [page, setPage] = useState(1);

  const debouncedSearch = useDebounce(search, 350);

  const { data, isLoading, isError } = useMembers({
    search: debouncedSearch || undefined,
    role: role || undefined,
    page,
    per_page: 20,
  });

  function handleSearchChange(value: string) {
    setSearch(value);
    setPage(1);
  }

  function handleRoleChange(value: string) {
    setRole(value);
    setPage(1);
  }

  const hasFilters = !!debouncedSearch || !!role;
  const isEmpty = !isLoading && !isError && data?.data.length === 0;
  const total = data?.meta.total ?? 0;
  const perPage = data?.meta.per_page ?? 20;

  return (
    <div className="px-4 py-6 max-w-2xl mx-auto md:max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold">Membros</h1>
        <Button asChild size="sm">
          <Link href="/members/new">
            <UserPlus className="h-4 w-4 mr-1.5" />
            Adicionar
          </Link>
        </Button>
      </div>

      {/* Filters */}
      <div className="flex gap-2 mb-5">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
          <Input
            placeholder="Buscar por nome ou e-mail…"
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="pl-9"
          />
        </div>
        <select
          value={role}
          onChange={(e) => handleRoleChange(e.target.value)}
          className="h-10 rounded-md border border-input bg-background px-3 text-sm text-foreground ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring"
          aria-label="Filtrar por perfil"
        >
          {ROLE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="grid gap-3 md:grid-cols-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <MemberCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar os membros. Verifique sua conexão e tente recarregar.
        </div>
      )}

      {/* Empty state */}
      {isEmpty && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <Users className="h-10 w-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {hasFilters
              ? "Nenhum membro encontrado com esses filtros."
              : "Nenhum membro cadastrado ainda. Adicione o primeiro membro da sua igreja."}
          </p>
          {!hasFilters && (
            <Button asChild size="sm">
              <Link href="/members/new">Adicionar primeiro membro</Link>
            </Button>
          )}
        </div>
      )}

      {/* List */}
      {!isLoading && !isError && data && data.data.length > 0 && (
        <>
          <div className="grid gap-3 md:grid-cols-2">
            {data.data.map((member) => (
              <MemberCard key={member.id} member={member} />
            ))}
          </div>

          {/* Pagination */}
          {total > perPage && (
            <div className="flex items-center justify-between mt-6 pt-4 border-t">
              <p className="text-sm text-muted-foreground">
                {total} {total === 1 ? "membro" : "membros"}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page === 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Anterior
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page * perPage >= total}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Próxima
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

"use client";

import { useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { useMe } from "@/hooks/useMembers";
import {
  useCategories,
  useCreateCategory,
  useUpdateCategory,
  useDeleteCategory,
} from "@/hooks/useInventory";
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
import type { ItemCategory } from "@/lib/api";

export default function CategoriesPage() {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState("");
  const [deletingCategory, setDeletingCategory] = useState<ItemCategory | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");

  const { data: meData } = useMe();
  const isLeadership = meData?.roles.some(
    (r) => r.base_profile === "leadership" || r.base_profile === "pastor"
  );

  const { data: categoriesData, isLoading, error } = useCategories();
  const { mutateAsync: doCreate, isPending: isCreating } = useCreateCategory();
  const { mutateAsync: doUpdate, isPending: isUpdating } = useUpdateCategory();
  const { mutateAsync: doDelete, isPending: isDeleting } = useDeleteCategory();

  const categories = categoriesData?.data ?? [];

  function startEdit(cat: ItemCategory) {
    setEditingId(cat.id);
    setEditingName(cat.name);
  }

  function cancelEdit() {
    setEditingId(null);
    setEditingName("");
  }

  async function saveEdit(id: string) {
    const name = editingName.trim();
    if (!name) return;
    try {
      await doUpdate({ id, data: { name } });
      toast.success("Categoria atualizada.");
      cancelEdit();
    } catch {
      toast.error("Não foi possível atualizar. Tente novamente.");
    }
  }

  async function confirmDelete() {
    if (!deletingCategory) return;
    try {
      await doDelete(deletingCategory.id);
      toast.success("Categoria excluída.");
      setDeletingCategory(null);
    } catch {
      toast.error("Não foi possível excluir. Tente novamente.");
    }
  }

  async function handleCreate() {
    const name = newName.trim();
    if (!name) return;
    try {
      await doCreate({ name });
      toast.success("Categoria criada.");
      setNewName("");
      setShowCreate(false);
    } catch {
      toast.error("Não foi possível criar. Tente novamente.");
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button asChild variant="ghost" size="sm">
            <Link href="/inventory">← Voltar</Link>
          </Button>
          <h1 className="text-xl font-semibold">Categorias</h1>
        </div>
        {isLeadership && (
          <Button size="sm" onClick={() => setShowCreate(true)}>
            + Nova categoria
          </Button>
        )}
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Error */}
      {error && !isLoading && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          Não foi possível carregar as categorias. Tente novamente.
        </div>
      )}

      {/* Empty */}
      {!isLoading && !error && categories.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <span className="text-5xl" aria-hidden="true">🗂️</span>
          <p className="text-sm text-muted-foreground">
            Nenhuma categoria cadastrada ainda.
          </p>
          {isLeadership && (
            <Button size="sm" onClick={() => setShowCreate(true)}>
              Criar primeira categoria
            </Button>
          )}
        </div>
      )}

      {/* Category list */}
      {!isLoading && !error && categories.length > 0 && (
        <div className="flex flex-col gap-2">
          {categories.map((cat) =>
            editingId === cat.id ? (
              <div
                key={cat.id}
                className="flex items-center gap-2 rounded-lg border bg-card p-3 shadow-sm"
              >
                <Input
                  className="flex-1"
                  value={editingName}
                  onChange={(e) => setEditingName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") saveEdit(cat.id);
                    if (e.key === "Escape") cancelEdit();
                  }}
                  autoFocus
                />
                <Button
                  size="sm"
                  disabled={isUpdating || !editingName.trim()}
                  onClick={() => saveEdit(cat.id)}
                >
                  {isUpdating ? "Salvando…" : "Salvar"}
                </Button>
                <Button size="sm" variant="ghost" onClick={cancelEdit}>
                  Cancelar
                </Button>
              </div>
            ) : (
              <div
                key={cat.id}
                className="flex items-center gap-3 rounded-lg border bg-card p-4 shadow-sm"
              >
                <span className="flex-1 text-sm font-medium">{cat.name}</span>
                {isLeadership && (
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => startEdit(cat)}
                    >
                      Editar
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => setDeletingCategory(cat)}
                    >
                      Excluir
                    </Button>
                  </div>
                )}
              </div>
            )
          )}
        </div>
      )}

      {/* Create dialog */}
      <Dialog
        open={showCreate}
        onOpenChange={(open) => {
          if (!open) {
            setShowCreate(false);
            setNewName("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Nova categoria</DialogTitle>
          </DialogHeader>
          <Input
            placeholder="Nome da categoria"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleCreate();
            }}
            autoFocus
          />
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button
              disabled={isCreating || !newName.trim()}
              onClick={handleCreate}
            >
              {isCreating ? "Criando…" : "Criar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <Dialog
        open={!!deletingCategory}
        onOpenChange={(open) => {
          if (!open) setDeletingCategory(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Excluir categoria?</DialogTitle>
            <DialogDescription>
              A categoria &ldquo;{deletingCategory?.name}&rdquo; será excluída.
              Os itens vinculados a ela perderão a categoria. Esta ação não pode
              ser desfeita.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancelar</Button>
            </DialogClose>
            <Button
              variant="destructive"
              disabled={isDeleting}
              onClick={confirmDelete}
            >
              {isDeleting ? "Excluindo…" : "Excluir"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

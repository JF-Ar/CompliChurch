"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import type { Resolver } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import Link from "next/link";
import { useCategories, useCreateItem } from "@/hooks/useInventory";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ApiError } from "@/lib/api";

const schema = z.object({
  name: z.string().min(1, "Nome é obrigatório"),
  item_type: z.enum(["asset", "consumable"]),
  category_id: z.string().optional(),
  description: z.string().optional(),
  asset_number: z.string().optional(),
  location: z.string().min(1, "Localização é obrigatória"),
  quantity: z.coerce.number().int().min(1, "Quantidade mínima é 1"),
  qty_min_alert: z.coerce.number().int().min(0).nullable().optional(),
  serial_number: z.string().optional(),
  notes: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

export default function NewInventoryItemPage() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);

  const { data: categoriesData } = useCategories();
  const { mutateAsync: createItem } = useCreateItem();

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema) as Resolver<FormValues>,
    defaultValues: { quantity: 1, item_type: "asset" },
  });

  const itemType = watch("item_type");

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      const item = await createItem({
        name: values.name,
        item_type: values.item_type,
        category_id: values.category_id || null,
        description: values.description || null,
        asset_number: values.asset_number || null,
        location: values.location,
        quantity: values.quantity,
        qty_min_alert: itemType === "consumable" ? (values.qty_min_alert ?? null) : null,
        serial_number: values.serial_number || null,
        notes: values.notes || null,
      });
      toast.success("Item cadastrado com sucesso.");
      router.push(`/inventory/${item.id}`);
    } catch (err) {
      const e = err as ApiError;
      setServerError(e?.error?.message ?? "Erro inesperado. Tente novamente.");
    }
  }

  return (
    <div className="flex flex-col gap-6 p-4 pb-24 md:pb-6 max-w-2xl mx-auto">
      <div className="flex items-center gap-3">
        <Link href="/inventory" className="text-sm text-muted-foreground hover:text-foreground">
          ← Patrimônio
        </Link>
      </div>
      <h1 className="text-xl font-semibold">Novo item</h1>

      <p className="text-xs text-muted-foreground -mt-4">Campos com * são obrigatórios</p>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
        {/* item_type */}
        <div className="flex flex-col gap-2">
          <Label>Tipo *</Label>
          <div className="flex gap-6">
            <label className="flex items-center gap-2 cursor-pointer min-h-[48px]">
              <input
                type="radio"
                value="asset"
                {...register("item_type")}
                className="h-4 w-4"
              />
              <span className="text-sm">Bem permanente</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer min-h-[48px]">
              <input
                type="radio"
                value="consumable"
                {...register("item_type")}
                className="h-4 w-4"
              />
              <span className="text-sm">Consumível</span>
            </label>
          </div>
          {errors.item_type && (
            <p className="text-xs text-destructive">{errors.item_type.message}</p>
          )}
        </div>

        {/* name */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="name">Nome *</Label>
          <Input id="name" {...register("name")} />
          {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
        </div>

        {/* category */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="category_id">Categoria</Label>
          <select
            id="category_id"
            {...register("category_id")}
            className="h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            <option value="">Sem categoria</option>
            {categoriesData?.data.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>

        {/* description */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="description">Descrição</Label>
          <textarea
            id="description"
            {...register("description")}
            rows={3}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        {/* asset_number */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="asset_number">Número do patrimônio</Label>
          <Input
            id="asset_number"
            {...register("asset_number")}
            placeholder="Deixar em branco para gerar automaticamente"
          />
        </div>

        {/* location */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="location">Localização *</Label>
          <Input id="location" {...register("location")} />
          {errors.location && (
            <p className="text-xs text-destructive">{errors.location.message}</p>
          )}
        </div>

        {/* quantity */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="quantity">Quantidade *</Label>
          <Input id="quantity" type="number" min={1} {...register("quantity")} />
          {errors.quantity && (
            <p className="text-xs text-destructive">{errors.quantity.message}</p>
          )}
        </div>

        {/* qty_min_alert — only for consumable */}
        {itemType === "consumable" && (
          <div className="flex flex-col gap-2">
            <Label htmlFor="qty_min_alert">Alerta de quantidade mínima</Label>
            <Input id="qty_min_alert" type="number" min={0} {...register("qty_min_alert")} />
            {errors.qty_min_alert && (
              <p className="text-xs text-destructive">{errors.qty_min_alert.message}</p>
            )}
          </div>
        )}

        {/* serial_number */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="serial_number">Número de série</Label>
          <Input id="serial_number" {...register("serial_number")} />
        </div>

        {/* notes */}
        <div className="flex flex-col gap-2">
          <Label htmlFor="notes">Observações</Label>
          <textarea
            id="notes"
            {...register("notes")}
            rows={3}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        {serverError && (
          <p className="text-sm text-destructive text-center">{serverError}</p>
        )}

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? "Salvando…" : "Cadastrar item"}
        </Button>
      </form>
    </div>
  );
}

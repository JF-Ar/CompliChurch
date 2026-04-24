"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface Props {
  memberName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isLoading: boolean;
}

export function DeactivateDialog({ memberName, open, onOpenChange, onConfirm, isLoading }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Desativar {memberName}?</DialogTitle>
          <DialogDescription>
            {memberName} perderá acesso ao sistema. O cadastro será mantido e a ação pode ser revertida
            pelo pastor. Esta ação não pode ser desfeita imediatamente.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={isLoading}>
              Cancelar
            </Button>
          </DialogClose>
          <Button variant="destructive" onClick={onConfirm} disabled={isLoading}>
            {isLoading ? "Desativando…" : "Desativar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

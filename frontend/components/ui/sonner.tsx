"use client";

import { Toaster as SonnerToaster } from "sonner";

export function Toaster() {
  return (
    <SonnerToaster
      position="bottom-center"
      richColors
      closeButton
      toastOptions={{ duration: 3000 }}
    />
  );
}

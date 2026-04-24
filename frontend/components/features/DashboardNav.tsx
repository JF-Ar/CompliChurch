"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Users, Calendar, CalendarDays, Package } from "lucide-react";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/members", label: "Membros", Icon: Users },
  { href: "/schedule", label: "Escala", Icon: Calendar },
  { href: "/agenda", label: "Agenda", Icon: CalendarDays },
  { href: "/inventory", label: "Patrimônio", Icon: Package },
];

export function DashboardNav() {
  const pathname = usePathname();

  return (
    <>
      {/* Bottom nav — mobile */}
      <nav className="fixed bottom-0 left-0 right-0 z-50 border-t bg-background md:hidden">
        <div className="flex">
          {NAV_ITEMS.map(({ href, label, Icon }) => {
            const active = pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "flex flex-1 flex-col items-center gap-1 py-2 text-xs transition-colors",
                  active ? "text-primary" : "text-muted-foreground hover:text-foreground"
                )}
              >
                <Icon className="h-5 w-5" />
                <span>{label}</span>
              </Link>
            );
          })}
        </div>
      </nav>

      {/* Sidebar — desktop */}
      <aside className="fixed inset-y-0 left-0 z-50 hidden w-56 flex-col border-r bg-background md:flex">
        <div className="px-4 py-5 text-sm font-semibold border-b">Igreja Organizada</div>
        <nav className="flex flex-col gap-1 p-2 pt-3">
          {NAV_ITEMS.map(({ href, label, Icon }) => {
            const active = pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                  active
                    ? "bg-accent text-foreground font-medium"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {label}
              </Link>
            );
          })}
        </nav>
      </aside>
    </>
  );
}

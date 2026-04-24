import Link from "next/link";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { RoleBadge } from "./RoleBadge";
import type { Member } from "@/lib/api";

export function MemberCard({ member }: { member: Member }) {
  const initials = member.name
    .split(" ")
    .slice(0, 2)
    .map((n) => n[0])
    .join("")
    .toUpperCase();

  return (
    <Link
      href={`/members/${member.id}`}
      className="flex items-center gap-3 rounded-lg border bg-card p-4 hover:bg-accent transition-colors active:scale-[0.99]"
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary select-none">
        {initials}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="font-medium text-sm truncate">{member.name}</p>
          {!member.is_active && (
            <Badge variant="muted" className="shrink-0">Inativo</Badge>
          )}
        </div>
        <p className="text-xs text-muted-foreground truncate">{member.email}</p>
        {member.roles.length > 0 && (
          <div className="flex gap-1 mt-1.5 flex-wrap">
            {member.roles.slice(0, 2).map((role) => (
              <RoleBadge key={role.id} role={role} />
            ))}
            {member.roles.length > 2 && (
              <Badge variant="outline">+{member.roles.length - 2}</Badge>
            )}
          </div>
        )}
      </div>

      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
    </Link>
  );
}

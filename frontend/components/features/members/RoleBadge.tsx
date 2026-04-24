import { Badge } from "@/components/ui/badge";
import type { RoleSummary } from "@/lib/api";

const PROFILE_VARIANT: Record<string, "default" | "secondary" | "warning" | "muted"> = {
  pastor: "default",
  leadership: "secondary",
  musician: "warning",
  member: "muted",
};

export function RoleBadge({ role }: { role: RoleSummary }) {
  return (
    <Badge variant={PROFILE_VARIANT[role.base_profile] ?? "secondary"}>
      {role.name}
    </Badge>
  );
}

import { Skeleton } from "@/components/ui/skeleton";

export function MemberCardSkeleton() {
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card p-4">
      <Skeleton className="h-10 w-10 rounded-full shrink-0" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-3 w-48" />
        <Skeleton className="h-5 w-20 rounded-full" />
      </div>
    </div>
  );
}

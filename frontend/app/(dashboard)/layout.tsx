import { DashboardNav } from "@/components/features/DashboardNav";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <DashboardNav />
      <main className="flex-1 pb-20 md:pb-0 md:pl-56">
        {children}
      </main>
    </div>
  );
}

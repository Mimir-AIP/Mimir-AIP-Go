import { Skeleton } from "@/components/ui/skeleton";
import { Card } from "@/components/ui/card";

export function CardListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
      {Array.from({ length: count }).map((_, i) => (
        <Card key={i} className="bg-navy text-white border-blue p-6">
          <Skeleton className="h-6 w-3/4 mb-4 bg-blue/20" />
          <Skeleton className="h-4 w-full mb-2 bg-blue/20" />
          <Skeleton className="h-4 w-2/3 mb-4 bg-blue/20" />
          <div className="flex gap-2 mt-4">
            <Skeleton className="h-9 w-20 bg-blue/20" />
            <Skeleton className="h-9 w-20 bg-blue/20" />
          </div>
        </Card>
      ))}
    </div>
  );
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-12 w-full bg-blue/20" />
      ))}
    </div>
  );
}

export function DetailsSkeleton() {
  return (
    <Card className="bg-navy text-white border-blue p-6">
      <Skeleton className="h-8 w-1/2 mb-6 bg-blue/20" />
      <div className="space-y-4">
        <div>
          <Skeleton className="h-4 w-24 mb-2 bg-blue/20" />
          <Skeleton className="h-6 w-full bg-blue/20" />
        </div>
        <div>
          <Skeleton className="h-4 w-24 mb-2 bg-blue/20" />
          <Skeleton className="h-6 w-full bg-blue/20" />
        </div>
        <div>
          <Skeleton className="h-4 w-24 mb-2 bg-blue/20" />
          <Skeleton className="h-32 w-full bg-blue/20" />
        </div>
      </div>
      <div className="flex gap-2 mt-6">
        <Skeleton className="h-10 w-24 bg-blue/20" />
        <Skeleton className="h-10 w-24 bg-blue/20" />
      </div>
    </Card>
  );
}

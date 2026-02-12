"use client";

import { JsonRenderer } from "@/components/json-renderer";
import { dashboardSchema } from "@/schemas/dashboard";

export default function DashboardPage() {
  return <JsonRenderer schema={dashboardSchema} />;
}

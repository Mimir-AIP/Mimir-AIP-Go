"use client";

import { JsonRenderer } from "@/components/json-renderer";
import { pipelinesListSchema } from "@/schemas/pipelines";

export default function PipelinesPage() {
  return <JsonRenderer schema={pipelinesListSchema} />;
}

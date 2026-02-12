"use client";

import { JsonRenderer } from "@/components/json-renderer";
import { modelsListSchema } from "@/schemas/models";

export default function ModelsPage() {
  return <JsonRenderer schema={modelsListSchema} />;
}

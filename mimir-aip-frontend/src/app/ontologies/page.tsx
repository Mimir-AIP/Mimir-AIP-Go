"use client";

import { JsonRenderer } from "@/components/json-renderer";
import { ontologiesListSchema } from "@/schemas/ontologies";

export default function OntologiesPage() {
  return <JsonRenderer schema={ontologiesListSchema} />;
}

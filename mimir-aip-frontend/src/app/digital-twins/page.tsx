"use client";

import { JsonRenderer } from "@/components/json-renderer";
import { digitalTwinsListSchema } from "@/schemas/digital-twins";

export default function DigitalTwinsPage() {
  return <JsonRenderer schema={digitalTwinsListSchema} />;
}

import { PageSchema } from "@/lib/ui-schema";

export const modelsListSchema: PageSchema = {
  title: "ML Models",
  description: "Monitor auto-trained model performance. Manage via chat.",
  components: [
    {
      type: "grid",
      dataSource: {
        endpoint: "/api/v1/models",
        transform: (data: any) => {
          // Handle response structure
          const models = data?.models || (Array.isArray(data) ? data : []);
          return models;
        },
      },
      cardTemplate: {
        title: "model_name",
        subtitle: "target_property",
        badge: {
          field: "is_active",
          colors: {
            true: "bg-green-900/40 text-green-400 border border-green-500",
            false: "bg-gray-900/40 text-gray-400 border border-gray-500",
          },
        },
        fields: [
          { label: "Algorithm", field: "algorithm" },
          { label: "Accuracy", field: "accuracy", format: "percentage" },
          { label: "Ontology", field: "ontology_id" },
          { label: "Created", field: "created_at", format: "date" },
        ],
        actions: [
          {
            label: "View Details",
            type: "link",
            href: "/models/{id}",
          },
        ],
      },
    },
  ],
};

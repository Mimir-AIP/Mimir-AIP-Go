import { PageSchema } from "@/lib/ui-schema";

export const pipelinesListSchema: PageSchema = {
  title: "Data Pipelines",
  description: "Manage data ingestion and processing pipelines",
  components: [
    {
      type: "grid",
      dataSource: {
        endpoint: "/api/v1/pipelines",
        transform: (data: any) => {
          // Handle both direct array and wrapped response
          const pipelines = Array.isArray(data) ? data : (data?.pipelines || []);
          return pipelines;
        },
      },
      cardTemplate: {
        title: "name",
        subtitle: "description",
        fields: [
          { label: "Type", field: "metadata.type" },
          { label: "Steps", field: "config.steps.length" },
          { label: "Created", field: "created_at", format: "date" },
        ],
        actions: [
          {
            label: "View Details",
            type: "link",
            href: "/pipelines/{id}",
          },
        ],
      },
    },
  ],
};



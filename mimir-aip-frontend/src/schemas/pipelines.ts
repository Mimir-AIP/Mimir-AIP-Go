import { PageSchema } from "@/lib/ui-schema";

export const pipelinesListSchema: PageSchema = {
  title: "Data Pipelines",
  description: "Manage data ingestion and processing pipelines",
  components: [
    {
      type: "table",
      dataSource: {
        endpoint: "/api/v1/pipelines",
        transform: (data: any) => {
          // Handle both direct array and wrapped response
          const pipelines = Array.isArray(data) ? data : (data?.pipelines || []);
          return pipelines;
        },
      },
      columns: [
        {
          key: "name",
          label: "Pipeline Name",
          type: "link",
          link: {
            href: "/pipelines/{id}",
          },
        },
        {
          key: "description",
          label: "Description",
          type: "text",
        },
        {
          key: "metadata.type",
          label: "Type",
          type: "text",
        },
        {
          key: "created_at",
          label: "Created",
          type: "date",
        },
      ],
      searchable: true,
      rowActions: [
        {
          label: "View",
          type: "link",
          href: "/pipelines/{id}",
          variant: "secondary",
        },
      ],
    },
  ],
};


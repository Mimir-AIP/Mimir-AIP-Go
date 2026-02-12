import { PageSchema } from "@/lib/ui-schema";

export const pipelinesListSchema: PageSchema = {
  title: "Data Pipelines",
  description: "Manage data ingestion and processing pipelines",
  components: [
    {
      type: "table",
      dataSource: {
        endpoint: "/api/v1/pipelines",
        transform: (data) => Array.isArray(data) ? data : [],
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
          key: "status",
          label: "Status",
          type: "badge",
          badge: {
            colors: {
              active: "bg-green-500/20 text-green-400",
              inactive: "bg-gray-500/20 text-gray-400",
              error: "bg-red-500/20 text-red-400",
            },
          },
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
          label: "Execute",
          type: "link",
          href: "/pipelines/{id}/execute",
          variant: "primary",
        },
        {
          label: "Edit",
          type: "link",
          href: "/pipelines/{id}/edit",
          variant: "secondary",
        },
      ],
    },
  ],
  actions: [
    {
      label: "Create Pipeline",
      type: "link",
      href: "/pipelines/new",
      variant: "primary",
    },
  ],
};

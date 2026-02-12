import { PageSchema } from "@/lib/ui-schema";

export const digitalTwinsListSchema: PageSchema = {
  title: "Digital Twins",
  description: "Manage and simulate digital representations of physical systems",
  components: [
    {
      type: "grid",
      dataSource: {
        endpoint: "/api/v1/twins",
        transform: (data) => {
          if (data?.success && data?.data?.twins) {
            return data.data.twins;
          }
          return Array.isArray(data) ? data : [];
        },
      },
      cardTemplate: {
        title: "name",
        subtitle: "description",
        badge: {
          field: "status",
          colors: {
            active: "bg-green-900/40 text-green-400 border border-green-500",
            inactive: "bg-gray-900/40 text-gray-400 border border-gray-500",
            error: "bg-red-900/40 text-red-400 border border-red-500",
          },
        },
        fields: [
          { label: "Ontology", field: "ontology_id" },
          { label: "Entities", field: "entity_count" },
          { label: "Created", field: "created_at", format: "date" },
        ],
        actions: [
          {
            label: "View Details",
            type: "link",
            href: "/digital-twins/{id}",
          },
        ],
      },
    },
  ],
  actions: [
    {
      label: "Create Digital Twin",
      type: "link",
      href: "/digital-twins/new",
      variant: "primary",
    },
  ],
};

import { PageSchema } from "@/lib/ui-schema";

export const ontologiesListSchema: PageSchema = {
  title: "Ontologies",
  description: "Monitor auto-generated knowledge schemas. Manage via chat.",
  components: [
    {
      type: "grid",
      dataSource: {
        endpoint: "/api/v1/ontology",
        transform: (data) => Array.isArray(data) ? data : [],
      },
      cardTemplate: {
        title: "name",
        subtitle: "description",
        badge: {
          field: "status",
          colors: {
            active: "bg-green-900/40 text-green-400 border border-green-500",
            deprecated: "bg-yellow-900/40 text-yellow-400 border border-yellow-500",
            draft: "bg-blue-900/40 text-blue-400 border border-blue-500",
          },
        },
        fields: [
          { label: "Version", field: "version" },
          { label: "Format", field: "format" },
          { label: "Created", field: "created_at", format: "date" },
        ],
        actions: [
          {
            label: "View Details",
            type: "link",
            href: "/ontologies/{id}",
          },
        ],
      },
      filters: [
        {
          name: "status",
          label: "All Status",
          type: "select",
          options: [
            { value: "", label: "All Status" },
            { value: "active", label: "Active" },
            { value: "deprecated", label: "Deprecated" },
            { value: "draft", label: "Draft" },
          ],
        },
      ],
    },
  ],
};

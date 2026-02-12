import { PageSchema } from "@/lib/ui-schema";

export const dashboardSchema: PageSchema = {
  title: "Dashboard",
  description: "System monitoring and overview",
  components: [
    {
      type: "stat",
      dataSource: {
        endpoint: "/api/v1/dashboard/stats",
        transform: (data) => {
          // Transform API response to stats format
          return {
            pipelines: data.pipelines?.length || 0,
            ontologies: data.ontologies?.filter((o: any) => o.status === 'active').length || 0,
            twins: data.twins?.length || 0,
            recentJobs: data.recentJobs?.length || 0,
          };
        },
      },
      cards: [
        {
          title: "Data Pipelines",
          value: "pipelines",
          icon: "git-branch",
          color: "orange",
          link: "/pipelines",
        },
        {
          title: "Ontologies",
          value: "ontologies",
          icon: "network",
          color: "blue-400",
          link: "/ontologies",
        },
        {
          title: "Digital Twins",
          value: "twins",
          icon: "copy",
          color: "purple-400",
          link: "/digital-twins",
        },
        {
          title: "Recent Executions",
          value: "recentJobs",
          icon: "clock",
          color: "yellow-400",
        },
      ],
    },
    {
      type: "table",
      title: "Recent Pipeline Executions",
      description: "Last 24 hours",
      dataSource: {
        endpoint: "/api/v1/jobs/recent",
        transform: (data) => Array.isArray(data) ? data.slice(0, 5) : [],
      },
      columns: [
        {
          key: "name",
          label: "Job Name",
          type: "text",
        },
        {
          key: "pipeline",
          label: "Pipeline",
          type: "text",
        },
        {
          key: "status",
          label: "Status",
          type: "badge",
          badge: {
            colors: {
              completed: "bg-green-500/20 text-green-400",
              failed: "bg-red-500/20 text-red-400",
              running: "bg-orange/20 text-orange",
              default: "bg-blue-500/20 text-blue-400",
            },
          },
        },
      ],
    },
  ],
};

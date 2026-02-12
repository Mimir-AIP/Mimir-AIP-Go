# JSON-Based UI Rendering System

## Overview

This document describes the new JSON-based UI rendering system implemented in Mimir AIP. This system dramatically simplifies frontend development by using declarative JSON schemas instead of manually coded React components.

## Architecture

### Key Components

1. **UI Schema Types** (`src/lib/ui-schema.ts`)
   - Defines TypeScript interfaces for all UI components
   - Provides type safety for JSON schemas

2. **JSON Renderer** (`src/components/json-renderer/`)
   - Main renderer that orchestrates component rendering
   - Individual renderers for each component type:
     - `FormRenderer.tsx` - Form inputs and validation
     - `TableRenderer.tsx` - Data tables with sorting/filtering
     - `GridRenderer.tsx` - Card-based grid layouts
     - `StatRenderer.tsx` - Statistics cards
     - `TabsRenderer.tsx` - Tabbed interfaces

3. **Page Schemas** (`src/schemas/`)
   - JSON schema definitions for each page
   - Declarative configuration of UI layout and data sources

## Benefits

### 1. Simplified Development
- **Before**: 200+ lines of React component code per page
- **After**: 30-50 lines of JSON configuration

### 2. Easy Maintenance
- Changes to UI require only JSON updates
- No need to understand React, hooks, or state management
- Single location for all UI definitions

### 3. AI-Friendly
- Agents with poor Next.js performance can easily modify JSON
- No complex React patterns to understand
- Clear, declarative structure

### 4. Consistent UI
- All pages use the same rendering logic
- Consistent styling and behavior
- Easier to maintain design system

## Usage

### Creating a New Page

1. **Define the page schema** in `src/schemas/[page-name].ts`:

```typescript
import { PageSchema } from "@/lib/ui-schema";

export const myPageSchema: PageSchema = {
  title: "My Page",
  description: "Page description",
  components: [
    // Component definitions
  ],
};
```

2. **Create the page component** in `src/app/[page-name]/page.tsx`:

```typescript
"use client";
import { JsonRenderer } from "@/components/json-renderer";
import { myPageSchema } from "@/schemas/my-page";

export default function MyPage() {
  return <JsonRenderer schema={myPageSchema} />;
}
```

That's it! No state management, no API calls, no complex React code.

## Component Types

### 1. Grid Component

Displays data in a card-based grid layout.

```typescript
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
        draft: "bg-blue-900/40 text-blue-400 border border-blue-500",
      },
    },
    fields: [
      { label: "Version", field: "version" },
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
      ],
    },
  ],
}
```

### 2. Table Component

Displays data in a table with sorting, filtering, and search.

```typescript
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
      link: { href: "/pipelines/{id}" },
    },
    {
      key: "status",
      label: "Status",
      type: "badge",
      badge: {
        colors: {
          active: "bg-green-500/20 text-green-400",
          error: "bg-red-500/20 text-red-400",
        },
      },
    },
  ],
  searchable: true,
  rowActions: [
    {
      label: "Execute",
      type: "link",
      href: "/pipelines/{id}/execute",
    },
  ],
}
```

### 3. Stat Component

Displays statistics as cards.

```typescript
{
  type: "stat",
  dataSource: {
    endpoint: "/api/v1/dashboard/stats",
    transform: (data) => ({
      pipelines: data.pipelines?.length || 0,
      ontologies: data.ontologies?.length || 0,
    }),
  },
  cards: [
    {
      title: "Data Pipelines",
      value: "pipelines",
      icon: "git-branch",
      color: "orange",
      link: "/pipelines",
    },
  ],
}
```

### 4. Form Component

Renders forms with validation.

```typescript
{
  type: "form",
  title: "Create Pipeline",
  fields: [
    {
      name: "name",
      label: "Pipeline Name",
      type: "text",
      required: true,
      placeholder: "Enter pipeline name",
    },
    {
      name: "type",
      label: "Pipeline Type",
      type: "select",
      options: [
        { value: "ingestion", label: "Data Ingestion" },
        { value: "processing", label: "Data Processing" },
      ],
    },
  ],
  submitAction: {
    label: "Create",
    action: "/api/v1/pipelines",
    method: "POST",
  },
}
```

### 5. Tabs Component

Organizes content into tabs.

```typescript
{
  type: "tabs",
  tabs: [
    {
      label: "Overview",
      content: { /* Another component schema */ },
    },
    {
      label: "Details",
      content: { /* Another component schema */ },
    },
  ],
}
```

## Data Source Configuration

### Basic API Call

```typescript
dataSource: {
  endpoint: "/api/v1/ontology",
}
```

### With Transform

Transform API response to match expected format:

```typescript
dataSource: {
  endpoint: "/api/v1/dashboard/stats",
  transform: (data) => ({
    pipelines: data.pipelines?.length || 0,
    ontologies: data.ontologies?.filter(o => o.status === 'active').length || 0,
  }),
}
```

### With Parameters

Pass query parameters or filters:

```typescript
dataSource: {
  endpoint: "/api/v1/ontology",
  params: {
    status: "active",
    limit: 10,
  },
}
```

## Field Types

### Text Fields
- `text` - Single line text input
- `textarea` - Multi-line text input
- `password` - Password input
- `number` - Numeric input

### Selection Fields
- `select` - Dropdown selection
- `checkbox` - Boolean checkbox

### Other Fields
- `date` - Date picker
- `file` - File upload

## Badge Colors

Badges use Tailwind CSS classes for styling:

```typescript
badge: {
  colors: {
    active: "bg-green-500/20 text-green-400",
    error: "bg-red-500/20 text-red-400",
    pending: "bg-orange/20 text-orange",
    default: "bg-blue-500/20 text-blue-400",
  },
}
```

## Icon Support

Available icons (from lucide-react):
- `git-branch` - Pipeline/branching icon
- `network` - Network/ontology icon
- `copy` - Copy/duplicate icon
- `clock` - Time/schedule icon
- `activity` - Activity/metrics icon
- `database` - Database/storage icon

## Migrated Pages

The following pages have been migrated to the JSON-based system:

1. **Dashboard** (`/dashboard`)
   - Stats cards showing system overview
   - Recent job executions table
   - Quick navigation links

2. **Ontologies List** (`/ontologies`)
   - Grid view of all ontologies
   - Status filtering
   - Card-based display with details

3. **Digital Twins List** (`/digital-twins`)
   - Grid view of all digital twins
   - Entity and relationship counts
   - Quick access to twin details

## Backend API Changes

### New Endpoint: Dashboard Stats

**Endpoint**: `GET /api/v1/dashboard/stats`

**Response**:
```json
{
  "pipelines": [...],
  "ontologies": [...],
  "twins": [...],
  "recentJobs": [...]
}
```

This endpoint aggregates data from multiple sources for the dashboard.

## Migration Guide

To migrate an existing page to the JSON-based system:

1. **Identify API calls** in the existing component
2. **Create a schema** in `src/schemas/[page-name].ts`
3. **Replace the page component** with `JsonRenderer`
4. **Test the page** to ensure functionality matches
5. **Remove old components** if no longer needed

### Example Migration

**Before** (200+ lines):
```typescript
export default function OntologiesPage() {
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);
  // ... lots more state and effects
  
  useEffect(() => {
    loadOntologies();
  }, []);
  
  // ... complex rendering logic
  return (
    <div className="space-y-6">
      {/* ... 150+ lines of JSX ... */}
    </div>
  );
}
```

**After** (7 lines):
```typescript
export default function OntologiesPage() {
  return <JsonRenderer schema={ontologiesListSchema} />;
}
```

## Future Enhancements

Potential improvements to the system:

1. **Server-side schemas**: Store schemas in database for runtime modification
2. **Schema editor UI**: Visual editor for creating/modifying schemas
3. **More component types**: Charts, graphs, calendars, etc.
4. **Conditional rendering**: Show/hide components based on data
5. **Validation rules**: More sophisticated form validation
6. **Caching**: Cache API responses for better performance

## Best Practices

1. **Keep schemas simple**: Don't over-complicate the JSON structure
2. **Use transforms wisely**: Transform API data to match expected format
3. **Consistent naming**: Use consistent field names across schemas
4. **Document complex transforms**: Add comments to explain non-obvious logic
5. **Test thoroughly**: Ensure data flows correctly through transforms
6. **Reuse components**: Create reusable schema fragments for common patterns

## Troubleshooting

### Component not rendering
- Check that the schema type is correct
- Verify the dataSource endpoint is valid
- Check browser console for errors

### API data not displaying
- Verify the API endpoint returns data
- Check the transform function (if used)
- Ensure field names match the API response

### Styles not applying
- Check Tailwind class names are correct
- Verify color mappings in badge configurations
- Test with browser dev tools to inspect CSS

## Conclusion

The JSON-based UI rendering system provides a powerful, flexible way to build UIs declaratively. It reduces complexity, improves maintainability, and makes the frontend more accessible to agents and developers with varying levels of React expertise.

For questions or issues, please refer to the component examples in `src/schemas/` or contact the development team.

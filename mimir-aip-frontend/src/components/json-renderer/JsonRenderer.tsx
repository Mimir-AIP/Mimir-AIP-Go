"use client";

import React from "react";
import { ComponentSchema, PageSchema } from "@/lib/ui-schema";
import { FormRenderer } from "./FormRenderer";
import { TableRenderer } from "./TableRenderer";
import { GridRenderer } from "./GridRenderer";
import { StatRenderer } from "./StatRenderer";
import { TabsRenderer } from "./TabsRenderer";

interface JsonRendererProps {
  schema: PageSchema;
}

export function JsonRenderer({ schema }: JsonRendererProps) {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-orange mb-2">{schema.title}</h1>
        {schema.description && (
          <p className="text-white/60">{schema.description}</p>
        )}
      </div>

      {/* Actions */}
      {schema.actions && schema.actions.length > 0 && (
        <div className="flex gap-3 mb-6">
          {schema.actions.map((action, idx) => (
            <ActionButton key={idx} action={action} />
          ))}
        </div>
      )}

      {/* Components */}
      {schema.components.map((component, idx) => (
        <ComponentRenderer key={idx} schema={component} />
      ))}
    </div>
  );
}

interface ComponentRendererProps {
  schema: ComponentSchema;
}

function ComponentRenderer({ schema }: ComponentRendererProps) {
  switch (schema.type) {
    case 'form':
      return <FormRenderer schema={schema} />;
    case 'table':
      return <TableRenderer schema={schema} />;
    case 'grid':
      return <GridRenderer schema={schema} />;
    case 'stat':
      return <StatRenderer schema={schema} />;
    case 'tabs':
      return <TabsRenderer schema={schema} />;
    default:
      return null;
  }
}

interface ActionButtonProps {
  action: any;
  data?: any;
}

export function ActionButton({ action, data }: ActionButtonProps) {
  const handleClick = async () => {
    if (action.confirmMessage) {
      if (!confirm(action.confirmMessage)) return;
    }

    if (action.href) {
      window.location.href = action.href;
    } else if (action.action) {
      // Handle API call
      // Implementation in individual renderers
    }
  };

  const variantClasses = {
    primary: 'bg-orange hover:bg-orange/80 text-white',
    secondary: 'bg-blue hover:bg-blue/80 text-white',
    danger: 'bg-red-500 hover:bg-red-600 text-white',
    ghost: 'bg-transparent hover:bg-blue/20 text-white border border-blue',
  };

  return (
    <button
      onClick={handleClick}
      className={`px-4 py-2 rounded transition-colors ${
        variantClasses[action.variant || 'primary']
      }`}
    >
      {action.label}
    </button>
  );
}

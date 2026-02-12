"use client";

import React, { useState } from "react";
import { TabSchema } from "@/lib/ui-schema";
import { Card } from "@/components/ui/card";
import { GridRenderer } from "./GridRenderer";
import { TableRenderer } from "./TableRenderer";
import { FormRenderer } from "./FormRenderer";
import { StatRenderer } from "./StatRenderer";

interface TabsRendererProps {
  schema: TabSchema;
}

export function TabsRenderer({ schema }: TabsRendererProps) {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <div className="space-y-4">
      {/* Tab Headers */}
      <div className="flex gap-2 border-b border-blue/30">
        {schema.tabs.map((tab, idx) => (
          <button
            key={idx}
            onClick={() => setActiveTab(idx)}
            className={`px-4 py-2 transition-colors ${
              activeTab === idx
                ? 'border-b-2 border-orange text-orange font-semibold'
                : 'text-white/60 hover:text-white'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div>
        {schema.tabs.map((tab, idx) => {
          if (idx !== activeTab) return null;
          
          const content = tab.content;
          switch (content.type) {
            case 'form':
              return <FormRenderer key={idx} schema={content} />;
            case 'table':
              return <TableRenderer key={idx} schema={content} />;
            case 'grid':
              return <GridRenderer key={idx} schema={content} />;
            case 'stat':
              return <StatRenderer key={idx} schema={content} />;
            default:
              return null;
          }
        })}
      </div>
    </div>
  );
}

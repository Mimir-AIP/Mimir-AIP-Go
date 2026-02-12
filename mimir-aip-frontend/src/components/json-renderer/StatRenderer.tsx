"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { StatSchema } from "@/lib/ui-schema";
import { Card } from "@/components/ui/card";
import { apiFetch } from "@/lib/api";
import { 
  GitBranch, 
  Network, 
  Copy, 
  Clock,
  Activity,
  Database
} from "lucide-react";

interface StatRendererProps {
  schema: StatSchema;
}

const iconMap: Record<string, any> = {
  'git-branch': GitBranch,
  'network': Network,
  'copy': Copy,
  'clock': Clock,
  'activity': Activity,
  'database': Database,
};

export function StatRenderer({ schema }: StatRendererProps) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await apiFetch(schema.dataSource.endpoint, {
        method: schema.dataSource.method || 'GET',
      });
      
      let result = response;
      if (schema.dataSource.transform) {
        result = schema.dataSource.transform(response);
      }
      
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  // Color mappings for Tailwind - must be complete class names
  const colorClasses: Record<string, { bg: string; text: string }> = {
    'orange': { bg: 'bg-orange-500/10', text: 'text-orange-500' },
    'blue-400': { bg: 'bg-blue-400/10', text: 'text-blue-400' },
    'purple-400': { bg: 'bg-purple-400/10', text: 'text-purple-400' },
    'yellow-400': { bg: 'bg-yellow-400/10', text: 'text-yellow-400' },
    'green-400': { bg: 'bg-green-400/10', text: 'text-green-400' },
  };

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[1, 2, 3, 4].map(i => (
          <Card key={i} className="bg-navy border-blue p-6 animate-pulse">
            <div className="h-8 bg-blue/30 rounded mb-2"></div>
            <div className="h-4 bg-blue/20 rounded"></div>
          </Card>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
        <p>Error loading stats: {error}</p>
      </Card>
    );
  }

  if (!data) return null;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {schema.cards.map((card, idx) => {
        const Icon = card.icon ? iconMap[card.icon] : Activity;
        const value = data[card.value];
        const CardWrapper = card.link ? Link : 'div';
        const colors = colorClasses[card.color || 'orange'];
        
        return (
          <CardWrapper key={idx} href={card.link || ''}>
            <Card className={`bg-navy border-blue p-6 ${card.link ? 'hover:border-orange/50 transition-colors cursor-pointer' : ''}`}>
              <div className="flex items-center justify-between mb-4">
                {Icon && (
                  <div className={`w-10 h-10 p-2 ${colors.bg} rounded-lg`}>
                    <Icon className={`w-full h-full ${colors.text}`} />
                  </div>
                )}
                <span className="text-xs text-white/40 uppercase tracking-wide">Active</span>
              </div>
              <p className="text-4xl font-bold text-white mb-1">{value}</p>
              <p className="text-sm text-white/60">{card.title}</p>
            </Card>
          </CardWrapper>
        );
      })}
    </div>
  );
}

"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { GridSchema } from "@/lib/ui-schema";
import { Card } from "@/components/ui/card";
import { apiFetch } from "@/lib/api";

interface GridRendererProps {
  schema: GridSchema;
}

export function GridRenderer({ schema }: GridRendererProps) {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Record<string, any>>({});

  useEffect(() => {
    loadData();
  }, [filters]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const url = schema.dataSource.endpoint;
      const params = { ...schema.dataSource.params, ...filters };
      const queryString = new URLSearchParams(params).toString();
      const fullUrl = queryString ? `${url}?${queryString}` : url;
      
      const response = await apiFetch(fullUrl, {
        method: schema.dataSource.method || 'GET',
      });
      
      let result = response;
      if (schema.dataSource.transform) {
        result = schema.dataSource.transform(response);
      }
      
      setData(Array.isArray(result) ? result : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
      setData([]);
    } finally {
      setLoading(false);
    }
  };

  const getBadgeClass = (value: string, colors: Record<string, string>) => {
    const color = colors[value] || colors['default'] || 'bg-gray-500';
    return `px-2 py-1 text-xs font-semibold rounded-full ${color}`;
  };
  
  const getNestedValue = (obj: any, path: string) => {
    return path.split('.').reduce((current, key) => current?.[key], obj);
  };

  const formatValue = (value: any, format?: string) => {
    if (value === null || value === undefined) return '-';
    if (format === 'date') {
      return new Date(value).toLocaleDateString();
    }
    if (format === 'percentage') {
      return `${(value * 100).toFixed(2)}%`;
    }
    return value;
  };

  if (loading) {
    return (
      <div className="space-y-6">
        {schema.title && <h2 className="text-2xl font-bold text-orange">{schema.title}</h2>}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <Card key={i} className="bg-navy border-blue p-6 animate-pulse h-40"></Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
        <p>Error: {error}</p>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          {schema.title && <h2 className="text-2xl font-bold text-orange">{schema.title}</h2>}
          {schema.description && <p className="text-white/60 mt-1">{schema.description}</p>}
        </div>
        
        {/* Filters */}
        {schema.filters && schema.filters.length > 0 && (
          <div className="flex gap-3">
            {schema.filters.map((field) => (
              <select
                key={field.name}
                value={filters[field.name] || ''}
                onChange={(e) => setFilters({ ...filters, [field.name]: e.target.value })}
                className="bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
              >
                <option value="">{field.label}</option>
                {field.options?.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            ))}
            <button
              onClick={loadData}
              className="bg-blue hover:bg-orange text-white px-4 py-2 rounded border border-blue"
            >
              Refresh
            </button>
          </div>
        )}
      </div>

      {/* Grid */}
      {data.length === 0 ? (
        <Card className="bg-navy border-blue p-12 text-center">
          <p className="text-white/60">No items found</p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {data.map((item, idx) => {
            const title = item[schema.cardTemplate.title];
            const subtitle = schema.cardTemplate.subtitle ? item[schema.cardTemplate.subtitle] : null;
            const badgeValue = schema.cardTemplate.badge ? item[schema.cardTemplate.badge.field] : null;
            const badgeDisplay = badgeValue === true ? 'Active' : badgeValue === false ? 'Inactive' : badgeValue;
            
            return (
              <Card key={item.id || idx} className="bg-navy border-blue p-6 h-full">
                <div className="flex justify-between items-start mb-3">
                  <h3 className="text-xl font-bold text-orange">{title}</h3>
                  {schema.cardTemplate.badge && badgeValue !== null && badgeValue !== undefined && (
                    <span className={getBadgeClass(
                      String(badgeValue),
                      schema.cardTemplate.badge.colors
                    )}>
                      {badgeDisplay}
                    </span>
                  )}
                </div>
                
                {subtitle && (
                  <p className="text-sm text-white/60 mb-4 line-clamp-2">{subtitle}</p>
                )}
                
                <div className="space-y-2 text-sm">
                  {schema.cardTemplate.fields.map((field) => {
                    const value = getNestedValue(item, field.field);
                    return (
                      <div key={field.field} className="flex justify-between">
                        <span className="text-white/40">{field.label}</span>
                        <span className="text-white">{formatValue(value, field.format)}</span>
                      </div>
                    );
                  })}
                </div>
                
                {schema.cardTemplate.actions && schema.cardTemplate.actions.length > 0 && (
                  <div className="mt-4 pt-4 border-t border-blue/30 flex gap-2">
                    {schema.cardTemplate.actions.map((action, aidx) => {
                      const href = action.href?.replace('{id}', item.id);
                      return (
                        <Link
                          key={aidx}
                          href={href || '#'}
                          className="text-xs bg-blue hover:bg-orange text-white px-3 py-1.5 rounded transition-colors"
                        >
                          {action.label}
                        </Link>
                      );
                    })}
                  </div>
                )}
              </Card>
            );
          })}
        </div>
      )}

      {/* Summary */}
      <div className="mt-6 p-4 bg-blue/20 rounded-lg">
        <p className="text-sm text-white/60">
          Total: {data.length} item{data.length !== 1 ? 's' : ''}
        </p>
      </div>
    </div>
  );
}

"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { TableSchema } from "@/lib/ui-schema";
import { Card } from "@/components/ui/card";
import { apiFetch } from "@/lib/api";

interface TableRendererProps {
  schema: TableSchema;
}

export function TableRenderer({ schema }: TableRendererProps) {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Record<string, any>>({});
  const [searchTerm, setSearchTerm] = useState('');

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

  const formatValue = (value: any, column: any) => {
    if (!value) return '-';
    if (column.type === 'date') {
      return new Date(value).toLocaleDateString();
    }
    if (column.format) {
      return column.format(value);
    }
    return value;
  };

  const filteredData = data.filter(item => {
    if (!searchTerm) return true;
    return schema.columns.some(col => {
      const value = item[col.key];
      return value && String(value).toLowerCase().includes(searchTerm.toLowerCase());
    });
  });

  if (loading) {
    return (
      <Card className="bg-navy border-blue p-6">
        {schema.title && <h2 className="text-2xl font-bold text-orange mb-4">{schema.title}</h2>}
        <div className="animate-pulse space-y-3">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-12 bg-blue/20 rounded"></div>
          ))}
        </div>
      </Card>
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
    <div className="space-y-4">
      {/* Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          {schema.title && <h2 className="text-2xl font-bold text-orange">{schema.title}</h2>}
          {schema.description && <p className="text-white/60">{schema.description}</p>}
        </div>
        
        <div className="flex gap-3">
          {/* Search */}
          {schema.searchable && (
            <input
              type="text"
              placeholder="Search..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
            />
          )}
          
          {/* Filters */}
          {schema.filters?.map((field) => (
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
      </div>

      {/* Table */}
      <Card className="bg-navy border-blue overflow-hidden">
        {filteredData.length === 0 ? (
          <div className="p-12 text-center text-white/60">No data found</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue/30">
                  {schema.columns.map((col) => (
                    <th key={col.key} className="text-left p-4 text-white/60 font-semibold">
                      {col.label}
                    </th>
                  ))}
                  {schema.rowActions && schema.rowActions.length > 0 && (
                    <th className="text-right p-4 text-white/60 font-semibold">Actions</th>
                  )}
                </tr>
              </thead>
              <tbody>
                {filteredData.map((item, idx) => (
                  <tr key={item.id || idx} className="border-b border-blue/10 hover:bg-blue/10">
                    {schema.columns.map((col) => {
                      const value = item[col.key];
                      
                      return (
                        <td key={col.key} className="p-4 text-white">
                          {col.type === 'badge' && col.badge ? (
                            <span className={getBadgeClass(value, col.badge.colors)}>
                              {value}
                            </span>
                          ) : col.type === 'link' && col.link ? (
                            <Link
                              href={col.link.href.replace('{id}', item.id)}
                              className="text-orange hover:underline"
                            >
                              {formatValue(value, col)}
                            </Link>
                          ) : (
                            formatValue(value, col)
                          )}
                        </td>
                      );
                    })}
                    {schema.rowActions && schema.rowActions.length > 0 && (
                      <td className="p-4 text-right">
                        <div className="flex gap-2 justify-end">
                          {schema.rowActions.map((action, aidx) => {
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
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

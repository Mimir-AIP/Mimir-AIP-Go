"use client";
import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { type ExecutionLog, type ExecutionLogEntry } from "@/lib/api";

interface LogViewerProps {
  logs: ExecutionLog[];
  loading?: boolean;
  onRefresh?: () => void;
}

const LOG_LEVEL_COLORS: Record<string, string> = {
  INFO: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  WARN: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  ERROR: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  DEBUG: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
};

const STATUS_COLORS: Record<string, string> = {
  running: "bg-blue-100 text-blue-800",
  completed: "bg-green-100 text-green-800",
  failed: "bg-red-100 text-red-800",
  pending: "bg-gray-100 text-gray-800",
};

export function LogViewer({ logs, loading, onRefresh }: LogViewerProps) {
  const [expandedLogs, setExpandedLogs] = useState<Set<string>>(new Set());
  const [filterLevel, setFilterLevel] = useState<string>("ALL");

  const toggleLog = (logId: string) => {
    const newExpanded = new Set(expandedLogs);
    if (newExpanded.has(logId)) {
      newExpanded.delete(logId);
    } else {
      newExpanded.add(logId);
    }
    setExpandedLogs(newExpanded);
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString();
    } catch {
      return timestamp;
    }
  };

  const filterEntries = (entries: ExecutionLogEntry[]) => {
    if (filterLevel === "ALL") return entries;
    return entries.filter((entry) => entry.level === filterLevel);
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
        <div className="h-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
      </div>
    );
  }

  if (!logs || logs.length === 0) {
    return (
      <Card className="p-8 text-center">
        <p className="text-gray-500 dark:text-gray-400">No execution logs found</p>
        {onRefresh && (
          <Button onClick={onRefresh} variant="outline" className="mt-4">
            Refresh
          </Button>
        )}
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex gap-2">
          <Button
            onClick={() => setFilterLevel("ALL")}
            variant={filterLevel === "ALL" ? "default" : "outline"}
            size="sm"
          >
            All
          </Button>
          {["INFO", "WARN", "ERROR", "DEBUG"].map((level) => (
            <Button
              key={level}
              onClick={() => setFilterLevel(level)}
              variant={filterLevel === level ? "default" : "outline"}
              size="sm"
            >
              {level}
            </Button>
          ))}
        </div>
        {onRefresh && (
          <Button onClick={onRefresh} variant="outline" size="sm">
            Refresh
          </Button>
        )}
      </div>

      <div className="space-y-3">
        {logs.map((log) => {
          const isExpanded = expandedLogs.has(log.id);
          const filteredEntries = filterEntries(log.entries || []);
          const duration = log.ended_at
            ? Math.round(
                (new Date(log.ended_at).getTime() -
                  new Date(log.started_at).getTime()) /
                  1000
              )
            : null;

          return (
            <Card key={log.id} className="overflow-hidden">
              <div
                className="p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                onClick={() => toggleLog(log.id)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <svg
                      className={`w-5 h-5 transition-transform ${
                        isExpanded ? "rotate-90" : ""
                      }`}
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">Execution {log.id.slice(0, 8)}</span>
                        <Badge className={STATUS_COLORS[log.status] || STATUS_COLORS.pending}>
                          {log.status}
                        </Badge>
                      </div>
                      <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                        {formatTimestamp(log.started_at)}
                        {duration && <span> • {duration}s</span>}
                        {log.pipeline_id && <span> • Pipeline: {log.pipeline_id}</span>}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{filteredEntries.length} entries</Badge>
                  </div>
                </div>
              </div>

              {isExpanded && (
                <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
                  {filteredEntries.length === 0 ? (
                    <div className="p-4 text-center text-gray-500 dark:text-gray-400">
                      No entries match the selected filter
                    </div>
                  ) : (
                    <div className="divide-y divide-gray-200 dark:divide-gray-700">
                      {filteredEntries.map((entry, idx) => (
                        <div key={idx} className="p-3 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors">
                          <div className="flex items-start gap-3">
                            <Badge className={LOG_LEVEL_COLORS[entry.level] || LOG_LEVEL_COLORS.INFO}>
                              {entry.level}
                            </Badge>
                            <div className="flex-1 min-w-0">
                              <div className="text-sm font-mono">{entry.message}</div>
                              <div className="flex items-center gap-3 mt-1 text-xs text-gray-500 dark:text-gray-400">
                                <span>{formatTimestamp(entry.timestamp)}</span>
                                {entry.step_name && <span>Step: {entry.step_name}</span>}
                                {entry.plugin_name && <span>Plugin: {entry.plugin_name}</span>}
                              </div>
                              {entry.data && Object.keys(entry.data).length > 0 && (
                                <details className="mt-2">
                                  <summary className="cursor-pointer text-xs text-blue-600 dark:text-blue-400 hover:underline">
                                    View data
                                  </summary>
                                  <pre className="mt-2 p-2 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto">
                                    {JSON.stringify(entry.data, null, 2)}
                                  </pre>
                                </details>
                              )}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </Card>
          );
        })}
      </div>
    </div>
  );
}

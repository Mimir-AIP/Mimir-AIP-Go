"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "sonner";
import {
  ArrowLeft,
  Table,
  Database,
  FileText,
  Loader2,
  CheckCircle,
  AlertCircle,
  Eye,
  EyeOff,
} from "lucide-react";

// Data preview response
interface DataPreviewResponse {
  upload_id: string;
  plugin_type: string;
  plugin_name: string;
  data: any;
  preview_rows: number;
  message: string;
}

// Column selection state
interface ColumnSelection {
  name: string;
  selected: boolean;
  dataType: string;
  sampleValues: any[];
}

export default function DataPreviewPage() {
  const params = useParams();
  const router = useRouter();
  const uploadId = params.id as string;

  const [loading, setLoading] = useState(true);
  const [previewData, setPreviewData] = useState<DataPreviewResponse | null>(null);
  const [columns, setColumns] = useState<ColumnSelection[]>([]);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    if (uploadId) {
      loadPreview();
    }
  }, [uploadId]);

  const loadPreview = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/v1/data/preview", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          upload_id: uploadId,
          max_rows: 50, // Preview first 50 rows
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to load preview");
      }

      const data = await response.json();
      setPreviewData(data);

      // Initialize column selections
      if (data.data?.columns && data.data?.rows) {
        const columnSelections: ColumnSelection[] = data.data.columns.map((col: string, index: number) => {
          // Infer data type and get sample values
          const sampleValues = data.data.rows.slice(0, 5).map((row: any) => row[col]).filter((val: any) => val != null);
          const dataType = inferDataType(sampleValues);

          return {
            name: col,
            selected: true, // Default to selected
            dataType,
            sampleValues,
          };
        });
        setColumns(columnSelections);
      }

    } catch (error) {
      console.error("Failed to load preview:", error);
      toast.error("Failed to load data preview");
    } finally {
      setLoading(false);
    }
  };

  const inferDataType = (values: any[]): string => {
    if (values.length === 0) return "unknown";

    const types = values.map(val => {
      if (typeof val === "number") return "number";
      if (typeof val === "boolean") return "boolean";
      if (typeof val === "string") {
        // Check if it looks like a date
        if (/^\d{4}-\d{2}-\d{2}/.test(val) || /^\d{2}\/\d{2}\/\d{4}/.test(val)) {
          return "date";
        }
        return "string";
      }
      return "unknown";
    });

    // Return most common type
    const typeCounts = types.reduce((acc, type) => {
      acc[type] = (acc[type] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    return Object.entries(typeCounts).sort(([,a], [,b]) => b - a)[0][0];
  };

  const toggleColumn = (columnName: string) => {
    setColumns(prev =>
      prev.map(col =>
        col.name === columnName
          ? { ...col, selected: !col.selected }
          : col
      )
    );
  };

  const handleGenerateOntology = async () => {
    const selectedColumns = columns.filter(col => col.selected).map(col => col.name);

    if (selectedColumns.length === 0) {
      toast.error("Please select at least one column");
      return;
    }

    setGenerating(true);
    try {
      const response = await fetch("/api/v1/data/select", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          upload_id: uploadId,
          selected_columns: selectedColumns,
          // Additional mappings could be added here
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Failed to generate ontology");
      }

      const result = await response.json();
      toast.success("Ontology generated successfully!");

      // Redirect to ontology page or show result
      router.push(`/ontologies`);

    } catch (error) {
      console.error("Ontology generation failed:", error);
      toast.error(error instanceof Error ? error.message : "Ontology generation failed");
    } finally {
      setGenerating(false);
    }
  };

  const getDataTypeIcon = (dataType: string) => {
    switch (dataType) {
      case "number":
        return <span className="text-blue-500">#</span>;
      case "string":
        return <span className="text-green-500">T</span>;
      case "boolean":
        return <span className="text-purple-500">✓</span>;
      case "date":
        return <span className="text-orange-500">📅</span>;
      default:
        return <span className="text-gray-500">?</span>;
    }
  };

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          <span className="ml-2 text-lg">Loading data preview...</span>
        </div>
      </div>
    );
  }

  if (!previewData) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <div className="text-center">
          <AlertCircle className="h-16 w-16 mx-auto text-red-500 mb-4" />
          <h2 className="text-2xl font-bold mb-2">Failed to Load Preview</h2>
          <p className="text-gray-600 mb-6">Unable to load the data preview. Please try again.</p>
          <Button onClick={() => router.back()}>Go Back</Button>
        </div>
      </div>
    );
  }

  const { data } = previewData;
  const hasData = data?.rows && data.rows.length > 0;

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <Link href="/data/upload" className="text-orange hover:underline mb-4 inline-block">
          ← Back to Upload
        </Link>
        <h1 className="text-3xl font-bold text-orange">Data Preview</h1>
        <p className="text-gray-400 mt-1">
          Review your data and select columns for ontology generation
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Column Selection */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Database className="h-5 w-5 mr-2" />
                Column Selection
              </CardTitle>
              <CardDescription>
                Choose which columns to include in your ontology
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {columns.map((column) => (
                  <div key={column.name} className="flex items-center space-x-3 p-3 border rounded-lg">
                    <Checkbox
                      id={`col-${column.name}`}
                      checked={column.selected}
                      onCheckedChange={() => toggleColumn(column.name)}
                    />
                    <div className="flex-1">
                      <label
                        htmlFor={`col-${column.name}`}
                        className="font-medium cursor-pointer flex items-center"
                      >
                        {column.selected ? (
                          <Eye className="h-4 w-4 mr-2 text-green-500" />
                        ) : (
                          <EyeOff className="h-4 w-4 mr-2 text-gray-400" />
                        )}
                        {column.name}
                      </label>
                      <div className="flex items-center mt-1 text-sm text-gray-500">
                        {getDataTypeIcon(column.dataType)}
                        <span className="ml-1 capitalize">{column.dataType}</span>
                        <span className="ml-2">• {column.sampleValues.length} samples</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              <div className="mt-6 space-y-3">
                <Button
                  onClick={handleGenerateOntology}
                  disabled={generating || columns.filter(c => c.selected).length === 0}
                  className="w-full"
                >
                  {generating ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Generating Ontology...
                    </>
                  ) : (
                    <>
                      <Database className="h-4 w-4 mr-2" />
                      Generate Ontology
                    </>
                  )}
                </Button>

                <div className="text-sm text-gray-500 text-center">
                  {columns.filter(c => c.selected).length} of {columns.length} columns selected
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Data Preview */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Table className="h-5 w-5 mr-2" />
                Data Preview
              </CardTitle>
              <CardDescription>
                {hasData
                  ? `Showing ${data.rows.length} of ${data.row_count || data.rows.length} rows, ${data.columns?.length || 0} columns`
                  : "No data available for preview"
                }
              </CardDescription>
            </CardHeader>
            <CardContent>
              {hasData ? (
                <div className="overflow-x-auto">
                  <table className="w-full border-collapse border border-gray-300">
                    <thead>
                      <tr className="bg-gray-50">
                        {data.columns?.map((col: string, index: number) => {
                          const column = columns.find(c => c.name === col);
                          const isSelected = column?.selected ?? false;
                          return (
                            <th
                              key={col}
                              className={`border border-gray-300 px-4 py-2 text-left text-sm font-medium ${
                                isSelected ? 'bg-green-50 text-green-800' : 'text-gray-500'
                              }`}
                            >
                              <div className="flex items-center">
                                {getDataTypeIcon(column?.dataType || 'unknown')}
                                <span className="ml-2">{col}</span>
                                {isSelected && <CheckCircle className="h-4 w-4 ml-2 text-green-500" />}
                              </div>
                            </th>
                          );
                        })}
                      </tr>
                    </thead>
                    <tbody>
                      {data.rows.slice(0, 10).map((row: any, rowIndex: number) => (
                        <tr key={rowIndex} className="hover:bg-gray-50">
                          {data.columns?.map((col: string) => (
                            <td key={col} className="border border-gray-300 px-4 py-2 text-sm">
                              {row[col] != null ? String(row[col]) : <span className="text-gray-400">null</span>}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>

                  {data.rows.length > 10 && (
                    <div className="mt-4 text-center text-sm text-gray-500">
                      Showing first 10 rows of {data.rows.length} total rows
                    </div>
                  )}
                </div>
              ) : (
                <div className="text-center py-12">
                  <FileText className="h-16 w-16 mx-auto text-gray-400 mb-4" />
                  <h3 className="text-xl font-semibold mb-2">No Data Available</h3>
                  <p className="text-gray-600">
                    Unable to preview data. The file may be empty or corrupted.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Data Statistics */}
          {hasData && (
            <Card className="mt-6">
              <CardHeader>
                <CardTitle>Data Statistics</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-blue-600">{data.columns?.length || 0}</div>
                    <div className="text-sm text-gray-600">Columns</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-600">{data.rows?.length || 0}</div>
                    <div className="text-sm text-gray-600">Rows</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-purple-600">
                      {columns.filter(c => c.dataType === 'string').length}
                    </div>
                    <div className="text-sm text-gray-600">Text Columns</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-orange-600">
                      {columns.filter(c => c.dataType === 'number').length}
                    </div>
                    <div className="text-sm text-gray-600">Numeric Columns</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
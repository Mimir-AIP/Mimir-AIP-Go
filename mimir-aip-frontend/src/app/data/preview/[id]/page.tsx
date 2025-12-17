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
  ChevronDown,
  ChevronUp,
  TrendingUp,
  TrendingDown,
  Activity,
  BarChart3,
  GitBranch,
} from "lucide-react";

// ==================== TYPE DEFINITIONS ====================

// Profiling Types
interface ValueFrequency {
  value: any;
  count: number;
  frequency: number;
}

interface ColumnProfile {
  column_name: string;
  data_type: string;
  total_count: number;
  distinct_count: number;
  distinct_percent: number;
  null_count: number;
  null_percent: number;
  min_value?: any;
  max_value?: any;
  mean?: number;
  median?: number;
  std_dev?: number;
  min_length?: number;
  max_length?: number;
  avg_length?: number;
  top_values: ValueFrequency[];
  data_quality_score: number;
  quality_issues: string[];
}

interface DataProfileSummary {
  total_rows: number;
  total_columns: number;
  total_distinct_values: number;
  overall_quality_score: number;
  suggested_primary_keys: string[];
  column_profiles: ColumnProfile[];
}

// Data preview response
interface DataPreviewResponse {
  upload_id: string;
  plugin_type: string;
  plugin_name: string;
  data: any;
  preview_rows: number;
  message: string;
  profile?: DataProfileSummary;
}

// Column selection state
interface ColumnSelection {
  name: string;
  selected: boolean;
  dataType: string;
  sampleValues: any[];
}

// ==================== MAIN COMPONENT ====================

export default function DataPreviewPage() {
  const params = useParams();
  const router = useRouter();
  const uploadId = params.id as string;

  const [loading, setLoading] = useState(true);
  const [previewData, setPreviewData] = useState<DataPreviewResponse | null>(null);
  const [columns, setColumns] = useState<ColumnSelection[]>([]);
  const [generating, setGenerating] = useState(false);
  const [enableProfiling, setEnableProfiling] = useState(true);
  const [createTwin, setCreateTwin] = useState(false);
  const [expandedProfiles, setExpandedProfiles] = useState<Set<string>>(new Set());

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
          max_rows: 50,
          profile: enableProfiling,
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

  const toggleProfileExpansion = (columnName: string) => {
    setExpandedProfiles(prev => {
      const newSet = new Set(prev);
      if (newSet.has(columnName)) {
        newSet.delete(columnName);
      } else {
        newSet.add(columnName);
      }
      return newSet;
    });
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
          create_twin: createTwin,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Failed to generate ontology");
      }

      const result = await response.json();
      
      if (createTwin && result.digital_twin) {
        toast.success(`Ontology and Digital Twin created successfully!`);
      } else {
        toast.success("Ontology generated successfully!");
      }

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

  const getQualityBadge = (score: number) => {
    if (score >= 0.8) {
      return <Badge className="bg-green-600">Good ({(score * 100).toFixed(0)}%)</Badge>;
    } else if (score >= 0.6) {
      return <Badge className="bg-yellow-600">Fair ({(score * 100).toFixed(0)}%)</Badge>;
    } else {
      return <Badge className="bg-red-600">Poor ({(score * 100).toFixed(0)}%)</Badge>;
    }
  };

  const getQualityColor = (score: number) => {
    if (score >= 0.8) return "text-green-600";
    if (score >= 0.6) return "text-yellow-600";
    return "text-red-600";
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

  const { data, profile } = previewData;
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

      {/* Data Quality Summary Card */}
      {profile && (
        <Card className="mb-6 border-2 border-blue-200">
          <CardHeader>
            <CardTitle className="flex items-center">
              <Activity className="h-5 w-5 mr-2 text-blue-500" />
              Data Quality Summary
            </CardTitle>
            <CardDescription>
              Overall assessment of your dataset health
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
              <div className="text-center">
                <div className={`text-3xl font-bold ${getQualityColor(profile.overall_quality_score)}`}>
                  {(profile.overall_quality_score * 100).toFixed(0)}%
                </div>
                <div className="text-sm text-gray-600">Overall Quality</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-blue-600">{profile.total_rows.toLocaleString()}</div>
                <div className="text-sm text-gray-600">Total Rows</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-purple-600">{profile.total_columns}</div>
                <div className="text-sm text-gray-600">Total Columns</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-green-600">{profile.total_distinct_values.toLocaleString()}</div>
                <div className="text-sm text-gray-600">Distinct Values</div>
              </div>
            </div>

            {profile.suggested_primary_keys && profile.suggested_primary_keys.length > 0 && (
              <div className="mt-4 p-3 bg-blue-50 rounded-lg border border-blue-200">
                <div className="flex items-center mb-2">
                  <Database className="h-4 w-4 mr-2 text-blue-600" />
                  <span className="font-semibold text-sm text-blue-900">Suggested Primary Keys</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {profile.suggested_primary_keys.map((key, idx) => (
                    <Badge key={idx} className="bg-blue-600">
                      {key}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

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
                {columns.map((column) => {
                  const columnProfile = profile?.column_profiles.find(p => p.column_name === column.name);
                  const isPrimaryKey = profile?.suggested_primary_keys.includes(column.name);

                  return (
                    <div key={column.name} className={`p-3 border rounded-lg ${isPrimaryKey ? 'border-blue-400 bg-blue-50' : ''}`}>
                      <div className="flex items-center space-x-3">
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
                            {isPrimaryKey && (
                              <Badge className="ml-2 bg-blue-600 text-xs">PK</Badge>
                            )}
                          </label>
                          <div className="flex items-center mt-1 text-sm text-gray-500">
                            {getDataTypeIcon(column.dataType)}
                            <span className="ml-1 capitalize">{column.dataType}</span>
                          </div>
                          {columnProfile && (
                            <div className="mt-2">
                              {getQualityBadge(columnProfile.data_quality_score)}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="mt-6 space-y-3">
                {/* Profiling Checkbox */}
                <div className="flex items-center space-x-2 p-3 border rounded-lg bg-gray-50">
                  <Checkbox
                    id="enable-profiling"
                    checked={enableProfiling}
                    onCheckedChange={(checked) => setEnableProfiling(checked as boolean)}
                  />
                  <label htmlFor="enable-profiling" className="text-sm font-medium cursor-pointer flex items-center">
                    <BarChart3 className="h-4 w-4 mr-2 text-blue-500" />
                    Enable Data Profiling
                  </label>
                </div>

                {/* Create Twin Checkbox */}
                <div className="flex items-center space-x-2 p-3 border rounded-lg bg-gray-50">
                  <Checkbox
                    id="create-twin"
                    checked={createTwin}
                    onCheckedChange={(checked) => setCreateTwin(checked as boolean)}
                  />
                  <label htmlFor="create-twin" className="text-sm font-medium cursor-pointer flex items-center">
                    <GitBranch className="h-4 w-4 mr-2 text-purple-500" />
                    Create Digital Twin
                  </label>
                </div>

                <Button
                  onClick={handleGenerateOntology}
                  disabled={generating || columns.filter(c => c.selected).length === 0}
                  className="w-full"
                >
                  {generating ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Generating...
                    </>
                  ) : (
                    <>
                      <Database className="h-4 w-4 mr-2" />
                      Generate Ontology {createTwin && "+ Twin"}
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
                          const isPrimaryKey = profile?.suggested_primary_keys.includes(col);
                          return (
                            <th
                              key={col}
                              className={`border border-gray-300 px-4 py-2 text-left text-sm font-medium ${
                                isSelected ? 'bg-green-50 text-green-800' : 'text-gray-500'
                              } ${isPrimaryKey ? 'border-2 border-blue-400' : ''}`}
                            >
                              <div className="flex items-center">
                                {getDataTypeIcon(column?.dataType || 'unknown')}
                                <span className="ml-2">{col}</span>
                                {isSelected && <CheckCircle className="h-4 w-4 ml-2 text-green-500" />}
                                {isPrimaryKey && <Database className="h-4 w-4 ml-2 text-blue-500" />}
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

          {/* Column Profiling Details */}
          {profile && profile.column_profiles.length > 0 && (
            <Card className="mt-6">
              <CardHeader>
                <CardTitle className="flex items-center">
                  <BarChart3 className="h-5 w-5 mr-2" />
                  Column Profiling Details
                </CardTitle>
                <CardDescription>
                  Detailed statistics and quality metrics for each column
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {profile.column_profiles.map((colProfile) => {
                    const isExpanded = expandedProfiles.has(colProfile.column_name);
                    return (
                      <div key={colProfile.column_name} className="border rounded-lg">
                        <div
                          className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
                          onClick={() => toggleProfileExpansion(colProfile.column_name)}
                        >
                          <div className="flex items-center space-x-3 flex-1">
                            <div>
                              <div className="font-medium">{colProfile.column_name}</div>
                              <div className="text-sm text-gray-500 capitalize">{colProfile.data_type}</div>
                            </div>
                          </div>
                          <div className="flex items-center space-x-3">
                            {getQualityBadge(colProfile.data_quality_score)}
                            {isExpanded ? (
                              <ChevronUp className="h-5 w-5 text-gray-500" />
                            ) : (
                              <ChevronDown className="h-5 w-5 text-gray-500" />
                            )}
                          </div>
                        </div>

                        {isExpanded && (
                          <div className="px-4 pb-4 border-t">
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4">
                              <div>
                                <div className="text-xs text-gray-500">Distinct</div>
                                <div className="font-semibold">{colProfile.distinct_percent.toFixed(1)}%</div>
                                <div className="text-xs text-gray-400">{colProfile.distinct_count.toLocaleString()} values</div>
                              </div>
                              <div>
                                <div className="text-xs text-gray-500">Nulls</div>
                                <div className={`font-semibold ${colProfile.null_percent > 20 ? 'text-red-600' : 'text-green-600'}`}>
                                  {colProfile.null_percent.toFixed(1)}%
                                </div>
                                <div className="text-xs text-gray-400">{colProfile.null_count.toLocaleString()} nulls</div>
                              </div>
                              {colProfile.mean !== undefined && (
                                <div>
                                  <div className="text-xs text-gray-500">Mean</div>
                                  <div className="font-semibold">{colProfile.mean.toFixed(2)}</div>
                                </div>
                              )}
                              {colProfile.avg_length !== undefined && (
                                <div>
                                  <div className="text-xs text-gray-500">Avg Length</div>
                                  <div className="font-semibold">{colProfile.avg_length.toFixed(1)}</div>
                                </div>
                              )}
                            </div>

                            {/* Top Values */}
                            {colProfile.top_values && colProfile.top_values.length > 0 && (
                              <div className="mt-4">
                                <div className="text-sm font-semibold mb-2">Top Values</div>
                                <div className="space-y-1">
                                  {colProfile.top_values.slice(0, 5).map((vf, idx) => (
                                    <div key={idx} className="flex items-center justify-between text-sm">
                                      <span className="truncate max-w-[200px]">{String(vf.value)}</span>
                                      <div className="flex items-center space-x-2">
                                        <div className="w-24 bg-gray-200 rounded-full h-2">
                                          <div
                                            className="bg-blue-500 h-2 rounded-full"
                                            style={{ width: `${vf.frequency * 100}%` }}
                                          />
                                        </div>
                                        <span className="text-gray-500 text-xs w-12 text-right">
                                          {(vf.frequency * 100).toFixed(1)}%
                                        </span>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            )}

                            {/* Quality Issues */}
                            {colProfile.quality_issues && colProfile.quality_issues.length > 0 && (
                              <div className="mt-4">
                                <div className="text-sm font-semibold mb-2 text-red-600">Quality Issues</div>
                                <ul className="space-y-1">
                                  {colProfile.quality_issues.map((issue, idx) => (
                                    <li key={idx} className="text-sm text-red-600 flex items-start">
                                      <AlertCircle className="h-4 w-4 mr-2 mt-0.5 flex-shrink-0" />
                                      {issue}
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Data Statistics */}
          {hasData && !profile && (
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

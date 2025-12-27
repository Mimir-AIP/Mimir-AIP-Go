"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { 
  Brain, 
  Upload, 
  CheckCircle, 
  XCircle, 
  AlertTriangle,
  RefreshCw,
  Save,
  Eye
} from "lucide-react";

interface InferredColumn {
  name: string;
  inferred_type: string;
  confidence: number;
  sample_values: string[];
  null_count: number;
  unique_count: number;
  suggested_mapping?: string;
}

interface TypeInferenceResult {
  columns: InferredColumn[];
  summary: {
    total_columns: number;
    numeric_columns: number;
    categorical_columns: number;
    datetime_columns: number;
    text_columns: number;
    confidence_score: number;
  };
}

interface TypeInferenceProps {
  ontologyId: string;
}

export function TypeInference({ ontologyId }: TypeInferenceProps) {
  const [columns, setColumns] = useState<InferredColumn[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string>("");
  const [analyzing, setAnalyzing] = useState(false);
  const [inferredTypes, setInferredTypes] = useState<TypeInferenceResult | null>(null);

  const dataTypes = [
    "string", "integer", "float", "boolean", 
    "date", "datetime", "uri", "email", 
    "phone", "address", "coordinate"
  ];

  useEffect(() => {
    loadSavedInferences();
  }, [ontologyId]);

  async function loadSavedInferences() {
    // Load any previously saved type inferences for this ontology
    try {
      const response = await fetch(`/api/v1/ontology/${ontologyId}/inferred-types`);
      if (response.ok) {
        const data = await response.json();
        if (data.data) {
          setInferredTypes(data.data);
        }
      }
    } catch (err) {
      console.error("Failed to load saved inferences:", err);
    }
  }

  function handleFileUpload(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    // Check file size (10MB limit)
    if (file.size > 10 * 1024 * 1024) {
      toast.error("File size exceeds 10MB limit");
      return;
    }

    // Check file type
    const allowedTypes = ["text/csv", "application/json", "text/plain"];
    if (!allowedTypes.includes(file.type) && !file.name.match(/\.(csv|json|txt)$/i)) {
      toast.error("Only CSV, JSON, and TXT files are supported");
      return;
    }

    setUploadedFile(file);
    setError(null);
    setInferredTypes(null);

    // Generate preview
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      setPreview(content.substring(0, 1000) + (content.length > 1000 ? "..." : ""));
    };
    reader.readAsText(file);
  }

  async function analyzeTypes() {
    if (!uploadedFile) {
      toast.error("Please upload a file first");
      return;
    }

    setAnalyzing(true);
    setError(null);

    try {
      const formData = new FormData();
      formData.append("file", uploadedFile);
      formData.append("ontology_id", ontologyId);

      const response = await fetch(`/api/v1/ontology/${ontologyId}/infer-types`, {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      if (data.success) {
        setInferredTypes(data.data);
        setColumns(data.data.columns || []);
        toast.success(`Successfully analyzed ${data.data.summary.total_columns} columns`);
      } else {
        throw new Error(data.error || "Failed to analyze types");
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to analyze types";
      setError(message);
      toast.error(message);
    } finally {
      setAnalyzing(false);
    }
  }

  async function saveInferences() {
    if (!inferredTypes) {
      toast.error("No type inferences to save");
      return;
    }

    try {
      const response = await fetch(`/api/v1/ontology/${ontologyId}/inferred-types`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: inferredTypes }),
      });

      if (!response.ok) {
        throw new Error("Failed to save inferences");
      }

      toast.success("Type inferences saved successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save inferences";
      toast.error(message);
    }
  }

  function updateColumnType(columnName: string, newType: string) {
    setColumns(prev => prev.map(col => 
      col.name === columnName 
        ? { ...col, suggested_mapping: newType }
        : col
    ));
  }

  function getConfidenceColor(confidence: number) {
    if (confidence >= 0.8) return "text-green-600";
    if (confidence >= 0.6) return "text-yellow-600";
    return "text-red-600";
  }

  function getTypeIcon(type: string) {
    switch (type.toLowerCase()) {
      case "integer":
      case "float":
        return "🔢";
      case "string":
        return "📝";
      case "boolean":
        return "✅";
      case "date":
      case "datetime":
        return "📅";
      case "uri":
        return "🔗";
      default:
        return "❓";
    }
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="animate-pulse">
          <div className="h-4 bg-gray-200 rounded w-1/4 mb-2"></div>
          <div className="h-8 bg-gray-200 rounded mb-4"></div>
          <div className="space-y-2">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="h-12 bg-gray-200 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error && !inferredTypes) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3 text-red-600">
            <XCircle className="h-6 w-6" />
            <div>
              <h4 className="font-medium">Analysis Failed</h4>
              <p className="text-sm mt-1">{error}</p>
            </div>
          </div>
          <Button onClick={() => setError(null)} className="mt-4">
            Try Again
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h3 className="text-lg font-semibold mb-2 flex items-center gap-2">
          <Brain className="h-5 w-5 text-blue-600" />
          AI-Powered Type Inference
        </h3>
        <p className="text-sm text-muted-foreground">
          Upload a data file and let our AI analyze the column types with confidence scoring
        </p>
      </div>

      {/* File Upload */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Upload className="h-4 w-4" />
            Upload Data File
          </CardTitle>
          <CardDescription>
            Upload CSV, JSON, or TXT files for automatic type detection (Max 10MB)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <Label htmlFor="file-upload">Select File</Label>
              <Input
                id="file-upload"
                type="file"
                accept=".csv,.json,.txt"
                onChange={handleFileUpload}
                className="mt-1"
              />
            </div>

            {uploadedFile && (
              <div className="space-y-2">
                <div className="flex items-center justify-between p-3 bg-muted rounded-md">
                  <span className="text-sm font-medium">{uploadedFile.name}</span>
                  <Badge variant="outline">
                    {(uploadedFile.size / 1024 / 1024).toFixed(2)} MB
                  </Badge>
                </div>

                {preview && (
                  <div className="text-xs">
                    <Label>Preview:</Label>
                    <pre className="mt-1 p-2 bg-gray-50 rounded border overflow-auto max-h-20">
                      {preview}
                    </pre>
                  </div>
                )}

                <Button
                  onClick={analyzeTypes}
                  disabled={analyzing}
                  className="w-full"
                >
                  {analyzing ? (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                      Analyzing...
                    </>
                  ) : (
                    <>
                      <Brain className="h-4 w-4 mr-2" />
                      Analyze Types
                    </>
                  )}
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Analysis Results */}
      {inferredTypes && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span className="flex items-center gap-2">
                <Eye className="h-4 w-4" />
                Inferred Types
              </span>
              <Button onClick={saveInferences} size="sm" variant="outline">
                <Save className="h-3 w-3 mr-1" />
                Save
              </Button>
            </CardTitle>
            <CardDescription>
              Review and edit the AI-detected column types before saving
            </CardDescription>
          </CardHeader>
          <CardContent>
            {/* Summary */}
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
              <div className="text-center p-3 bg-blue-50 rounded-lg">
                <div className="text-2xl font-bold text-blue-600">
                  {inferredTypes.summary.total_columns}
                </div>
                <div className="text-xs text-blue-600">Total Columns</div>
              </div>
              <div className="text-center p-3 bg-green-50 rounded-lg">
                <div className="text-2xl font-bold text-green-600">
                  {inferredTypes.summary.numeric_columns}
                </div>
                <div className="text-xs text-green-600">Numeric</div>
              </div>
              <div className="text-center p-3 bg-purple-50 rounded-lg">
                <div className="text-2xl font-bold text-purple-600">
                  {inferredTypes.summary.categorical_columns}
                </div>
                <div className="text-xs text-purple-600">Categorical</div>
              </div>
              <div className="text-center p-3 bg-orange-50 rounded-lg">
                <div className="text-2xl font-bold text-orange-600">
                  {inferredTypes.summary.datetime_columns}
                </div>
                <div className="text-xs text-orange-600">DateTime</div>
              </div>
              <div className="text-center p-3 bg-gray-50 rounded-lg">
                <div className="text-2xl font-bold text-gray-600">
                  {inferredTypes.summary.text_columns}
                </div>
                <div className="text-xs text-gray-600">Text</div>
              </div>
            </div>

            {/* Column Details */}
            <div className="space-y-3">
              {columns.map((column, index) => (
                <div key={index} className="border rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-3">
                      <span className="font-mono text-sm font-medium">
                        {column.name}
                      </span>
                      <span className="text-lg">
                        {getTypeIcon(column.inferred_type)}
                      </span>
                      <Badge variant="outline">
                        {column.inferred_type}
                      </Badge>
                      <Badge 
                        variant={column.confidence >= 0.8 ? "default" : 
                                column.confidence >= 0.6 ? "secondary" : "destructive"}
                      >
                        {(column.confidence * 100).toFixed(0)}% confidence
                      </Badge>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">Unique Values:</span>
                      <span className="ml-2 font-medium">{column.unique_count}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Null Values:</span>
                      <span className="ml-2 font-medium">{column.null_count}</span>
                    </div>
                    <div className="md:col-span-2">
                      <span className="text-muted-foreground">Sample Values:</span>
                      <span className="ml-2 text-xs">
                        {column.sample_values.slice(0, 3).join(", ")}
                        {column.sample_values.length > 3 && "..."}
                      </span>
                    </div>
                  </div>

                  {/* Type Override */}
                  <div className="mt-3 flex items-center gap-2">
                    <Label htmlFor={`type-select-${index}`} className="text-sm">
                      Override Type:
                    </Label>
                    <Select 
                      value={column.suggested_mapping || column.inferred_type}
                      onValueChange={(value) => updateColumnType(column.name, value)}
                    >
                      <SelectTrigger id={`type-select-${index}`} className="text-sm">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {dataTypes.map((type) => (
                          <SelectItem key={type} value={type}>
                            <span className="flex items-center gap-2">
                              <span>{getTypeIcon(type)}</span>
                              {type}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-4 pt-4 border-t flex justify-end">
              <Button onClick={saveInferences} disabled={analyzing}>
                <Save className="h-4 w-4 mr-2" />
                Save Type Mappings
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
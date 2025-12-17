"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";
import {
  Upload,
  FileText,
  Table,
  FileSpreadsheet,
  Settings,
  CheckCircle,
  AlertCircle,
  Loader2,
} from "lucide-react";

// Extended plugin info from backend
interface ExtendedPluginInfo {
  type: string;
  name: string;
  description: string;
  config_schema: any;
  supported_formats: string[];
}

// Upload state
interface UploadState {
  pluginType: string;
  pluginName: string;
  file: File | null;
  config: Record<string, any>;
  uploading: boolean;
  progress: number;
}

export default function DataUploadPage() {
  const router = useRouter();
  const [plugins, setPlugins] = useState<ExtendedPluginInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPlugin, setSelectedPlugin] = useState<ExtendedPluginInfo | null>(null);
  const [uploadState, setUploadState] = useState<UploadState>({
    pluginType: "",
    pluginName: "",
    file: null,
    config: {},
    uploading: false,
    progress: 0,
  });

  useEffect(() => {
    loadPlugins();
  }, []);

  const loadPlugins = async () => {
    try {
      const response = await fetch("/api/v1/data/plugins");
      if (!response.ok) {
        throw new Error("Failed to load plugins");
      }
      const data = await response.json();
      setPlugins(data.plugins || []);
    } catch (error) {
      console.error("Failed to load plugins:", error);
      toast.error("Failed to load available plugins");
    } finally {
      setLoading(false);
    }
  };

  const getPluginIcon = (pluginName: string) => {
    switch (pluginName) {
      case "csv":
        return <Table className="h-8 w-8 text-green-500" />;
      case "markdown":
        return <FileText className="h-8 w-8 text-blue-500" />;
      case "excel":
        return <FileSpreadsheet className="h-8 w-8 text-orange-500" />;
      default:
        return <Upload className="h-8 w-8 text-gray-500" />;
    }
  };

  const selectPlugin = (plugin: ExtendedPluginInfo) => {
    setSelectedPlugin(plugin);
    setUploadState({
      pluginType: plugin.type,
      pluginName: plugin.name,
      file: null,
      config: {},
      uploading: false,
      progress: 0,
    });
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file && selectedPlugin) {
      // Validate file type
      const fileExt = file.name.split(".").pop()?.toLowerCase();
      if (!selectedPlugin.supported_formats.includes(fileExt || "")) {
        toast.error(`Unsupported file type. Supported: ${selectedPlugin.supported_formats.join(", ")}`);
        return;
      }

      setUploadState(prev => ({
        ...prev,
        file,
      }));
    }
  };

  const handleConfigChange = (key: string, value: any) => {
    setUploadState(prev => ({
      ...prev,
      config: {
        ...prev.config,
        [key]: value,
      },
    }));
  };

  const handleUpload = async () => {
    if (!uploadState.file || !selectedPlugin) {
      toast.error("Please select a file first");
      return;
    }

    setUploadState(prev => ({ ...prev, uploading: true, progress: 0 }));

    try {
      const formData = new FormData();
      formData.append("file", uploadState.file);
      formData.append("plugin_type", uploadState.pluginType);
      formData.append("plugin_name", uploadState.pluginName);

      // Add config as JSON
      if (Object.keys(uploadState.config).length > 0) {
        formData.append("config", JSON.stringify(uploadState.config));
      }

      const response = await fetch("/api/v1/data/upload", {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || "Upload failed");
      }

      const result = await response.json();
      toast.success("File uploaded successfully!");

      // Redirect to preview page
      router.push(`/data/preview/${result.upload_id}`);

    } catch (error) {
      console.error("Upload failed:", error);
      toast.error(error instanceof Error ? error.message : "Upload failed");
    } finally {
      setUploadState(prev => ({ ...prev, uploading: false, progress: 0 }));
    }
  };

  const renderConfigForm = () => {
    if (!selectedPlugin || !selectedPlugin.config_schema?.properties) {
      return null;
    }

    const properties = selectedPlugin.config_schema.properties;
    const required = selectedPlugin.config_schema.required || [];

    return (
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Configuration</h3>
        {Object.entries(properties).map(([key, prop]: [string, any]) => {
          const isRequired = required.includes(key);
          const currentValue = uploadState.config[key] ?? prop.default;

          return (
            <div key={key} className="space-y-2">
              <label className="block text-sm font-medium">
                {key.replace(/_/g, " ").replace(/\b\w/g, l => l.toUpperCase())}
                {isRequired && <span className="text-red-500 ml-1">*</span>}
              </label>

              {prop.type === "boolean" ? (
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={currentValue || false}
                    onChange={(e) => handleConfigChange(key, e.target.checked)}
                    className="rounded border-gray-300"
                  />
                  <span className="text-sm text-muted-foreground">{prop.description}</span>
                </div>
              ) : prop.type === "string" ? (
                <div>
                  <input
                    type="text"
                    value={currentValue || ""}
                    onChange={(e) => handleConfigChange(key, e.target.value)}
                    placeholder={prop.description}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    required={isRequired}
                  />
                  {prop.enum && (
                    <select
                      value={currentValue || ""}
                      onChange={(e) => handleConfigChange(key, e.target.value)}
                      className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Select...</option>
                      {prop.enum.map((option: string) => (
                        <option key={option} value={option}>{option}</option>
                      ))}
                    </select>
                  )}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    );
  };

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          <span className="ml-2 text-lg">Loading plugins...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="mb-6">
        <Link href="/" className="text-orange hover:underline mb-4 inline-block">
          ← Back to Dashboard
        </Link>
        <h1 className="text-3xl font-bold text-orange">Data Ingestion</h1>
        <p className="text-gray-400 mt-1">
          Upload data files and automatically generate ontologies
        </p>
      </div>

      {!selectedPlugin ? (
        // Plugin Selection
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {plugins.map((plugin) => (
            <Card
              key={`${plugin.type}.${plugin.name}`}
              className="cursor-pointer hover:shadow-lg transition-shadow"
              onClick={() => selectPlugin(plugin)}
            >
              <CardHeader className="text-center">
                {getPluginIcon(plugin.name)}
                <CardTitle className="mt-4">{plugin.name.toUpperCase()}</CardTitle>
                <CardDescription>{plugin.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-1 mb-4">
                  {plugin.supported_formats.map((format) => (
                    <Badge key={format} variant="outline" className="text-xs">
                      .{format}
                    </Badge>
                  ))}
                </div>
                <Button className="w-full" variant="outline">
                  <Upload className="h-4 w-4 mr-2" />
                  Select Plugin
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        // Upload Interface
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  {getPluginIcon(selectedPlugin.name)}
                  <div>
                    <CardTitle>{selectedPlugin.name.toUpperCase()} Upload</CardTitle>
                    <CardDescription>{selectedPlugin.description}</CardDescription>
                  </div>
                </div>
                <Button
                  variant="outline"
                  onClick={() => setSelectedPlugin(null)}
                >
                  Change Plugin
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* File Upload */}
              <div className="space-y-4">
                <h3 className="text-lg font-semibold">File Upload</h3>
                <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
                  <input
                    type="file"
                    accept={selectedPlugin.supported_formats.map(f => `.${f}`).join(",")}
                    onChange={handleFileChange}
                    className="hidden"
                    id="file-upload"
                  />
                  <label htmlFor="file-upload" className="cursor-pointer">
                    <Upload className="h-12 w-12 mx-auto text-gray-400 mb-4" />
                    <p className="text-lg font-medium text-gray-900 mb-2">
                      Click to upload or drag and drop
                    </p>
                    <p className="text-sm text-gray-500">
                      Supported formats: {selectedPlugin.supported_formats.join(", ")}
                    </p>
                  </label>
                </div>

                {uploadState.file && (
                  <div className="flex items-center space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
                    <CheckCircle className="h-5 w-5 text-green-500" />
                    <div>
                      <p className="font-medium text-green-800">{uploadState.file.name}</p>
                      <p className="text-sm text-green-600">
                        {(uploadState.file.size / 1024).toFixed(1)} KB
                      </p>
                    </div>
                  </div>
                )}
              </div>

              {/* Configuration */}
              {renderConfigForm()}

              {/* Upload Progress */}
              {uploadState.uploading && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">Uploading...</span>
                    <span className="text-sm text-gray-500">{uploadState.progress}%</span>
                  </div>
                  <Progress value={uploadState.progress} className="w-full" />
                </div>
              )}

              {/* Actions */}
              <div className="flex justify-end space-x-3">
                <Button
                  variant="outline"
                  onClick={() => setSelectedPlugin(null)}
                  disabled={uploadState.uploading}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleUpload}
                  disabled={!uploadState.file || uploadState.uploading}
                >
                  {uploadState.uploading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Uploading...
                    </>
                  ) : (
                    <>
                      <Upload className="h-4 w-4 mr-2" />
                      Upload & Preview
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
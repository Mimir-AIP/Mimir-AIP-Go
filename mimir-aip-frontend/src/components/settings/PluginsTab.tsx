"use client";

import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  listPlugins,
  uploadPlugin,
  updatePlugin,
  deletePlugin,
  reloadPlugin,
  type Plugin,
} from "@/lib/api";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";

export default function PluginsTab() {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadPlugins();
  }, []);

  async function loadPlugins() {
    try {
      setLoading(true);
      const pluginList = await listPlugins();
      setPlugins(pluginList);
    } catch (error) {
      toast.error(`Failed to load plugins: ${error instanceof Error ? error.message : "Unknown error"}`);
    } finally {
      setLoading(false);
    }
  }

  async function handleUpload(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    // Validate file extension
    const validExtensions = [".so", ".dll"];
    const fileExt = file.name.substring(file.name.lastIndexOf("."));
    if (!validExtensions.includes(fileExt)) {
      toast.error("Invalid file type. Please upload .so (Linux) or .dll (Windows) files");
      return;
    }

    try {
      setUploading(true);
      await uploadPlugin(file);
      toast.success(`Plugin "${file.name}" uploaded successfully`);
      loadPlugins();
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    } catch (error) {
      toast.error(`Failed to upload plugin: ${error instanceof Error ? error.message : "Unknown error"}`);
    } finally {
      setUploading(false);
    }
  }

  async function handleToggle(id: string, name: string, currentEnabled: boolean) {
    try {
      await updatePlugin(id, { is_enabled: !currentEnabled });
      toast.success(`Plugin "${name}" ${!currentEnabled ? "enabled" : "disabled"}`);
      loadPlugins();
    } catch (error) {
      toast.error(`Failed to update plugin: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  async function handleReload(id: string, name: string) {
    try {
      await reloadPlugin(id);
      toast.success(`Plugin "${name}" reloaded`);
      loadPlugins();
    } catch (error) {
      toast.error(`Failed to reload plugin: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  async function handleDelete(id: string, name: string, isBuiltin: boolean) {
    if (isBuiltin) {
      toast.error("Cannot delete built-in plugins");
      return;
    }

    if (!confirm(`Are you sure you want to delete the plugin "${name}"?`)) {
      return;
    }

    try {
      await deletePlugin(id);
      toast.success("Plugin deleted");
      loadPlugins();
    } catch (error) {
      toast.error(`Failed to delete plugin: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  if (loading) {
    return <div className="text-center py-8">Loading plugins...</div>;
  }

  const builtinPlugins = plugins.filter((p) => p.is_builtin);
  const customPlugins = plugins.filter((p) => !p.is_builtin);

  return (
    <div className="space-y-6">
      {/* Header with Upload button */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-xl font-semibold">Plugins</h2>
          <p className="text-sm text-gray-400 mt-1">
            Manage built-in and custom plugins for data processing, AI models, and integrations
          </p>
        </div>
        <div>
          <input
            ref={fileInputRef}
            type="file"
            accept=".so,.dll"
            onChange={handleUpload}
            className="hidden"
          />
          <Button
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="bg-orange hover:bg-orange/80 text-navy"
          >
            {uploading ? "Uploading..." : "Upload Plugin"}
          </Button>
        </div>
      </div>

      {/* Built-in Plugins */}
      <div>
        <h3 className="text-lg font-semibold mb-3 text-orange">Built-in Plugins</h3>
        {builtinPlugins.length === 0 ? (
          <Card className="bg-blue border-blue">
            <CardContent className="py-8 text-center text-gray-400">
              No built-in plugins found
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {builtinPlugins.map((plugin) => (
              <Card key={plugin.id} className="bg-blue border-blue">
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-2">
                        <CardTitle className="text-orange">{plugin.name}</CardTitle>
                        <Badge variant="outline" className="text-xs border-orange text-orange">
                          {plugin.type}
                        </Badge>
                        <Badge variant="outline" className="text-xs border-blue text-gray-400">
                          v{plugin.version}
                        </Badge>
                        <Badge className="text-xs bg-green-600">Built-in</Badge>
                      </div>
                      <CardDescription className="text-gray-400">
                        {plugin.description || "No description available"}
                      </CardDescription>
                      {plugin.author && (
                        <p className="text-xs text-gray-500 mt-1">Author: {plugin.author}</p>
                      )}
                    </div>
                    <div className="flex items-center space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleReload(plugin.id, plugin.name)}
                        className="border-blue text-white hover:bg-blue"
                      >
                        Reload
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleToggle(plugin.id, plugin.name, plugin.is_enabled)}
                        className={`border-blue ${
                          plugin.is_enabled
                            ? "text-orange hover:bg-blue"
                            : "text-gray-400 hover:bg-blue"
                        }`}
                      >
                        {plugin.is_enabled ? "Enabled" : "Disabled"}
                      </Button>
                    </div>
                  </div>
                </CardHeader>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Custom Plugins */}
      <div>
        <h3 className="text-lg font-semibold mb-3 text-orange">Custom Plugins</h3>
        {customPlugins.length === 0 ? (
          <Card className="bg-blue border-blue">
            <CardContent className="py-8 text-center text-gray-400">
              No custom plugins installed. Upload one to get started.
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {customPlugins.map((plugin) => (
              <Card key={plugin.id} className="bg-blue border-blue">
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-2">
                        <CardTitle className="text-orange">{plugin.name}</CardTitle>
                        <Badge variant="outline" className="text-xs border-orange text-orange">
                          {plugin.type}
                        </Badge>
                        <Badge variant="outline" className="text-xs border-blue text-gray-400">
                          v{plugin.version}
                        </Badge>
                      </div>
                      <CardDescription className="text-gray-400">
                        {plugin.description || "No description available"}
                      </CardDescription>
                      {plugin.author && (
                        <p className="text-xs text-gray-500 mt-1">Author: {plugin.author}</p>
                      )}
                      <p className="text-xs text-gray-500 mt-1">Path: {plugin.file_path}</p>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleReload(plugin.id, plugin.name)}
                        className="border-blue text-white hover:bg-blue"
                      >
                        Reload
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleToggle(plugin.id, plugin.name, plugin.is_enabled)}
                        className={`border-blue ${
                          plugin.is_enabled
                            ? "text-orange hover:bg-blue"
                            : "text-gray-400 hover:bg-blue"
                        }`}
                      >
                        {plugin.is_enabled ? "Enabled" : "Disabled"}
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleDelete(plugin.id, plugin.name, plugin.is_builtin)}
                        className="bg-red-600 hover:bg-red-700"
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </CardHeader>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

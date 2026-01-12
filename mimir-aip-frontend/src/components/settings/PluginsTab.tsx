"use client";

import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  listPlugins,
  uploadPlugin,
  updatePlugin,
  deletePlugin,
  reloadPlugin,
  getPluginConfig,
  savePluginConfig,
  deletePluginConfig,
  type Plugin,
} from "@/lib/api";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Settings, X } from "lucide-react";
import { ModelSelect } from "@/components/settings/ModelSelect";

interface SchemaProperty {
  type: string;
  description?: string;
  default?: any;
  enum?: string[];
  minimum?: number;
  maximum?: number;
  format?: string;
  dynamic_model?: boolean;
  model_fetch_url?: string;
}

interface PluginSchema {
  type?: string;
  properties?: Record<string, SchemaProperty>;
  required?: string[];
}

function ConfigDialog({ 
  plugin, 
  isOpen, 
  onClose, 
  onSave 
}: { 
  plugin: Plugin | null; 
  isOpen: boolean; 
  onClose: () => void; 
  onSave: (config: Record<string, unknown>) => void;
}) {
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (isOpen && plugin) {
      loadConfig();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, plugin]);

  async function loadConfig() {
    if (!plugin) return;
    setLoading(true);
    try {
      const pluginConfig = await getPluginConfig(plugin.name);
      if (pluginConfig.config) {
        setConfig(pluginConfig.config);
      } else {
        // Initialize with defaults from schema
        const schema = plugin.input_schema as PluginSchema;
        const defaults: Record<string, unknown> = {};
        if (schema?.properties) {
          for (const [key, prop] of Object.entries(schema.properties)) {
            if (prop.default !== undefined) {
              defaults[key] = prop.default;
            }
          }
        }
        setConfig(defaults);
      }
    } catch (err) {
      console.error("Failed to load config:", err);
      // Initialize with defaults
      setConfig({});
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    if (!plugin) return;
    setSaving(true);
    try {
      await savePluginConfig(plugin.name, config);
      toast.success(`${plugin.name} configured successfully`);
      onSave(config);
      onClose();
    } catch {
      toast.error("Failed to save configuration");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!plugin) return;
    try {
      await deletePluginConfig(plugin.name);
      toast.success("Configuration deleted");
      setConfig({});
      onClose();
    } catch {
      toast.error("Failed to delete configuration");
    }
  }

  if (!isOpen || !plugin) return null;

  const schema = plugin.input_schema as PluginSchema;
  const properties = schema?.properties || {};

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <Card className="bg-blue border-blue w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Settings className="w-4 h-4" />
              Configure {plugin.name}
            </CardTitle>
            <CardDescription>{plugin.description}</CardDescription>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8">Loading configuration...</div>
          ) : (
            <div className="space-y-4">
              {Object.keys(properties).length === 0 ? (
                <p className="text-gray-400 text-center py-4">
                  This plugin has no configurable options.
                </p>
              ) : (
                Object.entries(properties).map(([key, prop]) => (
                  <div key={key} className="space-y-1">
                    <Label htmlFor={key}>
                      {key.replace(/_/g, " ").replace(/\b\w/g, l => l.toUpperCase())}
                      {schema.required?.includes(key) && <span className="text-orange ml-1">*</span>}
                    </Label>
                    
                    {/* Enum/Select field */}
                    {prop.enum ? (
                      <select
                        id={key}
                        value={String(config[key] ?? "")}
                        onChange={(e) => setConfig({ ...config, [key]: e.target.value })}
                        className="w-full bg-navy border-blue rounded px-3 py-2"
                      >
                        <option value="">Select...</option>
                        {prop.enum.map((option) => (
                          <option key={option} value={option}>{option}</option>
                        ))}
                      </select>
                    )
                    /* Dynamic model field (fetches from API) */ :
                    prop.dynamic_model && prop.model_fetch_url ? (
                      <ModelSelect
                        value={String(config[key] ?? prop.default ?? "")}
                        fetchUrl={prop.model_fetch_url}
                        onChange={(value) => setConfig({ ...config, [key]: value })}
                        placeholder="Select or search model..."
                        className="bg-navy border-blue"
                      />
                    )
                    /* Number field */ :
                    prop.type === "number" || prop.type === "integer" ? (
                      <Input
                        id={key}
                        type="number"
                        value={String(config[key] ?? prop.default ?? "")}
                        onChange={(e) => setConfig({ ...config, [key]: parseFloat(e.target.value) })}
                        min={prop.minimum}
                        max={prop.maximum}
                        className="bg-navy border-blue"
                      />
                    )
                    /* Boolean field */ :
                    prop.type === "boolean" ? (
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={key}
                          checked={Boolean(config[key] ?? prop.default ?? false)}
                          onChange={(e) => setConfig({ ...config, [key]: e.target.checked })}
                          className="w-4 h-4"
                        />
                        <Label htmlFor={key} className="text-sm text-gray-400">
                          {prop.description}
                        </Label>
                      </div>
                    )
                    /* Password field */ :
                    prop.format === "password" ? (
                      <Input
                        id={key}
                        type="password"
                        value={String(config[key] ?? "")}
                        onChange={(e) => setConfig({ ...config, [key]: e.target.value })}
                        placeholder="Enter value"
                        className="bg-navy border-blue"
                      />
                    )
                    /* Default text field */ :
                    (
                      <Input
                        id={key}
                        type="text"
                        value={String(config[key] ?? prop.default ?? "")}
                        onChange={(e) => setConfig({ ...config, [key]: e.target.value })}
                        placeholder={prop.description}
                        className="bg-navy border-blue"
                      />
                    )}
                  </div>
                ))
              )}

              <div className="flex justify-between pt-4">
                <Button
                  variant="outline"
                  onClick={handleDelete}
                  className="border-red-600 text-red-400 hover:bg-red-900/20"
                >
                  Delete Config
                </Button>
                <div className="flex gap-2">
                  <Button variant="outline" onClick={onClose}>
                    Cancel
                  </Button>
                  <Button
                    onClick={handleSave}
                    disabled={saving}
                    className="bg-orange hover:bg-orange/80 text-navy"
                  >
                    {saving ? "Saving..." : "Save"}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function PluginsTab() {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [configPlugin, setConfigPlugin] = useState<Plugin | null>(null);
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

  function openConfig(plugin: Plugin) {
    setConfigPlugin(plugin);
  }

  function closeConfig() {
    setConfigPlugin(null);
  }

  function handleConfigSaved() {
    loadPlugins();
  }

  if (loading) {
    return <div className="text-center py-8">Loading plugins...</div>;
  }

  const builtinPlugins = plugins.filter((p) => p.is_builtin);
  const customPlugins = plugins.filter((p) => !p.is_builtin);
  const configurablePlugins = plugins.filter(p => p.input_schema && Object.keys(p.input_schema).length > 0);

  return (
    <div className="space-y-6">
      <ConfigDialog 
        plugin={configPlugin} 
        isOpen={configPlugin !== null} 
        onClose={closeConfig}
        onSave={handleConfigSaved}
      />

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

      {/* Configurable Plugins */}
      {configurablePlugins.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-3 text-orange">Configurable Plugins</h3>
          <div className="grid gap-4">
            {configurablePlugins.map((plugin) => (
              <Card key={plugin.id} className="bg-blue border-blue">
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-2">
                        <CardTitle className="text-orange">{plugin.name}</CardTitle>
                        <Badge variant="outline" className="text-xs border-orange text-orange">
                          {plugin.type}
                        </Badge>
                        <Badge className="text-xs bg-blue-600">Configurable</Badge>
                      </div>
                      <CardDescription className="text-gray-400">
                        {plugin.description || "No description available"}
                      </CardDescription>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => openConfig(plugin)}
                      className="border-orange text-orange hover:bg-orange/10"
                    >
                      <Settings className="w-4 h-4 mr-1" />
                      Configure
                    </Button>
                  </div>
                </CardHeader>
              </Card>
            ))}
          </div>
        </div>
      )}

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

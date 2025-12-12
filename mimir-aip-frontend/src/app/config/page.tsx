"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { getConfig, updateConfig, reloadConfig, saveConfig, type Config } from "@/lib/api";
import { DetailsSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";

export default function ConfigPage() {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    fetchConfig();
  }, []);

  async function fetchConfig() {
    try {
      setLoading(true);
      setError(null);
      const data = await getConfig();
      setConfig(data);
      setEditValue(JSON.stringify(data, null, 2));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
      toast.error("Failed to load configuration");
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    try {
      setIsSaving(true);
      setError(null);
      const parsed = JSON.parse(editValue);
      await updateConfig(parsed);
      setConfig(parsed);
      toast.success("Configuration updated successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error(`Failed to update config: ${message}`);
    } finally {
      setIsSaving(false);
    }
  }

  async function handleReload() {
    try {
      setIsSaving(true);
      setError(null);
      await reloadConfig();
      toast.success("Configuration reloaded from file");
      // Fetch updated config
      await fetchConfig();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error(`Failed to reload config: ${message}`);
    } finally {
      setIsSaving(false);
    }
  }

  async function handleSaveToFile() {
    try {
      setIsSaving(true);
      setError(null);
      const result = await saveConfig("config.yaml", "yaml");
      toast.success(`Configuration saved to ${result.file}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error(`Failed to save config: ${message}`);
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-orange">Configuration</h1>
        <div className="flex gap-2">
          <Button onClick={handleReload} disabled={isSaving} variant="outline">
            Reload from File
          </Button>
          <Button onClick={handleSaveToFile} disabled={isSaving} variant="outline">
            Save to File
          </Button>
        </div>
      </div>

      {loading && <DetailsSkeleton />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchConfig} />}
      
      {config && !loading && (
        <Card className="bg-navy text-white border-blue p-6">
          <h2 className="text-xl font-bold text-orange mb-4">Edit Configuration</h2>
          <textarea
            className="w-full h-96 bg-blue/10 text-white p-4 rounded mb-4 border border-blue font-mono text-sm"
            value={editValue}
            onChange={e => setEditValue(e.target.value)}
            spellCheck={false}
          />
          <div className="flex gap-2">
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Saving..." : "Update Config"}
            </Button>
            <Button 
              onClick={() => setEditValue(JSON.stringify(config, null, 2))} 
              variant="outline"
              disabled={isSaving}
            >
              Reset
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}

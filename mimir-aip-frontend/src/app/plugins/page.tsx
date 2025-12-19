"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { getPlugins, type LegacyPlugin } from "@/lib/api";
import Link from "next/link";

export default function PluginsPage() {
  const [plugins, setPlugins] = useState<LegacyPlugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getPlugins();
        console.log("Plugins loaded:", data); // Debug log
        setPlugins(Array.isArray(data) ? data : []);
      } catch (err) {
        console.error("Plugin load error:", err); // Debug log
        setError(err instanceof Error ? err.message : "Unknown error");
        setPlugins([]);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-orange mb-2">Plugins</h1>
          <p className="text-gray-400 text-sm">
            View installed plugins and their details
          </p>
        </div>
        <Link href="/settings">
          <Button className="bg-orange hover:bg-orange/80 text-navy">
            Manage Plugins
          </Button>
        </Link>
      </div>

      {loading && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {[...Array(6)].map((_, i) => (
            <Card key={i} className="bg-navy text-white border-blue p-4 animate-pulse">
              <div className="h-6 bg-blue/50 rounded mb-2"></div>
              <div className="h-4 bg-blue/30 rounded mb-2"></div>
              <div className="h-4 bg-blue/30 rounded"></div>
            </Card>
          ))}
        </div>
      )}
      {error && (
        <div className="bg-red-900/20 border border-red-500 text-red-400 px-4 py-3 rounded mb-4">
          Error: {error}
        </div>
      )}
      
      {!loading && plugins.length === 0 && (
        <Card className="bg-blue border-blue p-8 text-center">
          <p className="text-gray-400 mb-4">No plugins found</p>
          <Link href="/settings">
            <Button className="bg-orange hover:bg-orange/80 text-navy">
              Go to Settings to Add Plugins
            </Button>
          </Link>
        </Card>
      )}

      {!loading && plugins.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {plugins.map((plugin) => (
            <Card key={plugin.name} className="bg-navy text-white border-blue p-4 plugin-card" data-testid="plugin-card">
              <h2 className="text-xl font-bold text-orange mb-2">{plugin.name}</h2>
              <p className="mb-2 text-gray-300">
                <span className="text-gray-400">Type:</span> {plugin.type || "Unknown"}
              </p>
              <p className="mb-2 text-gray-300">
                <span className="text-gray-400">Version:</span> {plugin.version || "N/A"}
              </p>
              <p className="text-gray-300">
                {plugin.description || "No description available"}
              </p>
              {plugin.author && (
                <p className="mt-2 text-xs text-gray-500">
                  By: {plugin.author}
                </p>
              )}
              <div className="mt-4 flex gap-2">
                <Button size="sm" variant="outline" className="border-blue text-white hover:bg-blue">
                  Configure
                </Button>
                <Button size="sm" variant="outline" className="border-blue text-white hover:bg-blue">
                  Enable
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

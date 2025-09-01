"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getPlugins } from "@/lib/api";

export default function PluginsPage() {
  const [plugins, setPlugins] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getPlugins();
        setPlugins(data);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">Plugins</h1>
      {loading && <p>Loading...</p>}
      {error && <p className="text-red-500">Error: {error}</p>}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {plugins.map((plugin) => (
          <Card key={plugin.name} className="bg-navy text-white border-blue">
            <h2 className="text-xl font-bold text-orange mb-2">{plugin.name}</h2>
            <p className="mb-2">Type: {plugin.type || "Unknown"}</p>
            <p className="mb-2">Description: {plugin.description || "No description"}</p>
            {/* TODO: Add View Details, Filter by Type actions */}
          </Card>
        ))}
      </div>
    </div>
  );
}

"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getConfig, updateConfig } from "@/lib/api";

export default function ConfigPage() {
  const [config, setConfig] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<string>("");

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getConfig();
        setConfig(data);
        setEditValue(JSON.stringify(data, null, 2));
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  async function handleSave() {
    try {
      setSuccess(null);
      setError(null);
      const parsed = JSON.parse(editValue);
      await updateConfig(parsed);
      setSuccess("Config updated successfully.");
      setConfig(parsed);
    } catch (err: any) {
      setError("Error: " + err.message);
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">Config</h1>
      {loading && <p>Loading...</p>}
      {error && <p className="text-red-500">{error}</p>}
      {success && <p className="text-green-500">{success}</p>}
      {config && (
        <Card className="bg-navy text-white border-blue">
          <h2 className="text-xl font-bold text-orange mb-2">Current Config</h2>
          <textarea
            className="w-full h-64 bg-blue/10 text-white p-4 rounded mb-4 border border-blue"
            value={editValue}
            onChange={e => setEditValue(e.target.value)}
          />
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-orange text-navy rounded hover:bg-blue hover:text-white font-bold"
          >
            Save Config
          </button>
        </Card>
      )}
    </div>
  );
}

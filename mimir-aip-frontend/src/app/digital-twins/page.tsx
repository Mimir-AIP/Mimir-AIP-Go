"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { listDigitalTwins, listOntologies, type DigitalTwin, type Ontology } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Network, Calendar, Database, Activity } from "lucide-react";

export default function DigitalTwinsPage() {
  const [twins, setTwins] = useState<DigitalTwin[]>([]);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      setLoading(true);
      setError(null);
      const [twinsData, ontologiesData] = await Promise.all([
        listDigitalTwins(),
        listOntologies(),
      ]);
      setTwins(twinsData);
      setOntologies(ontologiesData);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load digital twins";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function getOntologyName(ontologyId: string): string {
    const ontology = ontologies.find((o) => o.id === ontologyId);
    return ontology ? ontology.name : ontologyId;
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Digital Twins</h1>
          <p className="text-white/60">Loading digital twins...</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <Card key={i} className="bg-navy border-blue p-6 animate-pulse h-48"></Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Digital Twins</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>Error: {error}</p>
          <button 
            onClick={loadData}
            className="mt-4 bg-blue hover:bg-orange text-white px-4 py-2 rounded"
          >
            Retry
          </button>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold text-orange">Digital Twins</h1>
          <p className="text-white/60 mt-2">
            Monitor simulation status. Manage via chat.
          </p>
        </div>
        <button
          onClick={loadData}
          className="bg-blue hover:bg-orange text-white px-4 py-2 rounded border border-blue"
        >
          Refresh
        </button>
      </div>

      {twins.length === 0 ? (
        <Card className="bg-navy border-blue p-12 text-center">
          <Network className="h-16 w-16 mx-auto text-white/40 mb-4" />
          <h3 className="text-xl font-semibold text-white mb-2">No Digital Twins Yet</h3>
          <p className="text-white/60 mb-4">
            Digital twins are automatically created when you set up ontologies and pipelines
          </p>
          <p className="text-sm text-white/40">
            Create a pipeline to get started with automatic twin generation
          </p>
        </Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {twins.map((twin) => (
            <Card key={twin.id} className="bg-navy border-blue h-full">
              <div className="p-6">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-2">
                    <Network className="h-5 w-5 text-orange" />
                    <h3 className="text-lg font-semibold text-white">{twin.name}</h3>
                  </div>
                  <Badge variant="secondary">{twin.model_type}</Badge>
                </div>
                
                {twin.description && (
                  <p className="text-sm text-white/60 mb-4 line-clamp-2">
                    {twin.description}
                  </p>
                )}
                
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <Database className="h-4 w-4 text-white/40" />
                    <span className="text-white/40">Ontology:</span>
                    <span className="text-white truncate">
                      {getOntologyName(twin.ontology_id)}
                    </span>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4 pt-3 border-t border-blue/30">
                    <div className="flex items-center gap-2">
                      <Activity className="h-4 w-4 text-green-400" />
                      <div>
                        <p className="text-xl font-bold text-white">{twin.entity_count || 0}</p>
                        <p className="text-xs text-white/40">Entities</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Activity className="h-4 w-4 text-blue-400" />
                      <div>
                        <p className="text-xl font-bold text-white">{twin.relationship_count || 0}</p>
                        <p className="text-xs text-white/40">Relationships</p>
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2 text-xs text-white/40 pt-2 border-t border-blue/30">
                    <Calendar className="h-3 w-3" />
                    Created {formatDate(twin.created_at)}
                  </div>
                </div>
                
                <div className="mt-4 pt-4 border-t border-blue/30">
                  <span className="text-xs text-white/40">Auto-generated from pipeline and ontology data</span>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
      
      {/* Summary */}
      {twins.length > 0 && (
        <div className="mt-6 p-4 bg-blue/20 rounded-lg">
          <p className="text-sm text-white/60">
            Total: {twins.length} digital twin{twins.length === 1 ? "" : "s"} | 
            Auto-generated from pipeline and ontology data
          </p>
        </div>
      )}
    </div>
  );
}

"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  listOntologies,
  type Ontology,
} from "@/lib/api";
import { Card } from "@/components/ui/card";

export default function OntologiesPage() {
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");

  useEffect(() => {
    loadOntologies();
  }, [statusFilter]);

  const loadOntologies = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await listOntologies(statusFilter);
      setOntologies(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontologies");
      setOntologies([]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Ontologies</h1>
          <p className="text-white/60">Loading ontologies...</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <Card key={i} className="bg-navy border-blue p-6 animate-pulse h-40"></Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Ontologies</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>Error loading ontologies: {error}</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-orange">Ontologies</h1>
          <p className="text-white/60 mt-1">
            Monitor auto-generated knowledge schemas. Manage via chat.
          </p>
        </div>
        <div className="flex gap-3">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange"
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="deprecated">Deprecated</option>
            <option value="draft">Draft</option>
          </select>
          <button
            onClick={loadOntologies}
            className="bg-blue hover:bg-orange text-white px-4 py-2 rounded border border-blue"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Ontologies Grid */}
      {ontologies.length === 0 ? (
        <Card className="bg-navy border-blue p-12 text-center">
          <p className="text-white/60 mb-2">No ontologies found</p>
          <p className="text-sm text-white/40">
            Ontologies are automatically created when you create a pipeline
          </p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {ontologies.map((ontology) => (
            <Card key={ontology.id} className="bg-navy border-blue p-6 h-full">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-bold text-orange">{ontology.name}</h2>
                <span
                  className={`px-2 py-1 text-xs font-semibold rounded-full ${
                    ontology.status === "active"
                      ? "bg-green-900/40 text-green-400 border border-green-500"
                      : ontology.status === "deprecated"
                      ? "bg-yellow-900/40 text-yellow-400 border border-yellow-500"
                      : "bg-blue-900/40 text-blue-400 border border-blue-500"
                  }`}
                >
                  {ontology.status}
                </span>
              </div>
              
              {ontology.description && (
                <p className="text-sm text-white/60 mb-4 line-clamp-2">
                  {ontology.description}
                </p>
              )}
              
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-white/40">Version</span>
                  <span className="text-white">{ontology.version}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-white/40">Format</span>
                  <span className="text-white">{ontology.format}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-white/40">Created</span>
                  <span className="text-white">
                    {new Date(ontology.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
              
              <div className="mt-4 pt-4 border-t border-blue/30">
                <span className="text-xs text-white/40">Auto-generated from pipeline data</span>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Summary */}
      <div className="mt-6 p-4 bg-blue/20 rounded-lg">
        <p className="text-sm text-white/60">
          Total: {ontologies.length} ontolog{ontologies.length === 1 ? "y" : "ies"} | 
          Active: {ontologies.filter(o => o.status === 'active').length} | 
          Auto-generated from pipeline data
        </p>
      </div>
    </div>
  );
}

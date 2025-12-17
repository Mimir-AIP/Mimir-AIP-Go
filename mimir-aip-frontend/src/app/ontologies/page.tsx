"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  listOntologies,
  deleteOntology,
  type Ontology,
} from "@/lib/api";

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
      setOntologies(data || []); // Handle null/undefined responses
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontologies");
      setOntologies([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to delete ontology "${name}"?`)) {
      return;
    }

    try {
      await deleteOntology(id);
      await loadOntologies();
    } catch (err) {
      alert(`Failed to delete ontology: ${err instanceof Error ? err.message : "Unknown error"}`);
    }
  };

  const handleExport = (id: string, name: string, format: string) => {
    window.open(`/api/v1/ontology/${id}/export?format=${format}`, "_blank");
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-orange">Ontologies</h1>
          <p className="text-gray-400 mt-1">
            Manage ontologies and knowledge graph schemas
          </p>
        </div>
        <Link
          href="/ontologies/upload"
          className="bg-orange hover:bg-orange/80 text-navy px-4 py-2 rounded-lg font-semibold"
        >
          Upload Ontology
        </Link>
      </div>

      <div className="mb-4 flex gap-4 items-center">
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium text-white">Status:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="bg-navy border border-blue rounded px-3 py-1 text-white focus:border-orange focus:ring-1 focus:ring-orange"
          >
            <option value="">All</option>
            <option value="active">Active</option>
            <option value="deprecated">Deprecated</option>
            <option value="draft">Draft</option>
            <option value="archived">Archived</option>
          </select>
        </label>
        <button
          onClick={loadOntologies}
          className="bg-blue hover:bg-orange text-white px-4 py-1 rounded border border-blue"
        >
          Refresh
        </button>
      </div>

      {loading && (
        <div className="text-center py-12">
          <p className="text-gray-400">Loading ontologies...</p>
        </div>
      )}

      {error && (
        <div className="bg-red-900/20 border border-red-500 text-red-400 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {!loading && !error && ontologies.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-400 mb-4">No ontologies found</p>
          <Link
            href="/ontologies/upload"
            className="text-orange hover:underline"
          >
            Upload your first ontology
          </Link>
        </div>
      )}

      {!loading && !error && ontologies.length > 0 && (
        <div className="bg-blue border border-blue rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-700">
            <thead className="bg-navy">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Version
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Format
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-blue divide-y divide-gray-700">
              {ontologies.map((ontology) => (
                <tr key={ontology.id} className="hover:bg-navy transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-white">
                      {ontology.name}
                    </div>
                    {ontology.description && (
                      <div className="text-sm text-gray-400">
                        {ontology.description}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-white">
                    {ontology.version}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-white">
                    {ontology.format}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        ontology.status === "active"
                          ? "bg-green-900/40 text-green-400 border border-green-500"
                          : ontology.status === "deprecated"
                          ? "bg-yellow-900/40 text-yellow-400 border border-yellow-500"
                          : ontology.status === "draft"
                          ? "bg-blue-900/40 text-blue-400 border border-blue-500"
                          : "bg-gray-800 text-gray-400 border border-gray-600"
                      }`}
                    >
                      {ontology.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                    {new Date(ontology.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex justify-end gap-2">
                      <Link
                        href={`/ontologies/${ontology.id}`}
                        className="text-orange hover:text-orange/80"
                      >
                        View
                      </Link>
                      <button
                        onClick={() =>
                          handleExport(ontology.id, ontology.name, ontology.format)
                        }
                        className="text-green-400 hover:text-green-300"
                      >
                        Export
                      </button>
                      <button
                        onClick={() => handleDelete(ontology.id, ontology.name)}
                        className="text-red-400 hover:text-red-300"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-6">
        <p className="text-sm text-gray-400">
          Total: {ontologies.length} ontolog{ontologies.length === 1 ? "y" : "ies"}
        </p>
      </div>
    </div>
  );
}

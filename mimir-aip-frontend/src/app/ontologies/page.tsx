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
      setOntologies(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontologies");
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
          <h1 className="text-3xl font-bold">Ontologies</h1>
          <p className="text-gray-600 mt-1">
            Manage ontologies and knowledge graph schemas
          </p>
        </div>
        <Link
          href="/ontologies/upload"
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg"
        >
          Upload Ontology
        </Link>
      </div>

      <div className="mb-4 flex gap-4 items-center">
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium">Status:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="border rounded px-3 py-1"
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
          className="bg-gray-200 hover:bg-gray-300 px-4 py-1 rounded"
        >
          Refresh
        </button>
      </div>

      {loading && (
        <div className="text-center py-12">
          <p className="text-gray-600">Loading ontologies...</p>
        </div>
      )}

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {!loading && !error && ontologies.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-600 mb-4">No ontologies found</p>
          <Link
            href="/ontologies/upload"
            className="text-blue-600 hover:underline"
          >
            Upload your first ontology
          </Link>
        </div>
      )}

      {!loading && !error && ontologies.length > 0 && (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Version
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Format
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {ontologies.map((ontology) => (
                <tr key={ontology.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {ontology.name}
                    </div>
                    {ontology.description && (
                      <div className="text-sm text-gray-500">
                        {ontology.description}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {ontology.version}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {ontology.format}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        ontology.status === "active"
                          ? "bg-green-100 text-green-800"
                          : ontology.status === "deprecated"
                          ? "bg-yellow-100 text-yellow-800"
                          : ontology.status === "draft"
                          ? "bg-blue-100 text-blue-800"
                          : "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {ontology.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(ontology.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex justify-end gap-2">
                      <Link
                        href={`/ontologies/${ontology.id}`}
                        className="text-blue-600 hover:text-blue-900"
                      >
                        View
                      </Link>
                      <button
                        onClick={() =>
                          handleExport(ontology.id, ontology.name, ontology.format)
                        }
                        className="text-green-600 hover:text-green-900"
                      >
                        Export
                      </button>
                      <button
                        onClick={() => handleDelete(ontology.id, ontology.name)}
                        className="text-red-600 hover:text-red-900"
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
        <p className="text-sm text-gray-600">
          Total: {ontologies.length} ontolog{ontologies.length === 1 ? "y" : "ies"}
        </p>
      </div>
    </div>
  );
}

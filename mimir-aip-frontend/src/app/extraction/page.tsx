"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  listExtractionJobs,
  listOntologies,
  type ExtractionJob,
  type Ontology,
} from "@/lib/api";

export default function ExtractionJobsPage() {
  const [jobs, setJobs] = useState<ExtractionJob[]>([]);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [ontologyFilter, setOntologyFilter] = useState<string>("");

  useEffect(() => {
    loadOntologies();
  }, []);

  useEffect(() => {
    loadJobs();
  }, [statusFilter, ontologyFilter]);

  const loadOntologies = async () => {
    try {
      const data = await listOntologies("");
      setOntologies(data);
    } catch (err) {
      console.error("Failed to load ontologies:", err);
    }
  };

  const loadJobs = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await listExtractionJobs({
        status: statusFilter || undefined,
        ontology_id: ontologyFilter || undefined,
      });
      setJobs(response.data.jobs || []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load extraction jobs"
      );
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800";
      case "running":
        return "bg-blue-100 text-blue-800";
      case "failed":
        return "bg-red-100 text-red-800";
      case "pending":
        return "bg-yellow-100 text-yellow-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  const getExtractionTypeBadge = (type: string) => {
    switch (type) {
      case "deterministic":
        return "bg-purple-100 text-purple-800";
      case "llm":
        return "bg-indigo-100 text-indigo-800";
      case "hybrid":
        return "bg-pink-100 text-pink-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getOntologyName = (ontologyId: string) => {
    const ontology = ontologies.find((o) => o.id === ontologyId);
    return ontology?.name || ontologyId;
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold">Entity Extraction Jobs</h1>
        <p className="text-gray-600 mt-1">
          Track and manage entity extraction from data sources
        </p>
      </div>

      <div className="mb-4 flex gap-4 items-center flex-wrap">
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium">Ontology:</span>
          <select
            value={ontologyFilter}
            onChange={(e) => setOntologyFilter(e.target.value)}
            className="border rounded px-3 py-1"
          >
            <option value="">All</option>
            {ontologies.map((ont) => (
              <option key={ont.id} value={ont.id}>
                {ont.name}
              </option>
            ))}
          </select>
        </label>
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium">Status:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="border rounded px-3 py-1"
          >
            <option value="">All</option>
            <option value="pending">Pending</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </label>
        <button
          onClick={loadJobs}
          className="bg-gray-200 hover:bg-gray-300 px-4 py-1 rounded"
        >
          Refresh
        </button>
      </div>

      {loading && (
        <div className="text-center py-12">
          <p className="text-gray-600">Loading extraction jobs...</p>
        </div>
      )}

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {!loading && !error && jobs.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-600 mb-4">No extraction jobs found</p>
          <p className="text-sm text-gray-500">
            Use the REST API to create extraction jobs
          </p>
        </div>
      )}

      {!loading && !error && jobs.length > 0 && (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Job Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Ontology
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Entities
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Triples
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
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {job.job_name}
                    </div>
                    <div className="text-xs text-gray-500">{job.id}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {getOntologyName(job.ontology_id)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getExtractionTypeBadge(
                        job.extraction_type
                      )}`}
                    >
                      {job.extraction_type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusBadgeClass(
                        job.status
                      )}`}
                    >
                      {job.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {job.entities_extracted}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {job.triples_generated}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDate(job.created_at)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <Link
                      href={`/extraction/${job.id}`}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      View Details
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

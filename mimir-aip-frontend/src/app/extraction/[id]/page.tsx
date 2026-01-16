"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import {
  getExtractionJob,
  getOntology,
  type ExtractionJob,
  type ExtractedEntity,
  type Ontology,
} from "@/lib/api";

export default function ExtractionJobDetailsPage() {
  const params = useParams();
  const jobId = params.id as string;

  const [job, setJob] = useState<ExtractionJob | null>(null);
  const [entities, setEntities] = useState<ExtractedEntity[]>([]);
  const [ontology, setOntology] = useState<Ontology | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedEntity, setSelectedEntity] = useState<ExtractedEntity | null>(null);

  useEffect(() => {
    if (jobId) {
      loadJobDetails();
    }
  }, [jobId]);

  const loadJobDetails = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getExtractionJob(jobId);
      setJob(response.data.job);
      setEntities(response.data.entities || []);

      // Load ontology info
      if (response.data.job.ontology_id) {
        try {
          const ontData = await getOntology(response.data.job.ontology_id);
          if (ontData.success && ontData.data.ontology) {
            setOntology(ontData.data.ontology);
          }
        } catch (err) {
          console.error("Failed to load ontology:", err);
        }
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load extraction job"
      );
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-900/40 text-green-400 border border-green-500";
      case "running":
        return "bg-blue-900/40 text-blue-400 border border-blue-500";
      case "failed":
        return "bg-red-900/40 text-red-400 border border-red-500";
      case "pending":
        return "bg-yellow-900/40 text-yellow-400 border border-yellow-500";
      default:
        return "bg-gray-800 text-gray-400 border border-gray-600";
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  };

  const formatDuration = () => {
    if (!job?.started_at || !job?.completed_at) return "N/A";
    const start = new Date(job.started_at).getTime();
    const end = new Date(job.completed_at).getTime();
    const durationMs = end - start;
    const seconds = Math.floor(durationMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return minutes > 0
      ? `${minutes}m ${remainingSeconds}s`
      : `${remainingSeconds}s`;
  };

  if (loading) {
    return (
      <div className="p-6">
        <p className="text-gray-400">Loading extraction job details...</p>
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="p-6">
        <div className="bg-red-900/20 border border-red-400 text-red-400 px-4 py-3 rounded">
          {error || "Job not found"}
        </div>
        <Link
          href="/extraction"
          className="text-orange hover:underline mt-4 inline-block"
        >
          ← Back to Extraction Jobs
        </Link>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-4">
        <Link href="/extraction" className="text-orange hover:underline">
          ← Back to Extraction Jobs
        </Link>
      </div>

      <div className="mb-6">
        <h1 className="text-3xl font-bold text-orange">{job.job_name}</h1>
        <p className="text-gray-400 mt-1 text-sm">Job ID: {job.id}</p>
      </div>

      {/* Job Overview */}
      <div className="bg-blue rounded-lg shadow p-6 mb-6">
        <h2 className="text-xl font-semibold mb-4 text-white">Job Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <p className="text-sm text-gray-400">Status</p>
            <span
              className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusBadgeClass(
                job.status
              )}`}
            >
              {job.status}
            </span>
          </div>
          <div>
            <p className="text-sm text-gray-400">Extraction Type</p>
            <p className="font-medium capitalize text-white">{job.extraction_type}</p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Source Type</p>
            <p className="font-medium text-white">{job.source_type}</p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Duration</p>
            <p className="font-medium text-white">{formatDuration()}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4">
          <div>
            <p className="text-sm text-gray-400">Entities Extracted</p>
            <p className="text-2xl font-bold text-orange">
              {job.entities_extracted}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Triples Generated</p>
            <p className="text-2xl font-bold text-green-400">
              {job.triples_generated}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Created</p>
            <p className="text-sm text-white">{formatDate(job.created_at)}</p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Completed</p>
            <p className="text-sm text-white">{formatDate(job.completed_at)}</p>
          </div>
        </div>

        {ontology && (
          <div className="mt-4 pt-4 border-t border-gray-600">
            <p className="text-sm text-gray-400 mb-2">Ontology</p>
            <Link
              href={`/ontologies/${ontology.id}`}
              className="text-orange hover:underline font-medium"
            >
              {ontology.name} (v{ontology.version})
            </Link>
          </div>
        )}

        {job.error_message && (
          <div className="mt-4 pt-4 border-t border-gray-600">
            <p className="text-sm text-gray-400 mb-2">Error Message</p>
            <div className="bg-red-900/20 border border-red-400 text-red-400 px-3 py-2 rounded text-sm">
              {job.error_message}
            </div>
          </div>
        )}
      </div>

      {/* Extracted Entities */}
      <div className="bg-blue rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4 text-white">
          Extracted Entities ({entities.length})
        </h2>

        {entities.length === 0 ? (
          <p className="text-gray-400 text-center py-8">No entities extracted</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-700">
              <thead className="bg-navy">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Label
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    URI
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Confidence
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-blue divide-y divide-gray-700">
                {entities.map((entity) => (
                  <tr key={entity.id} className="hover:bg-navy">
                    <td className="px-4 py-3 text-sm text-white">
                      {entity.entity_label || "Untitled"}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-400">
                      {entity.entity_type.split("/").pop() ||
                        entity.entity_type}
                    </td>
                    <td className="px-4 py-3 text-xs text-gray-400 max-w-md truncate">
                      {entity.entity_uri}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {entity.confidence !== undefined ? (
                        <span
                          className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                            entity.confidence >= 0.8
                              ? "bg-green-900/40 text-green-400 border border-green-500"
                              : entity.confidence >= 0.5
                              ? "bg-yellow-900/40 text-yellow-400 border border-yellow-500"
                              : "bg-red-900/40 text-red-400 border border-red-500"
                          }`}
                        >
                          {(entity.confidence * 100).toFixed(0)}%
                        </span>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <button
                        onClick={() => setSelectedEntity(entity)}
                        className="text-orange hover:text-orange/80"
                      >
                        View Details
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Entity Details Modal */}
      {selectedEntity && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50" role="dialog" data-testid="entity-modal">
          <div className="bg-blue rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex justify-between items-start mb-4">
                <h3 className="text-lg font-semibold text-white">
                  {selectedEntity.entity_label || "Entity Details"}
                </h3>
                <button
                  onClick={() => setSelectedEntity(null)}
                  className="text-gray-400 hover:text-white text-xl"
                  aria-label="Close"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <p className="text-sm text-gray-400">URI</p>
                  <p className="text-sm font-mono break-all text-white">
                    {selectedEntity.entity_uri}
                  </p>
                </div>

                <div>
                  <p className="text-sm text-gray-400">Type</p>
                  <p className="text-sm font-mono text-white">{selectedEntity.entity_type}</p>
                </div>

                {selectedEntity.confidence !== undefined && (
                  <div>
                    <p className="text-sm text-gray-400">Confidence</p>
                    <p className="text-sm text-white">
                      {(selectedEntity.confidence * 100).toFixed(2)}%
                    </p>
                  </div>
                )}

                {selectedEntity.source_text && (
                  <div>
                    <p className="text-sm text-gray-400">Source Text</p>
                    <p className="text-sm bg-navy p-3 rounded text-white">
                      {selectedEntity.source_text}
                    </p>
                  </div>
                )}

                {selectedEntity.properties &&
                  Object.keys(selectedEntity.properties).length > 0 && (
                    <div>
                      <p className="text-sm text-gray-400 mb-2">Properties</p>
                      <div className="bg-navy p-3 rounded">
                        <pre className="text-xs overflow-x-auto text-white">
                          {JSON.stringify(selectedEntity.properties, null, 2)}
                        </pre>
                      </div>
                    </div>
                  )}
              </div>

              <div className="mt-6 flex justify-end">
                <button
                  onClick={() => setSelectedEntity(null)}
                  className="bg-orange hover:bg-orange/80 text-white px-4 py-2 rounded"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

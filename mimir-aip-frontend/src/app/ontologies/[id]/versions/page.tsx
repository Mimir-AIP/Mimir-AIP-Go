"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getOntology,
  listOntologyVersions,
  createOntologyVersion,
  compareOntologyVersions,
  deleteOntologyVersion,
  type Ontology,
  type OntologyVersion,
  type VersionDiff,
} from "@/lib/api";

export default function OntologyVersionsPage() {
  const params = useParams();
  const router = useRouter();
  const ontologyId = params.id as string;

  const [ontology, setOntology] = useState<Ontology | null>(null);
  const [versions, setVersions] = useState<OntologyVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create version modal state
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newVersion, setNewVersion] = useState("");
  const [changelog, setChangelog] = useState("");
  const [createdBy, setCreatedBy] = useState("");
  const [creating, setCreating] = useState(false);

  // Compare modal state
  const [showCompareModal, setShowCompareModal] = useState(false);
  const [compareV1, setCompareV1] = useState("");
  const [compareV2, setCompareV2] = useState("");
  const [comparing, setComparing] = useState(false);
  const [diffResult, setDiffResult] = useState<VersionDiff | null>(null);

  useEffect(() => {
    loadData();
  }, [ontologyId]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [ontologyRes, versionsRes] = await Promise.all([
        getOntology(ontologyId),
        listOntologyVersions(ontologyId),
      ]);

      if (ontologyRes.success) {
        setOntology(ontologyRes.data);
      }

      if (versionsRes.success) {
        setVersions(versionsRes.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateVersion = async () => {
    if (!newVersion.trim()) {
      alert("Version is required");
      return;
    }

    try {
      setCreating(true);
      const response = await createOntologyVersion(
        ontologyId,
        newVersion,
        changelog,
        createdBy || undefined
      );

      if (response.success) {
        setShowCreateModal(false);
        setNewVersion("");
        setChangelog("");
        setCreatedBy("");
        loadData();
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create version");
    } finally {
      setCreating(false);
    }
  };

  const handleCompare = async () => {
    if (!compareV1 || !compareV2) {
      alert("Please select two versions to compare");
      return;
    }

    try {
      setComparing(true);
      const response = await compareOntologyVersions(ontologyId, compareV1, compareV2);

      if (response.success) {
        setDiffResult(response.data);
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to compare versions");
    } finally {
      setComparing(false);
    }
  };

  const handleDelete = async (versionId: number) => {
    if (!confirm("Are you sure you want to delete this version? This cannot be undone.")) {
      return;
    }

    try {
      await deleteOntologyVersion(ontologyId, versionId);
      loadData();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete version");
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="text-center py-12">Loading versions...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-2 text-sm text-gray-600 mb-2">
          <Link href="/ontologies" className="hover:text-blue-600">
            Ontologies
          </Link>
          <span>/</span>
          <Link href={`/ontologies/${ontologyId}`} className="hover:text-blue-600">
            {ontology?.name || ontologyId}
          </Link>
          <span>/</span>
          <span>Versions</span>
        </div>
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold">Version History</h1>
            <p className="text-gray-600 mt-1">
              Manage and compare versions of {ontology?.name}
            </p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowCompareModal(true)}
              disabled={versions.length < 2}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded"
            >
              Compare Versions
            </button>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded"
            >
              Create Version
            </button>
          </div>
        </div>
      </div>

      {/* Versions Timeline */}
      {versions.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-8 text-center text-gray-500">
          No versions yet. Create your first version to track ontology changes over time.
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow">
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead className="bg-gray-100 border-b">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Version
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Changelog
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Previous
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Created By
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {versions.map((version, idx) => (
                  <tr key={version.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-semibold text-blue-600">
                          {version.version}
                        </span>
                        {idx === 0 && (
                          <span className="px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded">
                            Latest
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="max-w-md truncate text-sm text-gray-700">
                        {version.changelog || "-"}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="font-mono text-sm text-gray-600">
                        {version.previous_version || "-"}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {new Date(version.created_at).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {version.created_by || "-"}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex gap-2">
                        <Link
                          href={`/ontologies/${ontologyId}/versions/${version.id}`}
                          className="text-blue-600 hover:text-blue-900 text-sm"
                        >
                          Details
                        </Link>
                        {idx !== 0 && (
                          <button
                            onClick={() => handleDelete(version.id)}
                            className="text-red-600 hover:text-red-900 text-sm"
                          >
                            Delete
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Create Version Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4">
            <div className="p-6">
              <h2 className="text-2xl font-bold mb-4">Create New Version</h2>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Version Number *
                  </label>
                  <input
                    type="text"
                    value={newVersion}
                    onChange={(e) => setNewVersion(e.target.value)}
                    placeholder="e.g., 1.1.0, v2.0"
                    className="w-full border rounded px-3 py-2"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Changelog
                  </label>
                  <textarea
                    value={changelog}
                    onChange={(e) => setChangelog(e.target.value)}
                    placeholder="Describe what changed in this version..."
                    className="w-full border rounded px-3 py-2 h-32"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Created By
                  </label>
                  <input
                    type="text"
                    value={createdBy}
                    onChange={(e) => setCreatedBy(e.target.value)}
                    placeholder="Your name or email"
                    className="w-full border rounded px-3 py-2"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-2 mt-6">
                <button
                  onClick={() => setShowCreateModal(false)}
                  disabled={creating}
                  className="px-4 py-2 border rounded text-gray-700 hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreateVersion}
                  disabled={creating}
                  className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white rounded"
                >
                  {creating ? "Creating..." : "Create Version"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Compare Versions Modal */}
      {showCompareModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <h2 className="text-2xl font-bold mb-4">Compare Versions</h2>

              {!diffResult ? (
                <div className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        From Version
                      </label>
                      <select
                        value={compareV1}
                        onChange={(e) => setCompareV1(e.target.value)}
                        className="w-full border rounded px-3 py-2"
                      >
                        <option value="">Select version...</option>
                        {versions.map((v) => (
                          <option key={v.id} value={v.version}>
                            {v.version}
                          </option>
                        ))}
                      </select>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        To Version
                      </label>
                      <select
                        value={compareV2}
                        onChange={(e) => setCompareV2(e.target.value)}
                        className="w-full border rounded px-3 py-2"
                      >
                        <option value="">Select version...</option>
                        {versions.map((v) => (
                          <option key={v.id} value={v.version}>
                            {v.version}
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>

                  <div className="flex justify-end gap-2 mt-6">
                    <button
                      onClick={() => {
                        setShowCompareModal(false);
                        setCompareV1("");
                        setCompareV2("");
                        setDiffResult(null);
                      }}
                      disabled={comparing}
                      className="px-4 py-2 border rounded text-gray-700 hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleCompare}
                      disabled={comparing || !compareV1 || !compareV2}
                      className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded"
                    >
                      {comparing ? "Comparing..." : "Compare"}
                    </button>
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  {/* Summary */}
                  <div className="bg-gray-50 rounded-lg p-4">
                    <h3 className="font-semibold mb-2">
                      Changes from {diffResult.from_version} to {diffResult.to_version}
                    </h3>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <div className="text-gray-600">Classes</div>
                        <div className="flex gap-3 mt-1">
                          <span className="text-green-600">+{diffResult.summary.classes_added}</span>
                          <span className="text-red-600">-{diffResult.summary.classes_removed}</span>
                          <span className="text-blue-600">~{diffResult.summary.classes_modified}</span>
                        </div>
                      </div>
                      <div>
                        <div className="text-gray-600">Properties</div>
                        <div className="flex gap-3 mt-1">
                          <span className="text-green-600">+{diffResult.summary.properties_added}</span>
                          <span className="text-red-600">-{diffResult.summary.properties_removed}</span>
                          <span className="text-blue-600">~{diffResult.summary.properties_modified}</span>
                        </div>
                      </div>
                      <div>
                        <div className="text-gray-600">Total Changes</div>
                        <div className="font-semibold mt-1">{diffResult.summary.total_changes}</div>
                      </div>
                    </div>
                  </div>

                  {/* Detailed Changes */}
                  {diffResult.changes.length > 0 ? (
                    <div className="max-h-96 overflow-y-auto">
                      <table className="min-w-full text-sm">
                        <thead className="bg-gray-100 sticky top-0">
                          <tr>
                            <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                              Change
                            </th>
                            <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                              Type
                            </th>
                            <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                              Entity
                            </th>
                            <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                              Description
                            </th>
                          </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-gray-200">
                          {diffResult.changes.map((change, idx) => (
                            <tr key={idx}>
                              <td className="px-3 py-2">
                                <span
                                  className={`px-2 py-0.5 text-xs rounded ${
                                    change.change_type.includes("add")
                                      ? "bg-green-100 text-green-800"
                                      : change.change_type.includes("remove")
                                      ? "bg-red-100 text-red-800"
                                      : "bg-blue-100 text-blue-800"
                                  }`}
                                >
                                  {change.change_type}
                                </span>
                              </td>
                              <td className="px-3 py-2 text-gray-600">{change.entity_type}</td>
                              <td className="px-3 py-2 font-mono text-xs text-gray-700 truncate max-w-xs">
                                {change.entity_uri}
                              </td>
                              <td className="px-3 py-2 text-gray-600">
                                {change.description || "-"}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <div className="text-center py-8 text-gray-500">
                      No changes between these versions
                    </div>
                  )}

                  <div className="flex justify-end gap-2 mt-6">
                    <button
                      onClick={() => {
                        setShowCompareModal(false);
                        setCompareV1("");
                        setCompareV2("");
                        setDiffResult(null);
                      }}
                      className="px-4 py-2 border rounded text-gray-700 hover:bg-gray-50"
                    >
                      Close
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

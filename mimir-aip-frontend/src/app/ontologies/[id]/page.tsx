"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getOntology,
  getOntologyStats,
  deleteOntology,
  executeSPARQLQuery,
  type Ontology,
  type SPARQLQueryResult,
} from "@/lib/api";

interface OntologyStats {
  total_classes: number;
  total_properties: number;
  total_triples: number;
  total_entities: number;
  classes?: Array<{
    uri: string;
    label?: string;
    description?: string;
    parent_uris?: string[];
  }>;
  properties?: Array<{
    uri: string;
    label?: string;
    description?: string;
    property_type: string;
    domain?: string[];
    range?: string[];
  }>;
}

export default function OntologyDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const ontologyId = params.id as string;

  const [ontology, setOntology] = useState<Ontology | null>(null);
  const [stats, setStats] = useState<OntologyStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"overview" | "classes" | "properties" | "queries">("overview");
  const [queryResult, setQueryResult] = useState<SPARQLQueryResult | null>(null);
  const [queryLoading, setQueryLoading] = useState(false);
  const [queryError, setQueryError] = useState<string | null>(null);

  const loadOntologyData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [ontologyResponse, statsResponse] = await Promise.all([
        getOntology(ontologyId, true),
        getOntologyStats(ontologyId),
      ]);

      if (ontologyResponse.success && ontologyResponse.data.ontology) {
        setOntology(ontologyResponse.data.ontology);
      }

      if (statsResponse.success && statsResponse.data.stats) {
        setStats(statsResponse.data.stats as OntologyStats);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontology");
    } finally {
      setLoading(false);
    }
  }, [ontologyId]);

  useEffect(() => {
    loadOntologyData();
  }, [loadOntologyData]);

  const handleDelete = async () => {
    if (!ontology) return;
    
    if (!confirm(`Are you sure you want to delete ontology "${ontology.name}"?`)) {
      return;
    }

    try {
      await deleteOntology(ontologyId);
      router.push("/ontologies");
    } catch (err) {
      alert(`Failed to delete ontology: ${err instanceof Error ? err.message : "Unknown error"}`);
    }
  };

  const handleExport = () => {
    if (!ontology) return;
    window.open(`/api/v1/ontology/${ontologyId}/export?format=${ontology.format}`, "_blank");
  };

  const runSampleQuery = async (query: string) => {
    try {
      setQueryLoading(true);
      setQueryError(null);
      setQueryResult(null);

      const response = await executeSPARQLQuery(query);
      if (response.success) {
        setQueryResult(response.data);
      }
    } catch (err) {
      setQueryError(err instanceof Error ? err.message : "Query failed");
    } finally {
      setQueryLoading(false);
    }
  };

  const getSampleQueries = () => {
    if (!ontology) return [];
    
    return [
      {
        name: "List all classes",
        query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT ?class ?label
WHERE {
  GRAPH <${ontology.tdb2_graph}> {
    ?class a owl:Class .
    OPTIONAL { ?class rdfs:label ?label }
  }
}
LIMIT 100`,
      },
      {
        name: "List all properties",
        query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT ?property ?label ?type
WHERE {
  GRAPH <${ontology.tdb2_graph}> {
    {
      ?property a owl:ObjectProperty .
      BIND("object" AS ?type)
    } UNION {
      ?property a owl:DatatypeProperty .
      BIND("datatype" AS ?type)
    }
    OPTIONAL { ?property rdfs:label ?label }
  }
}
LIMIT 100`,
      },
      {
        name: "Show class hierarchy",
        query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT ?subClass ?subLabel ?superClass ?superLabel
WHERE {
  GRAPH <${ontology.tdb2_graph}> {
    ?subClass rdfs:subClassOf ?superClass .
    OPTIONAL { ?subClass rdfs:label ?subLabel }
    OPTIONAL { ?superClass rdfs:label ?superLabel }
  }
}
LIMIT 50`,
      },
    ];
  };

  const extractLocalName = (uri: string) => {
    const parts = uri.split(/[/#]/);
    return parts[parts.length - 1] || uri;
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <p className="text-gray-600">Loading ontology...</p>
        </div>
      </div>
    );
  }

  if (error || !ontology) {
    return (
      <div className="p-6">
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error || "Ontology not found"}
        </div>
        <Link href="/ontologies" className="text-blue-600 hover:underline">
          Back to Ontologies
        </Link>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-2 text-sm text-gray-600 mb-2">
          <Link href="/ontologies" className="hover:underline">
            Ontologies
          </Link>
          <span>/</span>
          <span>{ontology.name}</span>
        </div>
        
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold">{ontology.name}</h1>
            {ontology.description && (
              <p className="text-gray-600 mt-1">{ontology.description}</p>
            )}
          </div>
          
          <div className="flex gap-2">
            <Link
              href={`/ontologies/${ontologyId}/versions`}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg inline-flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Versions
            </Link>
            <Link
              href={`/ontologies/${ontologyId}/suggestions`}
              className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded-lg inline-flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
              </svg>
              Suggestions
            </Link>
            <button
              onClick={handleExport}
              className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg"
            >
              Export
            </button>
            <button
              onClick={handleDelete}
              className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg"
            >
              Delete
            </button>
          </div>
        </div>
      </div>

      {/* Metadata Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm text-gray-600">Version</div>
          <div className="text-2xl font-bold">{ontology.version}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm text-gray-600">Status</div>
          <div className="text-2xl font-bold">
            <span
              className={`px-2 inline-flex text-sm leading-5 font-semibold rounded-full ${
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
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm text-gray-600">Format</div>
          <div className="text-2xl font-bold uppercase">{ontology.format}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm text-gray-600">Created</div>
          <div className="text-lg font-bold">
            {new Date(ontology.created_at).toLocaleDateString()}
          </div>
        </div>
      </div>

      {/* Statistics Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-blue-50 rounded-lg shadow p-4">
            <div className="text-sm text-blue-600">Classes</div>
            <div className="text-3xl font-bold text-blue-900">{stats.total_classes}</div>
          </div>
          <div className="bg-purple-50 rounded-lg shadow p-4">
            <div className="text-sm text-purple-600">Properties</div>
            <div className="text-3xl font-bold text-purple-900">{stats.total_properties}</div>
          </div>
          <div className="bg-green-50 rounded-lg shadow p-4">
            <div className="text-sm text-green-600">Triples</div>
            <div className="text-3xl font-bold text-green-900">{stats.total_triples}</div>
          </div>
          <div className="bg-orange-50 rounded-lg shadow p-4">
            <div className="text-sm text-orange-600">Entities</div>
            <div className="text-3xl font-bold text-orange-900">{stats.total_entities}</div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 mb-4">
        <nav className="-mb-px flex space-x-8">
          {(["overview", "classes", "properties", "queries"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`${
                activeTab === tab
                  ? "border-blue-500 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
              } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm capitalize`}
            >
              {tab}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="bg-white rounded-lg shadow p-6">
        {activeTab === "overview" && (
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">Ontology Details</h3>
              <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-gray-500">ID</dt>
                  <dd className="mt-1 text-sm text-gray-900 font-mono">{ontology.id}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500">TDB2 Graph URI</dt>
                  <dd className="mt-1 text-sm text-gray-900 font-mono break-all">{ontology.tdb2_graph}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500">File Path</dt>
                  <dd className="mt-1 text-sm text-gray-900 font-mono">{ontology.file_path}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-gray-500">Last Updated</dt>
                  <dd className="mt-1 text-sm text-gray-900">{new Date(ontology.updated_at).toLocaleString()}</dd>
                </div>
                {ontology.created_by && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Created By</dt>
                    <dd className="mt-1 text-sm text-gray-900">{ontology.created_by}</dd>
                  </div>
                )}
              </dl>
            </div>

            <div>
              <h3 className="text-lg font-semibold mb-2">Quick Actions</h3>
              <div className="flex gap-2">
                <button
                  onClick={() => setActiveTab("queries")}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
                >
                  Run Sample Queries
                </button>
                <Link
                  href="/knowledge-graph"
                  className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded inline-block"
                >
                  Open Query Editor
                </Link>
              </div>
            </div>
          </div>
        )}

        {activeTab === "classes" && (
          <div>
            <h3 className="text-lg font-semibold mb-4">Classes ({stats?.total_classes || 0})</h3>
            {stats?.classes && stats.classes.length > 0 ? (
              <div className="space-y-3">
                {stats.classes.map((cls, idx) => (
                  <div key={idx} className="border rounded-lg p-4 hover:bg-gray-50">
                    <div className="font-medium text-blue-600 font-mono text-sm break-all">
                      {cls.uri}
                    </div>
                    {cls.label && (
                      <div className="text-sm text-gray-900 mt-1">{cls.label}</div>
                    )}
                    {cls.description && (
                      <div className="text-sm text-gray-600 mt-1">{cls.description}</div>
                    )}
                    {cls.parent_uris && cls.parent_uris.length > 0 && (
                      <div className="text-xs text-gray-500 mt-2">
                        <span className="font-medium">Extends:</span>{" "}
                        {cls.parent_uris.map(extractLocalName).join(", ")}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-600">No class information available. Try running a sample query.</p>
            )}
          </div>
        )}

        {activeTab === "properties" && (
          <div>
            <h3 className="text-lg font-semibold mb-4">Properties ({stats?.total_properties || 0})</h3>
            {stats?.properties && stats.properties.length > 0 ? (
              <div className="space-y-3">
                {stats.properties.map((prop, idx) => (
                  <div key={idx} className="border rounded-lg p-4 hover:bg-gray-50">
                    <div className="flex items-center gap-2">
                      <div className="font-medium text-purple-600 font-mono text-sm break-all">
                        {prop.uri}
                      </div>
                      <span className="text-xs px-2 py-1 bg-purple-100 text-purple-700 rounded">
                        {prop.property_type}
                      </span>
                    </div>
                    {prop.label && (
                      <div className="text-sm text-gray-900 mt-1">{prop.label}</div>
                    )}
                    {prop.description && (
                      <div className="text-sm text-gray-600 mt-1">{prop.description}</div>
                    )}
                    <div className="flex gap-4 mt-2 text-xs text-gray-500">
                      {prop.domain && prop.domain.length > 0 && (
                        <div>
                          <span className="font-medium">Domain:</span>{" "}
                          {prop.domain.map(extractLocalName).join(", ")}
                        </div>
                      )}
                      {prop.range && prop.range.length > 0 && (
                        <div>
                          <span className="font-medium">Range:</span>{" "}
                          {prop.range.map(extractLocalName).join(", ")}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-600">No property information available. Try running a sample query.</p>
            )}
          </div>
        )}

        {activeTab === "queries" && (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Sample SPARQL Queries</h3>
            <p className="text-sm text-gray-600">
              Click on a sample query below to execute it against this ontology.
            </p>

            <div className="space-y-3">
              {getSampleQueries().map((sample, idx) => (
                <div key={idx} className="border rounded-lg p-4">
                  <div className="flex justify-between items-start mb-2">
                    <h4 className="font-medium">{sample.name}</h4>
                    <button
                      onClick={() => runSampleQuery(sample.query)}
                      disabled={queryLoading}
                      className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white px-3 py-1 rounded text-sm"
                    >
                      {queryLoading ? "Running..." : "Run Query"}
                    </button>
                  </div>
                  <pre className="bg-gray-50 p-3 rounded text-xs overflow-x-auto">
                    {sample.query}
                  </pre>
                </div>
              ))}
            </div>

            {queryError && (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                {queryError}
              </div>
            )}

            {queryResult && (
              <div className="border rounded-lg p-4 bg-green-50">
                <h4 className="font-medium mb-2">Query Results</h4>
                <div className="text-sm text-gray-600 mb-2">
                  Found {queryResult.bindings?.length || 0} results
                </div>
                {queryResult.bindings && queryResult.bindings.length > 0 && (
                  <div className="overflow-x-auto">
                    <table className="min-w-full text-sm">
                      <thead className="bg-gray-100">
                        <tr>
                          {queryResult.variables?.map((variable: string) => (
                            <th key={variable} className="px-3 py-2 text-left font-medium">
                              {variable}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {queryResult.bindings.slice(0, 20).map((binding: Record<string, { value?: string }>, idx: number) => (
                          <tr key={idx} className="border-t">
                            {queryResult.variables?.map((variable: string) => (
                              <td key={variable} className="px-3 py-2 font-mono text-xs">
                                {binding[variable]?.value || "-"}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {queryResult.bindings.length > 20 && (
                      <p className="text-xs text-gray-500 mt-2">
                        Showing first 20 of {queryResult.bindings.length} results
                      </p>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

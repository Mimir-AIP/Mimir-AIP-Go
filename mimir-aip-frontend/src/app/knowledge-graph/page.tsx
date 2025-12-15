"use client";

import { useState } from "react";
import {
  executeSPARQLQuery,
  getKnowledgeGraphStats,
  type SPARQLQueryResult,
  type KnowledgeGraphStats,
} from "@/lib/api";
import { useEffect } from "react";

const SAMPLE_QUERIES = [
  {
    name: "Count all triples",
    query: `SELECT (COUNT(*) AS ?count)
WHERE {
  ?s ?p ?o
}`,
  },
  {
    name: "List all classes",
    query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT DISTINCT ?class ?label
WHERE {
  ?class a owl:Class .
  OPTIONAL { ?class rdfs:label ?label }
}
ORDER BY ?class
LIMIT 100`,
  },
  {
    name: "List all properties",
    query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT DISTINCT ?property ?label ?type
WHERE {
  {
    ?property a owl:ObjectProperty .
    BIND("object" AS ?type)
  } UNION {
    ?property a owl:DatatypeProperty .
    BIND("datatype" AS ?type)
  }
  OPTIONAL { ?property rdfs:label ?label }
}
ORDER BY ?property
LIMIT 100`,
  },
  {
    name: "Show class hierarchy",
    query: `PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

SELECT ?subClass ?subLabel ?superClass ?superLabel
WHERE {
  ?subClass rdfs:subClassOf ?superClass .
  OPTIONAL { ?subClass rdfs:label ?subLabel }
  OPTIONAL { ?superClass rdfs:label ?superLabel }
}
LIMIT 100`,
  },
  {
    name: "List all named graphs",
    query: `SELECT DISTINCT ?graph
WHERE {
  GRAPH ?graph { ?s ?p ?o }
}`,
  },
];

export default function KnowledgeGraphPage() {
  const [query, setQuery] = useState(SAMPLE_QUERIES[0].query);
  const [queryResult, setQueryResult] = useState<SPARQLQueryResult | null>(null);
  const [stats, setStats] = useState<KnowledgeGraphStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [queryHistory, setQueryHistory] = useState<string[]>([]);

  useEffect(() => {
    loadStats();
    // Load query history from localStorage
    const saved = localStorage.getItem("sparql_history");
    if (saved) {
      try {
        setQueryHistory(JSON.parse(saved));
      } catch (e) {
        console.error("Failed to load query history:", e);
      }
    }
  }, []);

  const loadStats = async () => {
    try {
      const response = await getKnowledgeGraphStats();
      if (response.success) {
        setStats(response.data);
      }
    } catch (err) {
      console.error("Failed to load stats:", err);
    }
  };

  const handleRunQuery = async () => {
    if (!query.trim()) {
      setError("Query cannot be empty");
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setQueryResult(null);

      const response = await executeSPARQLQuery(query);
      if (response.success) {
        setQueryResult(response.data);
        
        // Add to history (avoid duplicates)
        const newHistory = [query, ...queryHistory.filter(q => q !== query)].slice(0, 10);
        setQueryHistory(newHistory);
        localStorage.setItem("sparql_history", JSON.stringify(newHistory));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Query execution failed");
    } finally {
      setLoading(false);
    }
  };

  const handleLoadSample = (sampleQuery: string) => {
    setQuery(sampleQuery);
    setError(null);
    setQueryResult(null);
  };

  const handleExportCSV = () => {
    if (!queryResult || !queryResult.bindings || queryResult.bindings.length === 0) {
      return;
    }

    const headers = queryResult.variables || [];
    const rows = queryResult.bindings.map((binding: Record<string, { value?: string }>) =>
      headers.map((header) => {
        const value = binding[header]?.value || "";
        return `"${value.replace(/"/g, '""')}"`;
      }).join(",")
    );

    const csv = [headers.join(","), ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `sparql_results_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportJSON = () => {
    if (!queryResult) {
      return;
    }

    const json = JSON.stringify(queryResult, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `sparql_results_${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-3xl font-bold">SPARQL Query Editor</h1>
        <p className="text-gray-600 mt-1">
          Query the knowledge graph using SPARQL
        </p>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-blue-50 rounded-lg shadow p-4">
            <div className="text-sm text-blue-600">Total Triples</div>
            <div className="text-3xl font-bold text-blue-900">{stats.total_triples.toLocaleString()}</div>
          </div>
          <div className="bg-purple-50 rounded-lg shadow p-4">
            <div className="text-sm text-purple-600">Subjects</div>
            <div className="text-3xl font-bold text-purple-900">{stats.total_subjects.toLocaleString()}</div>
          </div>
          <div className="bg-green-50 rounded-lg shadow p-4">
            <div className="text-sm text-green-600">Predicates</div>
            <div className="text-3xl font-bold text-green-900">{stats.total_predicates.toLocaleString()}</div>
          </div>
          <div className="bg-orange-50 rounded-lg shadow p-4">
            <div className="text-sm text-orange-600">Named Graphs</div>
            <div className="text-3xl font-bold text-orange-900">{stats.named_graphs?.length || 0}</div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Query Editor */}
        <div className="lg:col-span-2 space-y-4">
          {/* Query Editor */}
          <div className="bg-white rounded-lg shadow p-4">
            <div className="flex justify-between items-center mb-3">
              <h3 className="font-semibold">Query Editor</h3>
              <div className="flex gap-2">
                <button
                  onClick={() => setQuery("")}
                  className="text-sm text-gray-600 hover:text-gray-900 px-3 py-1 border rounded"
                >
                  Clear
                </button>
                <button
                  onClick={handleRunQuery}
                  disabled={loading}
                  className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white px-4 py-1 rounded"
                >
                  {loading ? "Running..." : "Run Query"}
                </button>
              </div>
            </div>
            <textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full h-64 font-mono text-sm border rounded p-3 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your SPARQL query here..."
              spellCheck={false}
            />
            <div className="mt-2 text-xs text-gray-500">
              Press Ctrl+Enter to run query (not implemented yet)
            </div>
          </div>

          {/* Error Display */}
          {error && (
            <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              <div className="font-semibold">Query Error</div>
              <div className="text-sm mt-1">{error}</div>
            </div>
          )}

          {/* Results Display */}
          {queryResult && (
            <div className="bg-white rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-3">
                <div>
                  <h3 className="font-semibold">Results</h3>
                  <div className="text-sm text-gray-600">
                    {queryResult.query_type === "SELECT" && (
                      <>
                        {queryResult.bindings?.length || 0} row(s) returned
                        {queryResult.duration && ` in ${queryResult.duration}ms`}
                      </>
                    )}
                    {queryResult.query_type === "ASK" && (
                      <>Result: {queryResult.boolean ? "true" : "false"}</>
                    )}
                    {queryResult.query_type === "CONSTRUCT" && (
                      <>Constructed {queryResult.bindings?.length || 0} triple(s)</>
                    )}
                  </div>
                </div>
                {queryResult.bindings && queryResult.bindings.length > 0 && (
                  <div className="flex gap-2">
                    <button
                      onClick={handleExportCSV}
                      className="text-sm text-green-600 hover:text-green-900 px-3 py-1 border rounded"
                    >
                      Export CSV
                    </button>
                    <button
                      onClick={handleExportJSON}
                      className="text-sm text-blue-600 hover:text-blue-900 px-3 py-1 border rounded"
                    >
                      Export JSON
                    </button>
                  </div>
                )}
              </div>

              {/* ASK Result */}
              {queryResult.query_type === "ASK" && (
                <div className={`text-center py-8 text-2xl font-bold ${queryResult.boolean ? "text-green-600" : "text-red-600"}`}>
                  {queryResult.boolean ? "TRUE" : "FALSE"}
                </div>
              )}

              {/* SELECT Results */}
              {queryResult.query_type === "SELECT" && queryResult.bindings && queryResult.bindings.length > 0 && (
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead className="bg-gray-100 border-b">
                      <tr>
                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">#</th>
                        {queryResult.variables?.map((variable: string) => (
                          <th key={variable} className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                            {variable}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {queryResult.bindings.map((binding: Record<string, { value?: string; type?: string; datatype?: string; "xml:lang"?: string }>, idx: number) => (
                        <tr key={idx} className="hover:bg-gray-50">
                          <td className="px-3 py-2 text-gray-400 text-xs">{idx + 1}</td>
                          {queryResult.variables?.map((variable: string) => {
                            const value = binding[variable];
                            return (
                              <td key={variable} className="px-3 py-2">
                                {value ? (
                                  <div className="font-mono text-xs">
                                    <div className="break-all">{value.value}</div>
                                    {value.type && (
                                      <div className="text-gray-400 text-xs mt-1">
                                        {value.type}
                                        {value.datatype && ` (${value.datatype})`}
                                        {value["xml:lang"] && ` @${value["xml:lang"]}`}
                                      </div>
                                    )}
                                  </div>
                                ) : (
                                  <span className="text-gray-400">-</span>
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Empty Results */}
              {queryResult.query_type === "SELECT" && (!queryResult.bindings || queryResult.bindings.length === 0) && (
                <div className="text-center py-8 text-gray-500">
                  No results found
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right Column - Sample Queries & History */}
        <div className="space-y-4">
          {/* Sample Queries */}
          <div className="bg-white rounded-lg shadow p-4">
            <h3 className="font-semibold mb-3">Sample Queries</h3>
            <div className="space-y-2">
              {SAMPLE_QUERIES.map((sample, idx) => (
                <button
                  key={idx}
                  onClick={() => handleLoadSample(sample.query)}
                  className="w-full text-left text-sm text-blue-600 hover:text-blue-900 hover:bg-blue-50 px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  {sample.name}
                </button>
              ))}
            </div>
          </div>

          {/* Query History */}
          {queryHistory.length > 0 && (
            <div className="bg-white rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-3">
                <h3 className="font-semibold">Query History</h3>
                <button
                  onClick={() => {
                    setQueryHistory([]);
                    localStorage.removeItem("sparql_history");
                  }}
                  className="text-xs text-red-600 hover:text-red-900"
                >
                  Clear
                </button>
              </div>
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {queryHistory.map((historicalQuery, idx) => (
                  <button
                    key={idx}
                    onClick={() => handleLoadSample(historicalQuery)}
                    className="w-full text-left text-xs text-gray-700 hover:bg-gray-50 px-3 py-2 rounded border"
                  >
                    <div className="font-mono truncate">{historicalQuery.split("\n")[0]}</div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Named Graphs */}
          {stats && stats.named_graphs && stats.named_graphs.length > 0 && (
            <div className="bg-white rounded-lg shadow p-4">
              <h3 className="font-semibold mb-3">Named Graphs</h3>
              <div className="space-y-1 max-h-64 overflow-y-auto">
                {stats.named_graphs.map((graph, idx) => (
                  <div key={idx} className="text-xs font-mono text-gray-700 break-all px-2 py-1 bg-gray-50 rounded">
                    {graph}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Tips */}
          <div className="bg-blue-50 rounded-lg shadow p-4">
            <h3 className="font-semibold mb-2 text-blue-900">Tips</h3>
            <ul className="text-xs text-blue-800 space-y-1">
              <li>• Use PREFIX to define namespace shortcuts</li>
              <li>• Add LIMIT to prevent large result sets</li>
              <li>• Use OPTIONAL for optional patterns</li>
              <li>• Use FILTER to constrain results</li>
              <li>• Check Named Graphs to query specific ontologies</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}

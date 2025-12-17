"use client";

import { useState } from "react";
import {
  executeSPARQLQuery,
  getKnowledgeGraphStats,
  executeNLQuery,
  listOntologies,
  type SPARQLQueryResult,
  type KnowledgeGraphStats,
  type NLQueryResult,
  type Ontology,
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
  const [activeTab, setActiveTab] = useState<"sparql" | "natural-language">("sparql");
  const [query, setQuery] = useState(SAMPLE_QUERIES[0].query);
  const [queryResult, setQueryResult] = useState<SPARQLQueryResult | null>(null);
  const [stats, setStats] = useState<KnowledgeGraphStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [queryHistory, setQueryHistory] = useState<string[]>([]);

  // Natural Language Query state
  const [nlQuestion, setNlQuestion] = useState("");
  const [nlResult, setNlResult] = useState<NLQueryResult | null>(null);
  const [nlLoading, setNlLoading] = useState(false);
  const [nlError, setNlError] = useState<string | null>(null);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [selectedOntology, setSelectedOntology] = useState<string>("");

  useEffect(() => {
    loadStats();
    loadOntologies();
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

  const loadOntologies = async () => {
    try {
      const data = await listOntologies("active");
      setOntologies(data || []); // Handle null/undefined responses
    } catch (err) {
      console.error("Failed to load ontologies:", err);
      setOntologies([]); // Set empty array on error
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
    const rows = queryResult.bindings.map((binding) =>
      headers.map((header) => {
        const bindingObj = binding as Record<string, { value?: string }>;
        const value = bindingObj[header]?.value || "";
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

  const handleAskQuestion = async () => {
    if (!nlQuestion.trim()) {
      setNlError("Please enter a question");
      return;
    }

    try {
      setNlLoading(true);
      setNlError(null);
      setNlResult(null);

      const response = await executeNLQuery(nlQuestion, selectedOntology || undefined);
      if (response.success) {
        setNlResult(response.data);
      }
    } catch (err) {
      setNlError(err instanceof Error ? err.message : "Failed to process question");
    } finally {
      setNlLoading(false);
    }
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-orange">Knowledge Graph Query</h1>
        <p className="text-gray-400 mt-1">
          Query the knowledge graph using SPARQL or natural language
        </p>
      </div>

      {/* Tab Navigation */}
      <div className="mb-6 border-b border-blue">
        <div className="flex gap-4">
          <button
            onClick={() => setActiveTab("sparql")}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === "sparql"
                ? "border-blue-600 text-orange"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            SPARQL Query
          </button>
          <button
            onClick={() => setActiveTab("natural-language")}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === "natural-language"
                ? "border-blue-600 text-orange"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            Natural Language
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-blue rounded-lg shadow p-4">
            <div className="text-sm text-orange">Total Triples</div>
            <div className="text-3xl font-bold text-white">{stats.total_triples.toLocaleString()}</div>
          </div>
          <div className="bg-blue rounded-lg shadow p-4">
            <div className="text-sm text-purple-600">Subjects</div>
            <div className="text-3xl font-bold text-white">{stats.total_subjects.toLocaleString()}</div>
          </div>
          <div className="bg-blue rounded-lg shadow p-4">
            <div className="text-sm text-green-600">Predicates</div>
            <div className="text-3xl font-bold text-white">{stats.total_predicates.toLocaleString()}</div>
          </div>
          <div className="bg-blue rounded-lg shadow p-4">
            <div className="text-sm text-orange-600">Named Graphs</div>
            <div className="text-3xl font-bold text-white">{stats.named_graphs?.length || 0}</div>
          </div>
        </div>
      )}

      {/* SPARQL Tab Content */}
      {activeTab === "sparql" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Query Editor */}
        <div className="lg:col-span-2 space-y-4">
          {/* Query Editor */}
          <div className="bg-blue rounded-lg shadow p-4">
            <div className="flex justify-between items-center mb-3">
              <h3 className="font-semibold text-white">Query Editor</h3>
              <div className="flex gap-2">
                <button
                  onClick={() => setQuery("")}
                  className="text-sm text-gray-400 hover:text-white px-3 py-1 border rounded"
                >
                  Clear
                </button>
                <button
                  onClick={handleRunQuery}
                  disabled={loading}
                  className="bg-orange hover:bg-orange/80 disabled:bg-gray-400 text-white px-4 py-1 rounded"
                >
                  {loading ? "Running..." : "Run Query"}
                </button>
              </div>
            </div>
            <textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full h-64 font-mono text-sm border rounded p-3 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-navy text-white"
              placeholder="Enter your SPARQL query here..."
              spellCheck={false}
            />
            <div className="mt-2 text-xs text-gray-400">
              Press Ctrl+Enter to run query (not implemented yet)
            </div>
          </div>

          {/* Error Display */}
          {error && (
            <div className="bg-red-900/20 border border-red-400 text-red-400 px-4 py-3 rounded">
              <div className="font-semibold">Query Error</div>
              <div className="text-sm mt-1">{error}</div>
            </div>
          )}

          {/* Results Display */}
          {queryResult && (
            <div className="bg-blue rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-3">
                <div>
                  <h3 className="font-semibold text-white">Results</h3>
                  <div className="text-sm text-gray-400">
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
                      className="text-sm text-green-600 hover:text-white px-3 py-1 border rounded"
                    >
                      Export CSV
                    </button>
                    <button
                      onClick={handleExportJSON}
                      className="text-sm text-orange hover:text-orange/80 px-3 py-1 border rounded"
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
                    <thead className="bg-navy border-b">
                      <tr>
                        <th className="px-3 py-2 text-left text-xs font-medium text-gray-400 uppercase">#</th>
                        {queryResult.variables?.map((variable: string) => (
                          <th key={variable} className="px-3 py-2 text-left text-xs font-medium text-gray-400 uppercase">
                            {variable}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="bg-blue divide-y divide-gray-700">
                      {queryResult.bindings.map((binding: Record<string, unknown>, idx: number) => {
                        const typedBinding = binding as Record<string, { value?: string; type?: string; datatype?: string; "xml:lang"?: string }>;
                        return (
                        <tr key={idx} className="hover:bg-navy">
                          <td className="px-3 py-2 text-gray-400 text-xs">{idx + 1}</td>
                          {queryResult.variables?.map((variable: string) => {
                            const value = typedBinding[variable];
                            return (
                              <td key={variable} className="px-3 py-2">
                                {value ? (
                                   <div className="font-mono text-xs">
                                    <div className="break-all text-white">{value.value}</div>
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
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Empty Results */}
              {queryResult.query_type === "SELECT" && (!queryResult.bindings || queryResult.bindings.length === 0) && (
                <div className="text-center py-8 text-gray-400">
                  No results found
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right Column - Sample Queries & History */}
        <div className="space-y-4">
          {/* Sample Queries */}
          <div className="bg-blue rounded-lg shadow p-4">
            <h3 className="font-semibold mb-3 text-white">Sample Queries</h3>
            <div className="space-y-2">
              {SAMPLE_QUERIES.map((sample, idx) => (
                <button
                  key={idx}
                  onClick={() => handleLoadSample(sample.query)}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  {sample.name}
                </button>
              ))}
            </div>
          </div>

          {/* Query History */}
          {queryHistory.length > 0 && (
            <div className="bg-blue rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-3">
                <h3 className="font-semibold text-white">Query History</h3>
                <button
                  onClick={() => {
                    setQueryHistory([]);
                    localStorage.removeItem("sparql_history");
                  }}
                  className="text-xs text-red-400 hover:text-red-300"
                >
                  Clear
                </button>
              </div>
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {queryHistory.map((historicalQuery, idx) => (
                  <button
                    key={idx}
                    onClick={() => handleLoadSample(historicalQuery)}
                    className="w-full text-left text-xs text-gray-300 hover:bg-navy px-3 py-2 rounded border border-gray-600"
                  >
                    <div className="font-mono truncate">{historicalQuery.split("\n")[0]}</div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Named Graphs */}
          {stats && stats.named_graphs && stats.named_graphs.length > 0 && (
            <div className="bg-blue rounded-lg shadow p-4">
              <h3 className="font-semibold mb-3 text-white">Named Graphs</h3>
              <div className="space-y-1 max-h-64 overflow-y-auto">
                {stats.named_graphs.map((graph, idx) => (
                  <div key={idx} className="text-xs font-mono text-gray-300 break-all px-2 py-1 bg-navy rounded">
                    {graph}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Tips */}
          <div className="bg-blue rounded-lg shadow p-4">
            <h3 className="font-semibold mb-2 text-white">Tips</h3>
            <ul className="text-xs text-gray-400 space-y-1">
              <li>• Use PREFIX to define namespace shortcuts</li>
              <li>• Add LIMIT to prevent large result sets</li>
              <li>• Use OPTIONAL for optional patterns</li>
              <li>• Use FILTER to constrain results</li>
              <li>• Check Named Graphs to query specific ontologies</li>
            </ul>
          </div>
        </div>
        </div>
      )}

      {/* Natural Language Tab Content */}
      {activeTab === "natural-language" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Column - Question Input */}
          <div className="lg:col-span-2 space-y-4">
            {/* Question Input */}
            <div className="bg-blue rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-3">
                <h3 className="font-semibold text-white">Ask a Question</h3>
                <button
                  onClick={handleAskQuestion}
                  disabled={nlLoading}
                  className="bg-orange hover:bg-orange/80 disabled:bg-gray-400 text-white px-4 py-1 rounded"
                >
                  {nlLoading ? "Processing..." : "Ask Question"}
                </button>
              </div>
              
              {/* Ontology Selector */}
              {ontologies.length > 0 && (
                <div className="mb-3">
                  <label className="block text-sm font-medium text-gray-300 mb-1">
                    Select Ontology (Optional)
                  </label>
              <select
                value={selectedOntology}
                onChange={(e) => setSelectedOntology(e.target.value)}
                className="w-full border rounded px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-navy text-white border-gray-600"
              >
                <option value="">All Ontologies</option>
                {ontologies.map((ont) => (
                  <option key={ont.id} value={ont.id}>
                    {ont.name} ({ont.format})
                  </option>
                ))}
              </select>
                </div>
              )}

              <textarea
                value={nlQuestion}
                onChange={(e) => setNlQuestion(e.target.value)}
                className="w-full h-32 border rounded p-3 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-navy text-white border-gray-600"
                placeholder="Ask a question in natural language, e.g., 'Show me all the people in the database' or 'What are the properties of the Person class?'"
                spellCheck={true}
              />
              <div className="mt-2 text-xs text-gray-400">
                The system will translate your question to SPARQL and execute it
              </div>
            </div>

            {/* Error Display */}
            {nlError && (
              <div className="bg-red-900/20 border border-red-400 text-red-400 px-4 py-3 rounded">
                <div className="font-semibold">Error</div>
                <div className="text-sm mt-1">{nlError}</div>
              </div>
            )}

            {/* Results Display */}
            {nlResult && (
              <div className="space-y-4">
                {/* Your Question */}
                <div className="bg-blue rounded-lg shadow p-4">
                  <h3 className="font-semibold mb-2 text-white">Your Question</h3>
                  <p className="text-gray-300">{nlResult.question}</p>
                </div>

                {/* Generated SPARQL */}
                <div className="bg-blue rounded-lg shadow p-4">
                  <h3 className="font-semibold mb-2 text-white">Generated SPARQL Query</h3>
                  <pre className="bg-navy border rounded p-3 text-xs font-mono overflow-x-auto text-white">
                    {nlResult.sparql_query}
                  </pre>
                </div>

                {/* Explanation */}
                {nlResult.explanation && (
                  <div className="bg-blue rounded-lg shadow p-4">
                    <h3 className="font-semibold mb-2 text-white">Explanation</h3>
                    <p className="text-sm text-gray-400">{nlResult.explanation}</p>
                  </div>
                )}

                {/* Query Results */}
                <div className="bg-blue rounded-lg shadow p-4">
                  <div className="flex justify-between items-center mb-3">
                    <div>
                      <h3 className="font-semibold text-white">Results</h3>
                      <div className="text-sm text-gray-400">
                        {nlResult.results.query_type === "SELECT" && (
                          <>
                            {nlResult.results.bindings?.length || 0} row(s) returned
                            {nlResult.results.duration && ` in ${nlResult.results.duration}ms`}
                          </>
                        )}
                        {nlResult.results.query_type === "ASK" && (
                          <>Result: {nlResult.results.boolean ? "true" : "false"}</>
                        )}
                        {nlResult.results.query_type === "CONSTRUCT" && (
                          <>Constructed {nlResult.results.bindings?.length || 0} triple(s)</>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* ASK Result */}
                  {nlResult.results.query_type === "ASK" && (
                    <div className={`text-center py-8 text-2xl font-bold ${nlResult.results.boolean ? "text-green-600" : "text-red-600"}`}>
                      {nlResult.results.boolean ? "TRUE" : "FALSE"}
                    </div>
                  )}

                  {/* SELECT Results */}
                  {nlResult.results.query_type === "SELECT" && nlResult.results.bindings && nlResult.results.bindings.length > 0 && (
                    <div className="overflow-x-auto">
                      <table className="min-w-full text-sm">
                        <thead className="bg-navy border-b">
                          <tr>
                            <th className="px-3 py-2 text-left text-xs font-medium text-gray-400 uppercase">#</th>
                            {nlResult.results.variables?.map((variable: string) => (
                              <th key={variable} className="px-3 py-2 text-left text-xs font-medium text-gray-400 uppercase">
                                {variable}
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody className="bg-blue divide-y divide-gray-700">
                          {nlResult.results.bindings.map((binding: Record<string, unknown>, idx: number) => {
                            const typedBinding = binding as Record<string, { value?: string; type?: string; datatype?: string; "xml:lang"?: string }>;
                            return (
                            <tr key={idx} className="hover:bg-navy">
                              <td className="px-3 py-2 text-gray-400 text-xs">{idx + 1}</td>
                              {nlResult.results.variables?.map((variable: string) => {
                                const value = typedBinding[variable];
                                return (
                                  <td key={variable} className="px-3 py-2">
                                    {value ? (
                                      <div className="font-mono text-xs">
                                        <div className="break-all text-white">{value.value}</div>
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
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Empty Results */}
                  {nlResult.results.query_type === "SELECT" && (!nlResult.results.bindings || nlResult.results.bindings.length === 0) && (
                    <div className="text-center py-8 text-gray-400">
                      No results found
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Right Column - Examples & Tips */}
          <div className="space-y-4">
            {/* Example Questions */}
            <div className="bg-blue rounded-lg shadow p-4">
              <h3 className="font-semibold mb-3 text-white">Example Questions</h3>
              <div className="space-y-2">
                <button
                  onClick={() => setNlQuestion("Show me all the classes in the ontology")}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  Show me all the classes
                </button>
                <button
                  onClick={() => setNlQuestion("What properties does the Person class have?")}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  What properties does Person have?
                </button>
                <button
                  onClick={() => setNlQuestion("List all entities of type Organization")}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  List all Organizations
                </button>
                <button
                  onClick={() => setNlQuestion("Show me the relationships between Person and Organization")}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  Show relationships
                </button>
                <button
                  onClick={() => setNlQuestion("How many triples are in the knowledge graph?")}
                  className="w-full text-left text-sm text-orange hover:text-orange/80 hover:bg-blue px-3 py-2 rounded border border-transparent hover:border-blue-200"
                >
                  Count all triples
                </button>
              </div>
            </div>

            {/* Tips */}
            <div className="bg-blue rounded-lg shadow p-4">
              <h3 className="font-semibold mb-2 text-white">Tips</h3>
              <ul className="text-xs text-gray-400 space-y-1">
                <li>• Be specific about what you want to find</li>
                <li>• Mention class or property names if known</li>
                <li>• Use simple, clear language</li>
                <li>• Review the generated SPARQL to learn</li>
                <li>• Select an ontology for more targeted results</li>
              </ul>
            </div>

            {/* How it Works */}
            <div className="bg-blue rounded-lg shadow p-4">
              <h3 className="font-semibold mb-2 text-white">How it Works</h3>
              <div className="text-xs text-gray-400 space-y-2">
                <p>
                  The system uses an AI model to understand your question and translate it into a SPARQL query.
                </p>
                <p>
                  The generated query is validated for safety (read-only operations only) before execution.
                </p>
                <p>
                  You can see the generated SPARQL to learn how to write queries manually.
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

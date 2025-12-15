"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  uploadOntology,
  validateOntology,
  type OntologyUploadRequest,
} from "@/lib/api";

export default function UploadOntologyPage() {
  const router = useRouter();
  const [formData, setFormData] = useState<OntologyUploadRequest>({
    name: "",
    description: "",
    version: "1.0.0",
    format: "turtle",
    ontology_data: "",
    created_by: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<{
    valid: boolean;
    errors?: unknown[];
    warnings?: unknown[];
  } | null>(null);

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        const content = event.target?.result as string;
        setFormData((prev) => ({ ...prev, ontology_data: content }));

        // Auto-detect format from filename
        const extension = file.name.split(".").pop()?.toLowerCase();
        if (extension === "ttl") {
          setFormData((prev) => ({ ...prev, format: "turtle" }));
        } else if (extension === "rdf" || extension === "xml") {
          setFormData((prev) => ({ ...prev, format: "rdfxml" }));
        } else if (extension === "nt") {
          setFormData((prev) => ({ ...prev, format: "ntriples" }));
        } else if (extension === "jsonld") {
          setFormData((prev) => ({ ...prev, format: "jsonld" }));
        }
      };
      reader.readAsText(file);
    }
  };

  const handleValidate = async () => {
    if (!formData.ontology_data) {
      alert("Please provide ontology data to validate");
      return;
    }

    try {
      setValidating(true);
      setValidationResult(null);
      const response = await validateOntology(formData.ontology_data, formData.format);
      setValidationResult(response.data);
    } catch (err) {
      alert(`Validation failed: ${err instanceof Error ? err.message : "Unknown error"}`);
    } finally {
      setValidating(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!formData.name || !formData.version || !formData.ontology_data) {
      setError("Name, version, and ontology data are required");
      return;
    }

    try {
      setLoading(true);
      await uploadOntology(formData);
      router.push("/ontologies");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to upload ontology");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="mb-6">
        <Link href="/ontologies" className="text-blue-600 hover:underline mb-4 inline-block">
          ← Back to Ontologies
        </Link>
        <h1 className="text-3xl font-bold">Upload Ontology</h1>
        <p className="text-gray-600 mt-1">
          Upload a new ontology to the knowledge graph
        </p>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {validationResult && (
        <div
          className={`border px-4 py-3 rounded mb-4 ${
            validationResult.valid
              ? "bg-green-100 border-green-400 text-green-700"
              : "bg-yellow-100 border-yellow-400 text-yellow-700"
          }`}
        >
          <p className="font-semibold">
            {validationResult.valid ? "✓ Validation Passed" : "⚠ Validation Issues"}
          </p>
          {validationResult.errors && Array.isArray(validationResult.errors) && validationResult.errors.length > 0 && (
            <div className="mt-2">
              <p className="font-medium">Errors:</p>
              <ul className="list-disc ml-5">
                {validationResult.errors.map((error: any, i) => (
                  <li key={i}>{error.message || JSON.stringify(error)}</li>
                ))}
              </ul>
            </div>
          )}
          {validationResult.warnings && Array.isArray(validationResult.warnings) && validationResult.warnings.length > 0 && (
            <div className="mt-2">
              <p className="font-medium">Warnings:</p>
              <ul className="list-disc ml-5">
                {validationResult.warnings.map((warning: any, i) => (
                  <li key={i}>{warning.message || JSON.stringify(warning)}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">
            Name <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className="w-full border rounded px-3 py-2"
            placeholder="my-ontology"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Description</label>
          <textarea
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            className="w-full border rounded px-3 py-2"
            rows={2}
            placeholder="A description of your ontology"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              Version <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.version}
              onChange={(e) => setFormData({ ...formData, version: e.target.value })}
              className="w-full border rounded px-3 py-2"
              placeholder="1.0.0"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              Format <span className="text-red-500">*</span>
            </label>
            <select
              value={formData.format}
              onChange={(e) => setFormData({ ...formData, format: e.target.value })}
              className="w-full border rounded px-3 py-2"
              required
            >
              <option value="turtle">Turtle (.ttl)</option>
              <option value="rdfxml">RDF/XML (.rdf)</option>
              <option value="ntriples">N-Triples (.nt)</option>
              <option value="jsonld">JSON-LD (.jsonld)</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Created By</label>
          <input
            type="text"
            value={formData.created_by}
            onChange={(e) => setFormData({ ...formData, created_by: e.target.value })}
            className="w-full border rounded px-3 py-2"
            placeholder="username"
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">
            Ontology File <span className="text-red-500">*</span>
          </label>
          <input
            type="file"
            onChange={handleFileUpload}
            accept=".ttl,.rdf,.xml,.nt,.jsonld"
            className="w-full border rounded px-3 py-2"
          />
          <p className="text-sm text-gray-500 mt-1">
            Supported formats: Turtle, RDF/XML, N-Triples, JSON-LD
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">
            Ontology Data <span className="text-red-500">*</span>
          </label>
          <textarea
            value={formData.ontology_data}
            onChange={(e) => setFormData({ ...formData, ontology_data: e.target.value })}
            className="w-full border rounded px-3 py-2 font-mono text-sm"
            rows={12}
            placeholder="Paste your ontology data here or upload a file above"
            required
          />
          <div className="flex justify-end mt-2">
            <button
              type="button"
              onClick={handleValidate}
              disabled={validating || !formData.ontology_data}
              className="bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded disabled:opacity-50"
            >
              {validating ? "Validating..." : "Validate Syntax"}
            </button>
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-4">
          <Link
            href="/ontologies"
            className="bg-gray-200 hover:bg-gray-300 px-6 py-2 rounded"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={loading}
            className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded disabled:opacity-50"
          >
            {loading ? "Uploading..." : "Upload Ontology"}
          </button>
        </div>
      </form>
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { getModel, predict, deleteModel, updateModelStatus, type ClassifierModel, type PredictionRequest } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { toast } from "sonner";
import { ArrowLeft, Brain, TrendingUp, Calendar, Play, Pause, Trash2 } from "lucide-react";
import { DynamicPredictionForm } from "@/components/ml/DynamicPredictionForm";
import { ModelPerformanceDashboard } from "@/components/ml/PerformanceDashboard";

export default function ModelDetailPage() {
  const params = useParams();
  const router = useRouter();
  const modelId = params.id as string;

  const [model, setModel] = useState<ClassifierModel | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [predicting, setPredicting] = useState(false);
  const [predictionResult, setPredictionResult] = useState<any | null>(null);
  const [activeTab, setActiveTab] = useState<"overview" | "performance" | "predict">("overview");

  useEffect(() => {
    loadModel();
  }, [modelId]);

  async function loadModel() {
    try {
      setLoading(true);
      setError(null);
      const data = await getModel(modelId);
      setModel(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load model";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete() {
    if (!model) return;
    if (!confirm(`Are you sure you want to delete model "${model.name}"?`)) {
      return;
    }

    try {
      await deleteModel(modelId);
      toast.success(`Model "${model.name}" deleted successfully`);
      router.push("/models");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete model";
      toast.error(message);
    }
  }

  async function handleToggleStatus() {
    if (!model) return;

    try {
      await updateModelStatus(modelId, !model.is_active);
      toast.success(`Model "${model.name}" ${!model.is_active ? 'activated' : 'deactivated'} successfully`);
      loadModel();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to update model status";
      toast.error(message);
    }
  }

  async function handlePredict(inputData: Record<string, number | string>) {
    if (!model) return;

    try {
      setPredicting(true);
      const request: PredictionRequest = {
        data: inputData,
      };

      const result = await predict(modelId, request);
      setPredictionResult(result);
      toast.success("Prediction completed");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Prediction failed";
      toast.error(message);
    } finally {
      setPredicting(false);
    }
  }

  async function handleBatchPredict(inputData: Array<Record<string, number | string>>) {
    if (!model) return;

    try {
      setPredicting(true);
      // For batch predictions, we'll make multiple predict calls
      // In a production system, there should be a dedicated batch endpoint
      const results = await Promise.all(
        inputData.map((data) =>
          predict(modelId, { data })
        )
      );
      setPredictionResult(results);
      toast.success(`Batch prediction completed: ${results.length} predictions`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Batch prediction failed";
      toast.error(message);
    } finally {
      setPredicting(false);
    }
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatAccuracy(accuracy: number): string {
    return `${(accuracy * 100).toFixed(2)}%`;
  }

  if (loading) {
    return (
      <div className="container mx-auto py-8">
        <LoadingSkeleton />
      </div>
    );
  }

  if (error || !model) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="pt-6">
            <p className="text-red-600">Error: {error || "Model not found"}</p>
            <Link href="/models">
              <Button className="mt-4">Back to Models</Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      <div className="mb-6">
        <Link href="/models">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Models
          </Button>
        </Link>
      </div>

      {/* Model Header */}
      <div className="flex justify-between items-start mb-6">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Brain className="h-8 w-8 text-blue-600" />
            <h1 className="text-3xl font-bold">{model.name}</h1>
            <Badge variant={model.is_active ? "default" : "secondary"}>
              {model.is_active ? "Active" : "Inactive"}
            </Badge>
          </div>
          <p className="text-muted-foreground">
            Algorithm: {model.algorithm} • Target: {model.target_class}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={handleToggleStatus}
          >
            {model.is_active ? (
              <>
                <Pause className="h-4 w-4 mr-2" />
                Deactivate
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-2" />
                Activate
              </>
            )}
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Delete
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="mb-6 border-b border-blue">
        <div className="flex gap-4">
          <button
            onClick={() => setActiveTab("overview")}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === "overview"
                ? "border-blue-600 text-orange"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            Overview
          </button>
          <button
            onClick={() => setActiveTab("performance")}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === "performance"
                ? "border-blue-600 text-orange"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            Performance Dashboard
          </button>
          <button
            onClick={() => setActiveTab("predict")}
            className={`px-4 py-2 font-medium border-b-2 transition-colors ${
              activeTab === "predict"
                ? "border-blue-600 text-orange"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            Make Predictions
          </button>
        </div>
      </div>

      {/* Overview Tab */}
      {activeTab === "overview" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Performance Metrics */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-green-600" />
              Performance Metrics
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              <div>
                <p className="text-sm text-muted-foreground">Validation Accuracy</p>
                <p className="text-2xl font-bold text-green-600">
                  {formatAccuracy(model.validate_accuracy)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Precision</p>
                <p className="text-2xl font-bold">
                  {formatAccuracy(model.precision_score)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Recall</p>
                <p className="text-2xl font-bold">
                  {formatAccuracy(model.recall_score)}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">F1 Score</p>
                <p className="text-2xl font-bold">
                  {formatAccuracy(model.f1_score)}
                </p>
              </div>
            </div>

            <div className="mt-6 pt-6 border-t">
              <h4 className="font-medium mb-3">Training Information</h4>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="text-muted-foreground">Training Rows</p>
                  <p className="font-medium">{model.training_rows.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Validation Rows</p>
                  <p className="font-medium">{model.validation_rows.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Model Size</p>
                  <p className="font-medium">
                    {(model.model_size_bytes / 1024 / 1024).toFixed(2)} MB
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <div>
                    <p className="text-muted-foreground">Created</p>
                    <p className="font-medium text-xs">{formatDate(model.created_at)}</p>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Model Details */}
        <Card>
          <CardHeader>
            <CardTitle>Model Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm text-muted-foreground">Model ID</p>
              <p className="text-xs font-mono break-all">{model.id}</p>
            </div>
            {model.ontology_id && (
              <div>
                <p className="text-sm text-muted-foreground">Ontology ID</p>
                <p className="text-xs font-mono break-all">{model.ontology_id}</p>
              </div>
            )}
            <div>
              <p className="text-sm text-muted-foreground">Feature Columns</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {JSON.parse(model.feature_columns || "[]").map((col: string, idx: number) => (
                  <Badge key={idx} variant="outline" className="text-xs">
                    {col}
                  </Badge>
                ))}
              </div>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Class Labels</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {JSON.parse(model.class_labels || "[]").map((label: string, idx: number) => (
                  <Badge key={idx} variant="secondary" className="text-xs">
                    {label}
                  </Badge>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
      )}

      {/* Performance Dashboard Tab */}
      {activeTab === "performance" && model && (
        <ModelPerformanceDashboard model={model} />
      )}

      {/* Prediction Tab */}
      {activeTab === "predict" && model && (
        <DynamicPredictionForm
          featureColumns={JSON.parse(model.feature_columns || "[]")}
          onPredict={handlePredict}
          onBatchPredict={handleBatchPredict}
          predicting={predicting}
          predictionResult={predictionResult}
        />
      )}
    </div>
  );
}

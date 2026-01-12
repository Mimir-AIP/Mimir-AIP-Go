"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { listModels, deleteModel, updateModelStatus, recommendModels, type ClassifierModel, type ModelRecommendation } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { toast } from "sonner";
import { Plus, Brain, TrendingUp, Trash2, Play, Pause, Calendar, Sparkles, Search, Filter } from "lucide-react";

const MODEL_CATEGORIES = [
  { id: "all", name: "All Models", icon: "📊", description: "View all trained models" },
  { id: "anomaly_detection", name: "Anomaly Detection", icon: "🔍", description: "Detect outliers and unusual patterns" },
  { id: "classification", name: "Classification", icon: "🏷️", description: "Categorize data into classes" },
  { id: "clustering", name: "Clustering", icon: "📁", description: "Group similar data points" },
  { id: "regression", name: "Regression", icon: "📈", description: "Predict continuous values" },
  { id: "forecasting", name: "Forecasting", icon: "🔮", description: "Predict future trends" },
];

export default function ModelsPage() {
  const router = useRouter();
  const [models, setModels] = useState<ClassifierModel[]>([]);
  const [recommendations, setRecommendations] = useState<ModelRecommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [loadingRecommendations, setLoadingRecommendations] = useState(false);

  useEffect(() => {
    loadModels();
    loadRecommendations();
  }, []);

  async function loadModels() {
    try {
      setLoading(true);
      setError(null);
      const response = await listModels();
      setModels(response.models || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load models";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function loadRecommendations() {
    try {
      setLoadingRecommendations(true);
      const response = await recommendModels({ use_case: "general" });
      setRecommendations(response.recommendations || []);
    } catch (err) {
      console.log("No recommendations available");
    } finally {
      setLoadingRecommendations(false);
    }
  }

  async function handleDelete(modelId: string, modelName: string) {
    if (!confirm(`Are you sure you want to delete model "${modelName}"?`)) {
      return;
    }

    try {
      await deleteModel(modelId);
      toast.success(`Model "${modelName}" deleted successfully`);
      loadModels();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to delete model";
      toast.error(message);
    }
  }

  async function handleToggleStatus(modelId: string, currentStatus: boolean, modelName: string) {
    try {
      await updateModelStatus(modelId, !currentStatus);
      toast.success(`Model "${modelName}" ${!currentStatus ? 'activated' : 'deactivated'} successfully`);
      loadModels();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to update model status";
      toast.error(message);
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

  function getCategoryInfo(categoryId: string) {
    return MODEL_CATEGORIES.find(c => c.id === categoryId) || MODEL_CATEGORIES[0];
  }

  const filteredModels = selectedCategory === "all"
    ? models
    : models.filter(m => m.algorithm?.toLowerCase().includes(selectedCategory.replace("_", " ")));

  if (loading) {
    return (
      <div className="container mx-auto py-8">
        <LoadingSkeleton />
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="pt-6">
            <p className="text-red-600">Error: {error}</p>
            <Button onClick={loadModels} className="mt-4">
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8 space-y-8">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-orange">ML Models</h1>
          <p className="text-gray-400 mt-2">
            Train, manage, and deploy machine learning models
          </p>
        </div>
        <Link href="/models/train">
          <Button className="bg-orange hover:bg-orange/90 text-navy">
            <Plus className="h-4 w-4 mr-2" />
            Train Model
          </Button>
        </Link>
      </div>

      {/* Model Categories Filter */}
      <Card className="bg-navy border-blue">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-white">
            <Filter className="h-5 w-5 text-orange" />
            Model Categories
          </CardTitle>
          <CardDescription className="text-gray-400">
            Filter models by their primary use case or task type
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {MODEL_CATEGORIES.map((category) => (
              <button
                key={category.id}
                onClick={() => setSelectedCategory(category.id)}
                className={`p-4 rounded-lg text-left transition-all ${
                  selectedCategory === category.id
                    ? "bg-orange/20 border-orange border"
                    : "bg-blue/10 border-transparent border hover:bg-blue/20"
                }`}
              >
                <div className="text-2xl mb-1">{category.icon}</div>
                <div className={`text-sm font-medium ${
                  selectedCategory === category.id ? "text-orange" : "text-white"
                }`}>
                  {category.name}
                </div>
                <div className="text-xs text-gray-400 mt-1 line-clamp-2">
                  {category.description}
                </div>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Model Recommendations */}
      {recommendations.length > 0 && (
        <Card className="bg-navy border-green-500/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-green-400">
              <Sparkles className="h-5 w-5" />
              Recommended Models
            </CardTitle>
            <CardDescription className="text-gray-400">
              AI-suggested models based on your use cases and data characteristics
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {recommendations.slice(0, 6).map((rec, index) => (
                <div key={index} className="bg-blue/10 rounded-lg p-4 border border-blue/30">
                  <div className="flex items-start justify-between mb-2">
                    <div>
                      <h4 className="font-medium text-white">{rec.name}</h4>
                      <p className="text-xs text-gray-400">{rec.algorithm}</p>
                    </div>
                    <Badge className="bg-green-600 text-white">
                      {rec.confidence}% match
                    </Badge>
                  </div>
                  <p className="text-sm text-gray-400 mb-3">{rec.description}</p>
                  <div className="flex flex-wrap gap-1">
                    {(rec.use_cases || []).slice(0, 3).map((useCase) => (
                      <Badge key={useCase} variant="outline" className="text-xs border-blue text-blue-400">
                        {useCase}
                      </Badge>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Models Grid */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <h2 className="text-xl font-semibold text-white">
            {getCategoryInfo(selectedCategory).name}
          </h2>
          <Badge variant="outline" className="border-blue text-blue-400">
            {filteredModels.length} model{filteredModels.length !== 1 ? "s" : ""}
          </Badge>
        </div>

        {filteredModels.length === 0 ? (
          <Card className="bg-navy border-blue">
            <CardContent className="pt-6 text-center">
              <Brain className="h-12 w-12 mx-auto text-gray-400 mb-4" />
              <h3 className="text-lg font-medium text-white mb-2">No Models Found</h3>
              <p className="text-gray-400 mb-4">
                {selectedCategory === "all"
                  ? "Get started by training your first machine learning model"
                  : `No models found in the ${getCategoryInfo(selectedCategory).name} category`}
              </p>
              <Link href="/models/train">
                <Button className="bg-orange hover:bg-orange/90 text-navy">
                  <Plus className="h-4 w-4 mr-2" />
                  Train New Model
                </Button>
              </Link>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredModels.map((model) => (
              <Card
                key={model.id}
                className="bg-navy border-blue cursor-pointer hover:border-orange/50 transition-all"
                onClick={() => router.push(`/models/${model.id}`)}
              >
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <CardTitle className="flex items-center gap-2 text-orange">
                        <Brain className="h-5 w-5" />
                        {model.name}
                      </CardTitle>
                      <CardDescription className="text-gray-400 mt-1">
                        {model.algorithm}
                      </CardDescription>
                    </div>
                    <Badge variant={model.is_active ? "default" : "secondary"} className={model.is_active ? "bg-green-600" : ""}>
                      {model.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center gap-2 text-sm">
                      <TrendingUp className="h-4 w-4 text-green-500" />
                      <span className="text-gray-400">Accuracy:</span>
                      <span className="font-medium text-white">{formatAccuracy(model.validate_accuracy)}</span>
                    </div>

                    <div className="grid grid-cols-3 gap-2 text-sm">
                      <div>
                        <p className="text-gray-500 text-xs">Precision</p>
                        <p className="font-medium text-white">{formatAccuracy(model.precision_score)}</p>
                      </div>
                      <div>
                        <p className="text-gray-500 text-xs">Recall</p>
                        <p className="font-medium text-white">{formatAccuracy(model.recall_score)}</p>
                      </div>
                      <div>
                        <p className="text-gray-500 text-xs">F1 Score</p>
                        <p className="font-medium text-white">{formatAccuracy(model.f1_score)}</p>
                      </div>
                    </div>

                    <div className="text-sm text-gray-500 border-t border-blue/30 pt-3">
                      <div className="flex items-center gap-2">
                        <Calendar className="h-3 w-3" />
                        <span className="text-xs">
                          Trained {formatDate(model.created_at)}
                        </span>
                      </div>
                      <p className="text-xs mt-1">
                        {model.training_rows?.toLocaleString()} training rows
                      </p>
                    </div>

                    <div className="flex gap-2 pt-2" onClick={(e) => e.stopPropagation()}>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleToggleStatus(model.id, model.is_active, model.name)}
                        className="flex-1 border-blue hover:border-orange hover:text-orange"
                      >
                        {model.is_active ? (
                          <>
                            <Pause className="h-3 w-3 mr-1" />
                            Deactivate
                          </>
                        ) : (
                          <>
                            <Play className="h-3 w-3 mr-1" />
                            Activate
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(model.id, model.name)}
                        className="text-red-400 hover:text-red-300 hover:bg-red-500/10 border-red-500/30"
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

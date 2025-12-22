"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { listModels, deleteModel, updateModelStatus, type ClassifierModel } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { toast } from "sonner";
import { Plus, Brain, TrendingUp, Trash2, Play, Pause, Calendar } from "lucide-react";

export default function ModelsPage() {
  const router = useRouter();
  const [models, setModels] = useState<ClassifierModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadModels();
  }, []);

  async function loadModels() {
    try {
      setLoading(true);
      setError(null);
      const response = await listModels();
      // Backend returns null when no models exist, convert to empty array
      setModels(response.models || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load models";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
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
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold">ML Models</h1>
          <p className="text-muted-foreground mt-2">
            Train, manage, and deploy machine learning models
          </p>
        </div>
        <Link href="/models/train">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Train Model
          </Button>
        </Link>
      </div>

      {models.length === 0 ? (
        <Card>
          <CardContent className="pt-6 text-center">
            <Brain className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">No ML Models</h3>
            <p className="text-muted-foreground mb-4">
              Get started by training your first machine learning model
            </p>
            <Link href="/models/train">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Train Your First Model
              </Button>
            </Link>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {models.map((model) => (
            <Card 
              key={model.id} 
              className="cursor-pointer hover:shadow-lg transition-shadow"
              onClick={() => router.push(`/models/${model.id}`)}
            >
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <CardTitle className="flex items-center gap-2">
                      <Brain className="h-5 w-5 text-blue-600" />
                      {model.name}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      {model.algorithm}
                    </CardDescription>
                  </div>
                  <Badge variant={model.is_active ? "default" : "secondary"}>
                    {model.is_active ? "Active" : "Inactive"}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <TrendingUp className="h-4 w-4 text-green-600" />
                    <span className="font-medium">Accuracy:</span>
                    <span>{formatAccuracy(model.validate_accuracy)}</span>
                  </div>
                  
                  <div className="grid grid-cols-3 gap-2 text-sm">
                    <div>
                      <p className="text-muted-foreground text-xs">Precision</p>
                      <p className="font-medium">{formatAccuracy(model.precision_score)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground text-xs">Recall</p>
                      <p className="font-medium">{formatAccuracy(model.recall_score)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground text-xs">F1 Score</p>
                      <p className="font-medium">{formatAccuracy(model.f1_score)}</p>
                    </div>
                  </div>

                  <div className="text-sm text-muted-foreground border-t pt-3">
                    <div className="flex items-center gap-2">
                      <Calendar className="h-3 w-3" />
                      <span className="text-xs">
                        Trained {formatDate(model.created_at)}
                      </span>
                    </div>
                    <p className="text-xs mt-1">
                      {model.training_rows.toLocaleString()} training rows
                    </p>
                  </div>

                  <div className="flex gap-2 pt-2" onClick={(e) => e.stopPropagation()}>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleToggleStatus(model.id, model.is_active, model.name)}
                      className="flex-1"
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
                      className="text-red-600 hover:text-red-700"
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
  );
}

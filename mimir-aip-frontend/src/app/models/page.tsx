"use client";

import { useEffect, useState } from "react";
import { listModels, type ClassifierModel } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Brain, TrendingUp, Calendar } from "lucide-react";

export default function ModelsPage() {
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
      setModels(response.models || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load models";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  function formatAccuracy(accuracy: number): string {
    return `${(accuracy * 100).toFixed(2)}%`;
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">ML Models</h1>
          <p className="text-white/60">Loading models...</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <Card key={i} className="bg-navy border-blue p-6 animate-pulse h-48"></Card>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">ML Models</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>Error: {error}</p>
          <button 
            onClick={loadModels}
            className="mt-4 bg-blue hover:bg-orange text-white px-4 py-2 rounded"
          >
            Retry
          </button>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-orange">ML Models</h1>
          <p className="text-white/60 mt-2">
            Monitor auto-trained model performance. Manage via chat.
          </p>
        </div>
        <button
          onClick={loadModels}
          className="bg-blue hover:bg-orange text-white px-4 py-2 rounded border border-blue"
        >
          Refresh
        </button>
      </div>

      {models.length === 0 ? (
        <Card className="bg-navy border-blue p-12 text-center">
          <Brain className="h-16 w-16 mx-auto text-white/40 mb-4" />
          <h3 className="text-xl font-semibold text-white mb-2">No Models Found</h3>
          <p className="text-white/60 mb-4">
            ML models are automatically trained when you create ontologies from pipelines
          </p>
          <p className="text-sm text-white/40">
            Models will appear here once the autonomous training process completes
          </p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {models.map((model) => (
            <Card key={model.id} className="bg-navy border-blue h-full">
              <div className="p-6">
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-2">
                    <Brain className="h-5 w-5 text-orange" />
                    <h3 className="text-lg font-semibold text-white">{model.name}</h3>
                  </div>
                  <Badge variant={model.is_active ? "default" : "secondary"} className={model.is_active ? "bg-green-600" : ""}>
                    {model.is_active ? "Active" : "Inactive"}
                  </Badge>
                </div>
                
                <p className="text-sm text-white/60 mb-4">{model.algorithm}</p>
                
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <TrendingUp className="h-4 w-4 text-green-400" />
                    <span className="text-white/40">Accuracy:</span>
                    <span className="text-white font-medium">{formatAccuracy(model.validate_accuracy)}</span>
                  </div>
                  
                  <div className="grid grid-cols-3 gap-2 text-sm">
                    <div>
                      <p className="text-white/40 text-xs">Precision</p>
                      <p className="font-medium text-white">{formatAccuracy(model.precision_score)}</p>
                    </div>
                    <div>
                      <p className="text-white/40 text-xs">Recall</p>
                      <p className="font-medium text-white">{formatAccuracy(model.recall_score)}</p>
                    </div>
                    <div>
                      <p className="text-white/40 text-xs">F1 Score</p>
                      <p className="font-medium text-white">{formatAccuracy(model.f1_score)}</p>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2 text-xs text-white/40 pt-3 border-t border-blue/30">
                    <Calendar className="h-3 w-3" />
                    Trained {formatDate(model.created_at)}
                  </div>
                  
                  {model.training_rows && (
                    <p className="text-xs text-white/40">
                      {model.training_rows.toLocaleString()} training rows
                    </p>
                  )}
                </div>
                
                <div className="mt-4 pt-4 border-t border-blue/30">
                  <span className="text-xs text-white/40">Auto-trained from ontology data</span>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
      
      {/* Summary */}
      {models.length > 0 && (
        <div className="mt-6 p-4 bg-blue/20 rounded-lg">
          <p className="text-sm text-white/60">
            Total: {models.length} model{models.length === 1 ? "" : "s"} | 
            Active: {models.filter(m => m.is_active).length} | 
            Auto-trained from ontology data
          </p>
        </div>
      )}
    </div>
  );
}

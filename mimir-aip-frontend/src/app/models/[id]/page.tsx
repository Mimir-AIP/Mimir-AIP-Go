"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getModel, predictWithModel, type ClassifierModel } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Brain, TrendingUp, Calendar, Target, CheckCircle, Info, ChevronDown, ChevronRight, Play, Sparkles } from "lucide-react";

export default function ModelDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [model, setModel] = useState<ClassifierModel | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [expandedFeatures, setExpandedFeatures] = useState(false);
  
  // Prediction state
  const [predictionInputs, setPredictionInputs] = useState<Record<string, number>>({});
  const [predictionResult, setPredictionResult] = useState<any>(null);
  const [predicting, setPredicting] = useState(false);
  const [predictionError, setPredictionError] = useState<string | null>(null);
  const [showPredictionForm, setShowPredictionForm] = useState(false);

  useEffect(() => {
    loadModel();
  }, [id]);

  async function loadModel() {
    try {
      setLoading(true);
      setError(null);
      const modelData = await getModel(id);
      setModel(modelData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load model");
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

  function getAccuracyColor(accuracy: number) {
    if (accuracy >= 0.9) return "text-green-400";
    if (accuracy >= 0.7) return "text-yellow-400";
    return "text-red-400";
  }

  async function handlePredict() {
    if (!model) return;
    
    try {
      setPredicting(true);
      setPredictionError(null);
      const result = await predictWithModel(id, predictionInputs);
      setPredictionResult(result);
    } catch (err) {
      setPredictionError(err instanceof Error ? err.message : "Prediction failed");
    } finally {
      setPredicting(false);
    }
  }

  function updatePredictionInput(feature: string, value: string) {
    setPredictionInputs(prev => ({
      ...prev,
      [feature]: parseFloat(value) || 0
    }));
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/models" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Loading...</h1>
        </div>
        <Card className="bg-navy border-blue p-6 animate-pulse h-96"></Card>
      </div>
    );
  }

  if (error || !model) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/models" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Error</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>{error || "Model not found"}</p>
        </Card>
      </div>
    );
  }

  const featureColumns = JSON.parse(model.feature_columns || "[]");
  const classLabels = JSON.parse(model.class_labels || "[]");
  const confusionMatrix = JSON.parse(model.confusion_matrix || "{}");
  const featureImportance = JSON.parse(model.feature_importance || "{}");
  const hyperparams = JSON.parse(model.hyperparameters || "{}");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/models" className="text-white/60 hover:text-orange transition-colors">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-orange">{model.name}</h1>
          <p className="text-white/60 text-sm">Algorithm: <span className="text-blue capitalize">{model.algorithm}</span></p>
        </div>
        <Badge className={model.is_active ? "bg-green-500/20 text-green-400 border-green-500" : "bg-gray-500/20 text-gray-400 border-gray-500"}>
          {model.is_active ? "Active" : "Inactive"}
        </Badge>
      </div>

      {/* Performance Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <CheckCircle className="h-5 w-5 text-green-400" />
            <div>
              <p className="text-white/60 text-xs">Validation Accuracy</p>
              <p className={`text-xl font-bold ${getAccuracyColor(model.validate_accuracy)}`}>
                {formatAccuracy(model.validate_accuracy)}
              </p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Brain className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Training Accuracy</p>
              <p className={`text-xl font-bold ${getAccuracyColor(model.train_accuracy)}`}>
                {formatAccuracy(model.train_accuracy)}
              </p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <TrendingUp className="h-5 w-5 text-orange" />
            <div>
              <p className="text-white/60 text-xs">F1 Score</p>
              <p className="text-white text-xl font-bold">{formatAccuracy(model.f1_score)}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Created</p>
              <p className="text-white text-sm font-semibold">{formatDate(model.created_at)}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Target & Classes */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3 mb-3">
            <Target className="h-5 w-5 text-orange" />
            <h3 className="text-white font-semibold">Target Class</h3>
          </div>
          <p className="text-white/60 text-sm">{model.target_class}</p>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3 mb-3">
            <Brain className="h-5 w-5 text-blue" />
            <h3 className="text-white font-semibold">Classes ({classLabels.length})</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {classLabels.map((label: string) => (
              <Badge key={label} variant="outline" className="text-xs">
                {label}
              </Badge>
            ))}
          </div>
        </Card>
      </div>

      {/* Feature Importance */}
      {featureColumns.length > 0 && (
        <Card className="bg-navy border-blue">
          <button
            onClick={() => setExpandedFeatures(!expandedFeatures)}
            className="w-full p-4 flex items-center justify-between hover:bg-blue/10 transition-colors"
          >
            <div className="flex items-center gap-3">
              <TrendingUp className="h-5 w-5 text-blue" />
              <h3 className="text-white font-semibold">Feature Importance ({featureColumns.length} features)</h3>
            </div>
            {expandedFeatures ? (
              <ChevronDown className="h-5 w-5 text-orange" />
            ) : (
              <ChevronRight className="h-5 w-5 text-white/60" />
            )}
          </button>
          {expandedFeatures && (
            <div className="px-4 pb-4 divide-y divide-blue/30">
              {featureColumns.map((feature: string) => {
                const importance = featureImportance[feature] || 0;
                return (
                  <div key={feature} className="py-3 flex items-center justify-between">
                    <span className="text-white text-sm">{feature}</span>
                    <div className="flex items-center gap-3">
                      <div className="w-24 h-2 bg-blue/20 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-orange"
                          style={{ width: `${importance * 100}%` }}
                        />
                      </div>
                      <span className="text-white/60 text-xs w-12 text-right">
                        {(importance * 100).toFixed(1)}%
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      )}

      {/* Confusion Matrix (if available) */}
      {Object.keys(confusionMatrix).length > 0 && (
        <Card className="bg-navy border-blue p-4">
          <h3 className="text-white font-semibold mb-4">Confusion Matrix</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="text-left text-white/60 p-2"></th>
                  {classLabels.map((label: string) => (
                    <th key={label} className="text-center text-white/60 p-2">{label}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {classLabels.map((actual: string) => (
                  <tr key={actual}>
                    <td className="text-white/60 p-2 font-medium">{actual}</td>
                    {classLabels.map((predicted: string) => {
                      const value = confusionMatrix[actual]?.[predicted] || 0;
                      const isDiagonal = actual === predicted;
                      return (
                        <td
                          key={`${actual}-${predicted}`}
                          className={`text-center p-2 ${isDiagonal ? "text-green-400 font-bold" : "text-white/60"}`}
                        >
                          {value}
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {/* Read More Button */}
      <button
        onClick={() => setShowDetails(!showDetails)}
        className="w-full py-3 bg-blue/20 hover:bg-blue/30 border border-blue text-blue hover:text-white rounded transition-colors flex items-center justify-center gap-2"
      >
        <Info className="h-4 w-4" />
        {showDetails ? "Hide Advanced Details" : "Read More - Advanced Details"}
      </button>

      {/* Advanced Details */}
      {showDetails && (
        <Card className="bg-navy border-blue p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Advanced Details</h3>
          <div className="space-y-3 text-sm">
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Model ID:</span>
              <span className="text-white col-span-2 font-mono">{model.id}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Ontology ID:</span>
              <span className="text-white col-span-2 font-mono">{model.ontology_id}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Training Rows:</span>
              <span className="text-white col-span-2">{model.training_rows}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Validation Rows:</span>
              <span className="text-white col-span-2">{model.validation_rows}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Model Size:</span>
              <span className="text-white col-span-2">{(model.model_size_bytes / 1024).toFixed(2)} KB</span>
            </div>
            <div className="grid grid-cols-3 gap-4 pt-2 border-t border-blue/30">
              <span className="text-white/60">Precision:</span>
              <span className="text-white col-span-2">{formatAccuracy(model.precision_score)}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Recall:</span>
              <span className="text-white col-span-2">{formatAccuracy(model.recall_score)}</span>
            </div>
            {hyperparams.max_depth && (
              <div className="grid grid-cols-3 gap-4 pt-2 border-t border-blue/30">
                <span className="text-white/60">Max Depth:</span>
                <span className="text-white col-span-2">{hyperparams.max_depth}</span>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Simple Prediction Interface */}
      <Card className="bg-gradient-to-br from-navy to-navy/80 border-orange/50 p-6">
        <div className="flex items-center gap-3 mb-4">
          <Sparkles className="h-6 w-6 text-orange" />
          <h2 className="text-xl font-bold text-white">Test Predictions</h2>
          <span className="text-xs text-white/40">Simple interface for non-technical users</span>
        </div>
        
        <p className="text-white/60 text-sm mb-4">
          Enter values below to see what the model predicts. No technical knowledge required!
        </p>

        <button
          onClick={() => setShowPredictionForm(!showPredictionForm)}
          className="w-full py-2 bg-orange/20 hover:bg-orange/30 border border-orange/50 text-orange rounded transition-colors flex items-center justify-center gap-2 mb-4"
        >
          <Play className="h-4 w-4" />
          {showPredictionForm ? "Hide Prediction Form" : "Show Prediction Form"}
        </button>

        {showPredictionForm && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {featureColumns.map((feature: string) => (
                <div key={feature} className="space-y-1">
                  <label className="text-xs text-white/60">{feature}</label>
                  <Input
                    type="number"
                    placeholder="0.0"
                    className="bg-blue/20 border-blue text-white"
                    onChange={(e) => updatePredictionInput(feature, e.target.value)}
                  />
                </div>
              ))}
            </div>

            <Button
              onClick={handlePredict}
              disabled={predicting}
              className="w-full bg-orange hover:bg-orange/80 text-white"
            >
              {predicting ? "Predicting..." : "Get Prediction"}
            </Button>

            {predictionError && (
              <div className="p-3 bg-red-500/20 border border-red-500 rounded text-red-400 text-sm">
                {predictionError}
              </div>
            )}

            {predictionResult && (
              <div className="p-4 bg-green-500/10 border border-green-500/50 rounded-lg">
                <h4 className="text-sm font-semibold text-green-400 mb-2">Prediction Result</h4>
                <div className="space-y-2">
                  {predictionResult.predicted_class && (
                    <div className="flex justify-between items-center">
                      <span className="text-white/60">Predicted Class:</span>
                      <Badge className="bg-orange text-white">{predictionResult.predicted_class}</Badge>
                    </div>
                  )}
                  {predictionResult.confidence && (
                    <div className="flex justify-between items-center">
                      <span className="text-white/60">Confidence:</span>
                      <span className="text-green-400 font-bold">{(predictionResult.confidence * 100).toFixed(1)}%</span>
                    </div>
                  )}
                  {predictionResult.predicted_value !== undefined && (
                    <div className="flex justify-between items-center">
                      <span className="text-white/60">Predicted Value:</span>
                      <span className="text-orange font-bold">{predictionResult.predicted_value.toFixed(2)}</span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </Card>

      {/* Actions */}
      <div className="flex gap-4">
        <Link
          href={`/chat?model_id=${id}`}
          className="flex-1 py-3 bg-blue/20 hover:bg-blue/30 text-blue border border-blue rounded text-center transition-colors"
        >
          Chat About This Model
        </Link>
      </div>
    </div>
  );
}

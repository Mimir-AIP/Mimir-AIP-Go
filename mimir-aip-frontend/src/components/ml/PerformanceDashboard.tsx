"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import {
  TrendingUp,
  Target,
  Activity,
  BarChart3,
  CheckCircle2,
  AlertCircle,
} from "lucide-react";

interface ModelPerformanceProps {
  model: {
    id: string;
    name: string;
    algorithm: string;
    train_accuracy: number;
    validate_accuracy: number;
    precision_score: number;
    recall_score: number;
    f1_score: number;
    confusion_matrix: string;
    feature_importance: string;
    training_rows: number;
    validation_rows: number;
  };
}

interface ConfusionMatrix {
  labels: string[];
  matrix: number[][];
}

interface FeatureImportance {
  feature: string;
  importance: number;
}

export function ModelPerformanceDashboard({ model }: ModelPerformanceProps) {
  // Parse confusion matrix
  let confusionMatrix: ConfusionMatrix | null = null;
  if (model.confusion_matrix) {
    try {
      confusionMatrix = JSON.parse(model.confusion_matrix);
    } catch (e) {
      console.error("Failed to parse confusion matrix:", e);
    }
  }

  // Parse feature importance
  let featureImportance: FeatureImportance[] = [];
  if (model.feature_importance) {
    try {
      const parsed = JSON.parse(model.feature_importance);
      featureImportance = Object.entries(parsed)
        .map(([feature, importance]) => ({
          feature,
          importance: importance as number,
        }))
        .sort((a, b) => b.importance - a.importance)
        .slice(0, 10); // Top 10 features
    } catch (e) {
      console.error("Failed to parse feature importance:", e);
    }
  }

  const getScoreColor = (score: number) => {
    if (score >= 0.8) return "text-green-500";
    if (score >= 0.6) return "text-yellow-500";
    return "text-red-500";
  };

  const getScoreBackground = (score: number) => {
    if (score >= 0.8) return "bg-green-500/20 border-green-500/30";
    if (score >= 0.6) return "bg-yellow-500/20 border-yellow-500/30";
    return "bg-red-500/20 border-red-500/30";
  };

  return (
    <div className="space-y-6">
      {/* Performance Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-white/60 flex items-center gap-2">
              <Target className="h-4 w-4 text-orange" />
              Accuracy
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-white/60">Training</span>
                  <span className={`text-lg font-bold ${getScoreColor(model.train_accuracy)}`}>
                    {(model.train_accuracy * 100).toFixed(1)}%
                  </span>
                </div>
                <Progress value={model.train_accuracy * 100} className="h-2 mt-1" />
              </div>
              <div>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-white/60">Validation</span>
                  <span className={`text-lg font-bold ${getScoreColor(model.validate_accuracy)}`}>
                    {(model.validate_accuracy * 100).toFixed(1)}%
                  </span>
                </div>
                <Progress value={model.validate_accuracy * 100} className="h-2 mt-1" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-white/60 flex items-center gap-2">
              <Activity className="h-4 w-4 text-orange" />
              Precision
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className={`text-3xl font-bold ${getScoreColor(model.precision_score)}`}>
              {(model.precision_score * 100).toFixed(1)}%
            </div>
            <Progress value={model.precision_score * 100} className="h-2 mt-2" />
            <p className="text-xs text-white/60 mt-2">
              True positives / (True positives + False positives)
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-white/60 flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-orange" />
              Recall
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className={`text-3xl font-bold ${getScoreColor(model.recall_score)}`}>
              {(model.recall_score * 100).toFixed(1)}%
            </div>
            <Progress value={model.recall_score * 100} className="h-2 mt-2" />
            <p className="text-xs text-white/60 mt-2">
              True positives / (True positives + False negatives)
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-white/60 flex items-center gap-2">
              <BarChart3 className="h-4 w-4 text-orange" />
              F1 Score
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className={`text-3xl font-bold ${getScoreColor(model.f1_score)}`}>
              {(model.f1_score * 100).toFixed(1)}%
            </div>
            <Progress value={model.f1_score * 100} className="h-2 mt-2" />
            <p className="text-xs text-white/60 mt-2">
              Harmonic mean of precision and recall
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Dataset Statistics */}
      <Card>
        <CardHeader>
          <CardTitle>Dataset Statistics</CardTitle>
          <CardDescription>Training and validation dataset information</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-2">
              <div className="flex items-center justify-between p-3 rounded-lg bg-blue/20 border border-blue/30">
                <span className="text-white/80">Training Samples</span>
                <span className="text-lg font-bold text-white">{model.training_rows.toLocaleString()}</span>
              </div>
              <div className="flex items-center justify-between p-3 rounded-lg bg-orange/20 border border-orange/30">
                <span className="text-white/80">Validation Samples</span>
                <span className="text-lg font-bold text-white">{model.validation_rows.toLocaleString()}</span>
              </div>
            </div>
            <div className="flex items-center justify-center">
              <div className="text-center">
                <div className="text-4xl font-bold text-white">
                  {model.training_rows + model.validation_rows > 0
                    ? ((model.training_rows / (model.training_rows + model.validation_rows)) * 100).toFixed(0)
                    : 0}
                  %
                </div>
                <div className="text-sm text-white/60 mt-1">Training Split</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Confusion Matrix */}
      {confusionMatrix && (
        <Card>
          <CardHeader>
            <CardTitle>Confusion Matrix</CardTitle>
            <CardDescription>Model prediction accuracy breakdown by class</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr>
                    <th className="border border-blue/30 p-2 bg-navy"></th>
                    {confusionMatrix.labels.map((label, idx) => (
                      <th key={idx} className="border border-blue/30 p-2 bg-navy text-sm font-medium text-orange">
                        Predicted: {label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {confusionMatrix.matrix.map((row, i) => (
                    <tr key={i}>
                      <td className="border border-blue/30 p-2 bg-navy text-sm font-medium text-orange">
                        Actual: {confusionMatrix.labels[i]}
                      </td>
                      {row.map((value, j) => {
                        const isCorrect = i === j;
                        const total = row.reduce((sum, v) => sum + v, 0);
                        const percentage = total > 0 ? (value / total) * 100 : 0;
                        
                        return (
                          <td
                            key={j}
                            className={`border border-blue/30 p-3 text-center ${
                              isCorrect ? "bg-green-500/20" : "bg-red-500/10"
                            }`}
                          >
                            <div className="text-lg font-bold text-white">{value}</div>
                            <div className="text-xs text-white/60">{percentage.toFixed(1)}%</div>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="mt-4 p-3 bg-blue/10 rounded-lg border border-blue/20">
              <div className="flex items-start gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5" />
                <div className="text-sm text-white/80">
                  <strong>Diagonal values</strong> (green) represent correct predictions. Higher values indicate better performance.
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Feature Importance */}
      {featureImportance.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Feature Importance</CardTitle>
            <CardDescription>Top features contributing to model predictions</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {featureImportance.map((item, idx) => {
                const maxImportance = featureImportance[0]?.importance || 1;
                const percentage = (item.importance / maxImportance) * 100;
                
                return (
                  <div key={idx} className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-white">{item.feature}</span>
                      <span className="text-sm text-white/60">{item.importance.toFixed(4)}</span>
                    </div>
                    <div className="h-2 bg-blue/20 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-gradient-to-r from-orange to-orange/60 rounded-full transition-all"
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="mt-4 p-3 bg-blue/10 rounded-lg border border-blue/20">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-4 w-4 text-orange mt-0.5" />
                <div className="text-sm text-white/80">
                  Feature importance shows which input features have the most influence on the model's predictions.
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Model Health Indicators */}
      <Card>
        <CardHeader>
          <CardTitle>Model Health Check</CardTitle>
          <CardDescription>Key indicators of model quality</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex items-center gap-3 p-3 rounded-lg border border-blue/30">
              {Math.abs(model.train_accuracy - model.validate_accuracy) < 0.1 ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertCircle className="h-5 w-5 text-yellow-500" />
              )}
              <div className="flex-1">
                <div className="font-medium text-white">Overfitting Check</div>
                <div className="text-sm text-white/60">
                  Training vs Validation gap: {(Math.abs(model.train_accuracy - model.validate_accuracy) * 100).toFixed(1)}%
                  {Math.abs(model.train_accuracy - model.validate_accuracy) < 0.1 
                    ? " (Good - Low overfitting)" 
                    : " (Warning - May be overfitting)"}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3 p-3 rounded-lg border border-blue/30">
              {model.f1_score >= 0.7 ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertCircle className="h-5 w-5 text-yellow-500" />
              )}
              <div className="flex-1">
                <div className="font-medium text-white">F1 Score Assessment</div>
                <div className="text-sm text-white/60">
                  {model.f1_score >= 0.8 && "Excellent - Strong performance"}
                  {model.f1_score >= 0.7 && model.f1_score < 0.8 && "Good - Acceptable performance"}
                  {model.f1_score >= 0.5 && model.f1_score < 0.7 && "Fair - Room for improvement"}
                  {model.f1_score < 0.5 && "Poor - Consider retraining"}
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3 p-3 rounded-lg border border-blue/30">
              {model.training_rows >= 100 ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertCircle className="h-5 w-5 text-yellow-500" />
              )}
              <div className="flex-1">
                <div className="font-medium text-white">Dataset Size</div>
                <div className="text-sm text-white/60">
                  {model.training_rows >= 1000 && "Large dataset - Good for training"}
                  {model.training_rows >= 100 && model.training_rows < 1000 && "Moderate dataset - Adequate for simple models"}
                  {model.training_rows < 100 && "Small dataset - Consider collecting more data"}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

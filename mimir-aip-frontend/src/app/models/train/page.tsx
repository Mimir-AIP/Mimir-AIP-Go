"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { trainModel, autoTrainWithData, type TrainModelRequest, type AutoTrainWithDataRequest } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ArrowLeft, Brain, Upload } from "lucide-react";
import Link from "next/link";

export default function TrainModelPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [modelName, setModelName] = useState("");
  const [targetColumn, setTargetColumn] = useState("");
  const [algorithm, setAlgorithm] = useState("random_forest");
  const [csvFile, setCsvFile] = useState<File | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    
    if (!modelName || !targetColumn) {
      toast.error("Please fill in all required fields");
      return;
    }

    if (!csvFile) {
      toast.error("Please upload a CSV file");
      return;
    }

    try {
      setLoading(true);

      // Read CSV file as text
      const csvText = await csvFile.text();
      
      // Use auto-train API which accepts CSV data directly
      const request: AutoTrainWithDataRequest = {
        data: csvText,
        target_column: targetColumn,
        model_name: modelName,
        algorithm: algorithm,
        test_split: 0.2,
      };

      const response = await autoTrainWithData(request);
      
      toast.success(`Model "${modelName}" trained successfully!`);
      router.push(`/models/${response.model_id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to train model";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) {
      if (!file.name.endsWith('.csv')) {
        toast.error("Please upload a CSV file");
        return;
      }
      setCsvFile(file);
    }
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

      <Card className="max-w-2xl mx-auto">
        <CardHeader>
          <div className="flex items-center gap-2">
            <Brain className="h-6 w-6 text-blue-600" />
            <CardTitle>Train New ML Model</CardTitle>
          </div>
          <CardDescription>
            Upload training data and configure your machine learning model
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Model Name */}
            <div className="space-y-2">
              <Label htmlFor="model-name">Model Name *</Label>
              <Input
                id="model-name"
                name="model_name"
                placeholder="Enter model name (e.g., Product Price Predictor)"
                value={modelName}
                onChange={(e) => setModelName(e.target.value)}
                required
              />
            </div>

            {/* CSV File Upload */}
            <div className="space-y-2">
              <Label htmlFor="csv-file">Training Data (CSV) *</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="csv-file"
                  type="file"
                  accept=".csv"
                  onChange={handleFileChange}
                  required
                  className="cursor-pointer"
                />
                <Upload className="h-4 w-4 text-muted-foreground" />
              </div>
              {csvFile && (
                <p className="text-sm text-muted-foreground">
                  Selected: {csvFile.name} ({(csvFile.size / 1024).toFixed(2)} KB)
                </p>
              )}
            </div>

            {/* Target Column */}
            <div className="space-y-2">
              <Label htmlFor="target-column">Target Column *</Label>
              <Input
                id="target-column"
                name="target_column"
                placeholder="Enter target column name (e.g., price, category)"
                value={targetColumn}
                onChange={(e) => setTargetColumn(e.target.value)}
                required
              />
              <p className="text-xs text-muted-foreground">
                The column you want to predict
              </p>
            </div>

            {/* Algorithm Selection */}
            <div className="space-y-2">
              <Label htmlFor="algorithm">Algorithm</Label>
              <select
                id="algorithm"
                name="algorithm"
                value={algorithm}
                onChange={(e) => setAlgorithm(e.target.value)}
                className="w-full h-10 px-3 py-2 text-sm bg-background border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
              >
                <option value="random_forest">Random Forest</option>
                <option value="logistic_regression">Logistic Regression</option>
                <option value="decision_tree">Decision Tree</option>
                <option value="svm">Support Vector Machine (SVM)</option>
                <option value="naive_bayes">Naive Bayes</option>
                <option value="knn">K-Nearest Neighbors (KNN)</option>
              </select>
              <p className="text-xs text-muted-foreground">
                The algorithm will be automatically selected if not specified
              </p>
            </div>

            {/* Submit Button */}
            <div className="flex gap-3 pt-4">
              <Button
                type="submit"
                disabled={loading}
                className="flex-1"
              >
                {loading ? (
                  <>Training...</>
                ) : (
                  <>
                    <Brain className="h-4 w-4 mr-2" />
                    Start Training
                  </>
                )}
              </Button>
              <Link href="/models" className="flex-1">
                <Button type="button" variant="outline" className="w-full">
                  Cancel
                </Button>
              </Link>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

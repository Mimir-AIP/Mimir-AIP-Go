"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { Upload, Sparkles, FileJson } from "lucide-react";
import { toast } from "sonner";

interface DynamicPredictionFormProps {
  featureColumns: string[];
  onPredict: (data: Record<string, number | string>) => Promise<void>;
  onBatchPredict?: (data: Array<Record<string, number | string>>) => Promise<void>;
  predicting: boolean;
  predictionResult: any | null;
}

export function DynamicPredictionForm({
  featureColumns,
  onPredict,
  onBatchPredict,
  predicting,
  predictionResult,
}: DynamicPredictionFormProps) {
  const [formData, setFormData] = useState<Record<string, string>>({});
  const [jsonInput, setJsonInput] = useState("");
  const [batchJsonInput, setBatchJsonInput] = useState("");
  const [activeTab, setActiveTab] = useState<"form" | "json" | "batch">("form");

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Convert form data to appropriate types
    const data: Record<string, number | string> = {};
    for (const [key, value] of Object.entries(formData)) {
      // Try to parse as number, otherwise keep as string
      const numValue = parseFloat(value);
      data[key] = isNaN(numValue) ? value : numValue;
    }

    await onPredict(data);
  };

  const handleJsonSubmit = async () => {
    if (!jsonInput.trim()) {
      toast.error("Please enter JSON data");
      return;
    }

    try {
      const data = JSON.parse(jsonInput);
      await onPredict(data);
    } catch (err) {
      toast.error("Invalid JSON format");
    }
  };

  const handleBatchSubmit = async () => {
    if (!batchJsonInput.trim()) {
      toast.error("Please enter JSON array data");
      return;
    }

    if (!onBatchPredict) {
      toast.error("Batch prediction is not available for this model");
      return;
    }

    try {
      const data = JSON.parse(batchJsonInput);
      if (!Array.isArray(data)) {
        toast.error("Batch input must be a JSON array");
        return;
      }
      await onBatchPredict(data);
    } catch (err) {
      toast.error("Invalid JSON format");
    }
  };

  const generateSampleJson = () => {
    const sample: Record<string, number> = {};
    featureColumns.forEach((col, idx) => {
      sample[col] = idx * 10; // Just sample values
    });
    setJsonInput(JSON.stringify(sample, null, 2));
  };

  const generateBatchSample = () => {
    const samples = Array(3)
      .fill(0)
      .map((_, batchIdx) => {
        const sample: Record<string, number> = {};
        featureColumns.forEach((col, idx) => {
          sample[col] = (idx + batchIdx) * 10;
        });
        return sample;
      });
    setBatchJsonInput(JSON.stringify(samples, null, 2));
  };

  return (
    <Card className="lg:col-span-3">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Sparkles className="h-5 w-5" />
          Make Predictions
        </CardTitle>
        <CardDescription>
          Enter feature values to get predictions from this model
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as typeof activeTab)}>
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="form">Dynamic Form</TabsTrigger>
            <TabsTrigger value="json">JSON Input</TabsTrigger>
            {onBatchPredict && <TabsTrigger value="batch">Batch Predict</TabsTrigger>}
          </TabsList>

          {/* Dynamic Form Tab */}
          <TabsContent value="form" className="space-y-4">
            <form onSubmit={handleFormSubmit} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {featureColumns.map((feature) => (
                  <div key={feature} className="space-y-2">
                    <Label htmlFor={feature}>{feature}</Label>
                    <Input
                      id={feature}
                      name={feature}
                      type="text"
                      placeholder={`Enter ${feature}`}
                      value={formData[feature] || ""}
                      onChange={(e) =>
                        setFormData((prev) => ({ ...prev, [feature]: e.target.value }))
                      }
                      required
                    />
                  </div>
                ))}
              </div>
              <Button type="submit" disabled={predicting} className="w-full md:w-auto">
                {predicting ? "Predicting..." : "Run Prediction"}
              </Button>
            </form>
          </TabsContent>

          {/* JSON Input Tab */}
          <TabsContent value="json" className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <Label htmlFor="json-input">Input Data (JSON)</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={generateSampleJson}
                >
                  <FileJson className="h-4 w-4 mr-2" />
                  Generate Sample
                </Button>
              </div>
              <Textarea
                id="json-input"
                name="json_input"
                placeholder={`{\n  "feature1": 10,\n  "feature2": 20\n}`}
                value={jsonInput}
                onChange={(e) => setJsonInput(e.target.value)}
                rows={8}
                className="font-mono text-sm"
              />
            </div>
            <Button onClick={handleJsonSubmit} disabled={predicting}>
              {predicting ? "Predicting..." : "Run Prediction"}
            </Button>
          </TabsContent>

          {/* Batch Prediction Tab */}
          {onBatchPredict && (
            <TabsContent value="batch" className="space-y-4">
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <Label htmlFor="batch-input">Batch Input Data (JSON Array)</Label>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={generateBatchSample}
                  >
                    <FileJson className="h-4 w-4 mr-2" />
                    Generate Sample
                  </Button>
                </div>
                <Textarea
                  id="batch-input"
                  name="batch_input"
                  placeholder={`[\n  {"feature1": 10, "feature2": 20},\n  {"feature1": 15, "feature2": 25}\n]`}
                  value={batchJsonInput}
                  onChange={(e) => setBatchJsonInput(e.target.value)}
                  rows={10}
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  Provide an array of objects for batch prediction. Each object should contain
                  all required features.
                </p>
              </div>
              <Button onClick={handleBatchSubmit} disabled={predicting}>
                {predicting ? "Predicting..." : "Run Batch Prediction"}
              </Button>
            </TabsContent>
          )}
        </Tabs>

        {/* Prediction Result */}
        {predictionResult && (
          <div className="mt-6 p-4 bg-muted rounded-md">
            <h4 className="font-medium mb-2 flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-green-600" />
              Prediction Result:
            </h4>
            
            {/* Pretty format for single prediction */}
            {!Array.isArray(predictionResult) && predictionResult.prediction !== undefined && (
              <div className="space-y-2">
                <div className="text-2xl font-bold text-green-600">
                  Prediction: {predictionResult.prediction}
                </div>
                {predictionResult.confidence && (
                  <div className="text-sm text-muted-foreground">
                    Confidence: {(predictionResult.confidence * 100).toFixed(2)}%
                  </div>
                )}
                {predictionResult.probabilities && (
                  <div className="mt-2">
                    <p className="text-sm font-medium mb-1">Class Probabilities:</p>
                    <div className="space-y-1">
                      {Object.entries(predictionResult.probabilities).map(([cls, prob]: [string, any]) => (
                        <div key={cls} className="flex items-center gap-2">
                          <span className="text-xs w-24">{cls}:</span>
                          <div className="flex-1 bg-gray-200 rounded-full h-2">
                            <div
                              className="bg-blue-600 h-2 rounded-full"
                              style={{ width: `${(prob as number) * 100}%` }}
                            />
                          </div>
                          <span className="text-xs w-12 text-right">
                            {((prob as number) * 100).toFixed(1)}%
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Table format for batch predictions */}
            {Array.isArray(predictionResult) && (
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-2 px-3">#</th>
                      <th className="text-left py-2 px-3">Prediction</th>
                      <th className="text-left py-2 px-3">Confidence</th>
                    </tr>
                  </thead>
                  <tbody>
                    {predictionResult.map((result: any, idx: number) => (
                      <tr key={idx} className="border-b">
                        <td className="py-2 px-3">{idx + 1}</td>
                        <td className="py-2 px-3 font-medium">{result.prediction}</td>
                        <td className="py-2 px-3">
                          {result.confidence
                            ? `${(result.confidence * 100).toFixed(2)}%`
                            : "-"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Raw JSON fallback */}
            <details className="mt-4">
              <summary className="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
                View Raw JSON
              </summary>
              <pre className="mt-2 text-xs font-mono overflow-auto bg-background p-2 rounded border">
                {JSON.stringify(predictionResult, null, 2)}
              </pre>
            </details>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

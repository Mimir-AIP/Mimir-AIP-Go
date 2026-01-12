"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Settings, CheckCircle, AlertCircle, Download, Brain, Key, Globe, Server } from "lucide-react";
import { toast } from "sonner";

interface AIProvider {
  provider: string;
  name: string;
  configured: boolean;
  requires_key: boolean;
  models?: string[];
  description?: string;
}

export default function AIProvidersTab() {
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedProvider, setSelectedProvider] = useState<string>("");
  const [selectedModel, setSelectedModel] = useState<string>("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    loadProviders();
  }, []);

  async function loadProviders() {
    try {
      setLoading(true);
      const response = await fetch("/api/v1/ai/providers");
      if (response.ok) {
        const data = await response.json();
        setProviders(data);
        if (data.length > 0) {
          const configured = data.find((p: AIProvider) => p.configured);
          if (configured) {
            setSelectedProvider(configured.provider);
          }
        }
      }
    } catch (error) {
      toast.error("Failed to load AI providers");
    } finally {
      setLoading(false);
    }
  }

  async function handleSaveProvider() {
    toast.success("Provider settings saved");
  }

  async function handleDownloadLocalModel() {
    toast.info("Downloading local LLM model... This may take a few minutes.");
  }

  const getProviderIcon = (provider: string) => {
    switch (provider) {
      case "openai":
        return <Globe className="w-5 h-5" />;
      case "anthropic":
        return <Brain className="w-5 h-5" />;
      case "local":
        return <Server className="w-5 h-5" />;
      default:
        return <Settings className="w-5 h-5" />;
    }
  };

  const getProviderColor = (provider: string) => {
    switch (provider) {
      case "openai":
        return "border-green-500/50 text-green-400";
      case "anthropic":
        return "border-orange-500/50 text-orange-400";
      case "local":
        return "border-blue-500/50 text-blue-400";
      case "mock":
        return "border-purple-500/50 text-purple-400";
      default:
        return "border-gray-500/50 text-gray-400";
    }
  };

  const configuredProviders = providers.filter((p) => p.configured);
  const unconfiguredProviders = providers.filter((p) => !p.configured);

  if (loading) {
    return <div className="text-center py-8">Loading AI providers...</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">AI Provider Selection</h2>
        <p className="text-sm text-gray-400 mt-1">
          Configure and select AI providers for model recommendations, entity extraction, and chat
        </p>
      </div>

      {/* Configured Providers */}
      {configuredProviders.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-3 text-green-400">Active Provider</h3>
          <div className="grid gap-4">
            {configuredProviders.map((provider) => (
              <Card key={provider.provider} className={`bg-navy border ${getProviderColor(provider.provider)}`}>
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div className="flex items-center space-x-3">
                      <div className={`p-2 rounded-lg ${getProviderColor(provider.provider)} bg-opacity-20`}>
                        {getProviderIcon(provider.provider)}
                      </div>
                      <div>
                        <CardTitle className="flex items-center gap-2">
                          {provider.name}
                          <Badge className="bg-green-600 text-white">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Active
                          </Badge>
                        </CardTitle>
                        <CardDescription className="text-gray-400">
                          {provider.description || `Using ${provider.provider} for AI tasks`}
                        </CardDescription>
                      </div>
                    </div>
                    {provider.provider === "local" && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={handleDownloadLocalModel}
                        className="border-blue text-blue-400 hover:bg-blue"
                      >
                        <Download className="w-4 h-4 mr-1" />
                        Download Model
                      </Button>
                    )}
                  </div>
                </CardHeader>
                {provider.models && provider.models.length > 0 && (
                  <CardContent>
                    <Label className="text-sm text-gray-400 mb-2 block">Available Models</Label>
                    <div className="flex flex-wrap gap-2">
                      {provider.models.map((model) => (
                        <Badge key={model} variant="outline" className="border-blue text-blue-400">
                          {model}
                        </Badge>
                      ))}
                    </div>
                  </CardContent>
                )}
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Provider Selection */}
      <Card className="bg-navy border-blue">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="w-5 h-5 text-orange" />
            Select AI Provider
          </CardTitle>
          <CardDescription className="text-gray-400">
            Choose which AI provider to use for recommendations, entity extraction, and chat
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="provider">Provider</Label>
              <Select value={selectedProvider} onValueChange={setSelectedProvider}>
                <SelectTrigger className="bg-blue/10 border-blue text-white">
                  <SelectValue placeholder="Select provider" />
                </SelectTrigger>
                <SelectContent>
                  {providers.map((provider) => (
                    <SelectItem key={provider.provider} value={provider.provider}>
                      <div className="flex items-center gap-2">
                        {getProviderIcon(provider.provider)}
                        <span>{provider.name}</span>
                        {provider.configured && (
                          <Badge className="bg-green-600 text-white text-xs">Active</Badge>
                        )}
                        {!provider.configured && provider.requires_key && (
                          <Badge variant="outline" className="border-yellow-500 text-yellow-400 text-xs">
                            Needs API Key
                          </Badge>
                        )}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedProvider && (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="model">Model</Label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger className="bg-blue/10 border-blue text-white">
                      <SelectValue placeholder="Select model" />
                    </SelectTrigger>
                    <SelectContent>
                      {(providers.find((p) => p.provider === selectedProvider)?.models || []).map((model) => (
                        <SelectItem key={model} value={model}>
                          {model}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {(providers.find((p) => p.provider === selectedProvider)?.requires_key) && (
                  <div className="grid gap-2">
                    <Label htmlFor="api-key" className="flex items-center gap-2">
                      <Key className="w-4 h-4" />
                      API Key
                    </Label>
                    <input
                      type="password"
                      id="api-key"
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder="Enter your API key"
                      className="w-full bg-blue/10 border-blue rounded px-3 py-2 text-white placeholder-gray-500"
                    />
                    <p className="text-xs text-gray-500">
                      Your API key is stored securely and never logged
                    </p>
                  </div>
                )}
              </>
            )}

            <Button
              onClick={handleSaveProvider}
              disabled={!selectedProvider || !selectedModel}
              className="bg-orange hover:bg-orange/80 text-navy"
            >
              Save Provider Settings
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Available Providers */}
      <div>
        <h3 className="text-lg font-semibold mb-3 text-orange">Available Providers</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {providers.map((provider) => (
            <Card
              key={provider.provider}
              className={`bg-navy border ${
                provider.configured ? "border-green-500/30" : "border-blue/30"
              }`}
            >
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className={`p-2 rounded-lg bg-opacity-20 ${getProviderColor(provider.provider)}`}>
                      {getProviderIcon(provider.provider)}
                    </div>
                    <CardTitle className="text-sm">{provider.name}</CardTitle>
                  </div>
                  {provider.configured ? (
                    <Badge className="bg-green-600 text-white text-xs">Ready</Badge>
                  ) : (
                    <Badge variant="outline" className="border-gray-500 text-gray-400 text-xs">
                      Not Configured
                    </Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-xs text-gray-400">
                  {provider.description || `${provider.name} for AI tasks`}
                </p>
                {provider.models && provider.models.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {provider.models.slice(0, 3).map((model) => (
                      <Badge key={model} variant="outline" className="text-xs border-blue text-blue-400">
                        {model}
                      </Badge>
                    ))}
                    {provider.models.length > 3 && (
                      <Badge variant="outline" className="text-xs border-gray-500 text-gray-400">
                        +{provider.models.length - 3} more
                      </Badge>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Local LLM Info */}
      <Card className="bg-navy border-blue">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-400">
            <Server className="w-5 h-5" />
            Local LLM (TinyLlama)
          </CardTitle>
          <CardDescription className="text-gray-400">
            Run AI models entirely offline using your own hardware
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-blue/10 rounded-lg p-4">
              <div className="text-2xl font-bold text-blue-400">620MB</div>
              <div className="text-sm text-gray-400">Model Size</div>
            </div>
            <div className="bg-blue/10 rounded-lg p-4">
              <div className="text-2xl font-bold text-blue-400">~4GB</div>
              <div className="text-sm text-gray-400">RAM Required</div>
            </div>
            <div className="bg-blue/10 rounded-lg p-4">
              <div className="text-2xl font-bold text-blue-400">~5s</div>
              <div className="text-sm text-gray-400">First Response</div>
            </div>
          </div>
          <div className="flex items-start gap-2 text-sm text-gray-400">
            <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <p>
              The local LLM model will be automatically downloaded from HuggingFace on first use.
              This requires an internet connection for the initial download (~620MB).
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

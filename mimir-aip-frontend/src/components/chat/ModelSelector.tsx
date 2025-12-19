"use client";

import { useState, useEffect } from "react";
import { Bot, Loader2 } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";

interface ModelSelectorProps {
  provider: string;
  model: string;
  onProviderChange: (provider: string) => void;
  onModelChange: (model: string) => void;
}

// Fallback providers if API fails
const FALLBACK_PROVIDERS = [
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
  { value: "ollama", label: "Ollama (Local)" },
  { value: "mock", label: "Mock (Testing)" },
];

// Default models for each provider (fallback if API doesn't provide)
const DEFAULT_MODELS: Record<string, Array<{ value: string; label: string }>> = {
  openai: [
    { value: "gpt-4", label: "GPT-4" },
    { value: "gpt-4-turbo", label: "GPT-4 Turbo" },
    { value: "gpt-3.5-turbo", label: "GPT-3.5 Turbo" },
  ],
  anthropic: [
    { value: "claude-3-opus", label: "Claude 3 Opus" },
    { value: "claude-3-sonnet", label: "Claude 3 Sonnet" },
    { value: "claude-3-haiku", label: "Claude 3 Haiku" },
  ],
  ollama: [
    { value: "llama3", label: "Llama 3" },
    { value: "mistral", label: "Mistral" },
    { value: "codellama", label: "Code Llama" },
  ],
  mock: [
    { value: "mock-gpt-4", label: "Mock GPT-4" },
    { value: "mock-claude-3", label: "Mock Claude 3" },
  ],
};

export function ModelSelector({
  provider,
  model,
  onProviderChange,
  onModelChange,
}: ModelSelectorProps) {
  const [providers, setProviders] = useState<Array<{ value: string; label: string }>>(FALLBACK_PROVIDERS);
  const [providerModels, setProviderModels] = useState<Record<string, Array<{ value: string; label: string }>>>(DEFAULT_MODELS);
  const [isLoading, setIsLoading] = useState(true);

  // Fetch available AI providers from plugins API
  useEffect(() => {
    async function loadProviders() {
      try {
        const response = await fetch('/api/v1/plugins');
        const allPlugins = await response.json();
        
        // Filter AI type plugins
        const aiPlugins = allPlugins.filter((p: any) => p.type === 'AI');
        
        if (aiPlugins.length > 0) {
          const providerList = aiPlugins.map((p: any) => ({
            value: p.name.toLowerCase(),
            label: p.name.charAt(0).toUpperCase() + p.name.slice(1),
          }));
          setProviders(providerList);
          
          // Build models map from API response
          const modelsMap: Record<string, Array<{ value: string; label: string }>> = { ...DEFAULT_MODELS };
          aiPlugins.forEach((p: any) => {
            if (p.available_models && p.available_models.length > 0) {
              const providerName = p.name.toLowerCase();
              modelsMap[providerName] = p.available_models.map((m: string) => ({
                value: m,
                label: formatModelName(m),
              }));
            }
          });
          setProviderModels(modelsMap);
        }
      } catch (error) {
        console.error('Failed to load AI providers:', error);
        // Keep fallback providers and models
      } finally {
        setIsLoading(false);
      }
    }

    loadProviders();
  }, []);

  // Format model name for display (e.g., "mock-gpt-4" -> "Mock GPT-4")
  function formatModelName(modelName: string): string {
    return modelName
      .split('-')
      .map(part => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ');
  }

  const availableModels = providerModels[provider] || [
    { value: provider, label: `Default ${provider}` }
  ];

  const handleProviderChange = (newProvider: string) => {
    onProviderChange(newProvider);
    // Auto-select first model of new provider
    const firstModel = providerModels[newProvider]?.[0]?.value || newProvider;
    onModelChange(firstModel);
  };

  return (
    <div className="flex items-center gap-4 p-3 bg-muted/30 rounded-lg border">
      {isLoading ? (
        <Loader2 className="h-5 w-5 text-muted-foreground animate-spin" />
      ) : (
        <Bot className="h-5 w-5 text-muted-foreground" />
      )}
      
      <div className="flex-1 flex items-center gap-4">
        <div className="flex-1">
          <Label className="text-xs text-muted-foreground mb-1">Provider</Label>
          <Select value={provider} onValueChange={handleProviderChange} disabled={isLoading}>
            <SelectTrigger className="h-8">
              <SelectValue placeholder="Select provider" />
            </SelectTrigger>
            <SelectContent>
              {providers.map((p) => (
                <SelectItem key={p.value} value={p.value}>
                  {p.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex-1">
          <Label className="text-xs text-muted-foreground mb-1">Model</Label>
          <Select value={model} onValueChange={onModelChange} disabled={isLoading}>
            <SelectTrigger className="h-8">
              <SelectValue placeholder="Select model" />
            </SelectTrigger>
            <SelectContent>
              {availableModels.map((m) => (
                <SelectItem key={m.value} value={m.value}>
                  {m.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  );
}

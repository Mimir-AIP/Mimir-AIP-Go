"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  listAPIKeys,
  createAPIKey,
  updateAPIKey,
  deleteAPIKey,
  testAPIKey,
  type APIKey,
  type CreateAPIKeyRequest,
} from "@/lib/api";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export default function APIKeysTab() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [formData, setFormData] = useState<CreateAPIKeyRequest>({
    provider: "openai",
    name: "",
    key_value: "",
    endpoint_url: "",
  });

  useEffect(() => {
    loadAPIKeys();
  }, []);

  async function loadAPIKeys() {
    try {
      setLoading(true);
      const keys = await listAPIKeys();
      setApiKeys(keys);
    } catch (error) {
      toast.error(`Failed to load API keys: ${error instanceof Error ? error.message : "Unknown error"}`);
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    if (!formData.name || !formData.key_value) {
      toast.error("Name and API Key are required");
      return;
    }

    try {
      await createAPIKey(formData);
      toast.success("API key created successfully");
      setDialogOpen(false);
      setFormData({ provider: "openai", name: "", key_value: "", endpoint_url: "" });
      loadAPIKeys();
    } catch (error) {
      toast.error(`Failed to create API key: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  async function handleToggle(id: string, currentActive: boolean) {
    try {
      await updateAPIKey(id, { is_active: !currentActive });
      toast.success(`API key ${!currentActive ? "enabled" : "disabled"}`);
      loadAPIKeys();
    } catch (error) {
      toast.error(`Failed to update API key: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  async function handleTest(id: string, name: string) {
    try {
      const result = await testAPIKey(id);
      if (result.success) {
        toast.success(`API key "${name}" is valid`);
      } else {
        toast.error(`API key test failed: ${result.message}`);
      }
    } catch (error) {
      toast.error(`Failed to test API key: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  async function handleDelete(id: string, name: string) {
    if (!confirm(`Are you sure you want to delete the API key "${name}"?`)) {
      return;
    }

    try {
      await deleteAPIKey(id);
      toast.success("API key deleted");
      loadAPIKeys();
    } catch (error) {
      toast.error(`Failed to delete API key: ${error instanceof Error ? error.message : "Unknown error"}`);
    }
  }

  if (loading) {
    return <div className="text-center py-8">Loading API keys...</div>;
  }

  return (
    <div className="space-y-6">
      {/* Header with Add button */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-xl font-semibold">API Keys</h2>
          <p className="text-sm text-gray-400 mt-1">
            Manage LLM provider API keys for OpenAI, Anthropic, Ollama, and custom endpoints
          </p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <Button className="bg-orange hover:bg-orange/80 text-navy">
              Add API Key
            </Button>
          </DialogTrigger>
          <DialogContent className="bg-navy border-blue text-white">
            <DialogHeader>
              <DialogTitle>Add New API Key</DialogTitle>
              <DialogDescription className="text-gray-400">
                Add a new LLM provider API key to enable AI features
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 mt-4">
              <div>
                <Label htmlFor="provider">Provider</Label>
                <Select
                  name="provider"
                  value={formData.provider}
                  onValueChange={(value) => setFormData({ ...formData, provider: value })}
                >
                  <SelectTrigger className="bg-blue border-blue text-white">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="anthropic">Anthropic</SelectItem>
                    <SelectItem value="ollama">Ollama</SelectItem>
                    <SelectItem value="custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="My OpenAI Key"
                  className="bg-blue border-blue text-white"
                />
              </div>
              <div>
                <Label htmlFor="key">API Key</Label>
                <Input
                  id="key"
                  name="key"
                  type="password"
                  value={formData.key_value}
                  onChange={(e) => setFormData({ ...formData, key_value: e.target.value })}
                  placeholder="sk-..."
                  className="bg-blue border-blue text-white"
                />
              </div>
              {(formData.provider === "ollama" || formData.provider === "custom") && (
                <div>
                  <Label htmlFor="endpoint">Endpoint URL</Label>
                  <Input
                    id="endpoint"
                    value={formData.endpoint_url}
                    onChange={(e) => setFormData({ ...formData, endpoint_url: e.target.value })}
                    placeholder="http://localhost:11434"
                    className="bg-blue border-blue text-white"
                  />
                </div>
              )}
              <div className="flex justify-end space-x-2 mt-6">
                <Button
                  variant="outline"
                  onClick={() => setDialogOpen(false)}
                  className="border-blue text-white hover:bg-blue"
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreate}
                  className="bg-orange hover:bg-orange/80 text-navy"
                >
                  Create
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* API Keys List */}
      {apiKeys.length === 0 ? (
        <Card className="bg-blue border-blue">
          <CardContent className="py-12 text-center text-gray-400">
            No API keys configured. Add one to enable LLM features.
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 api-keys-list">
          {apiKeys.map((key) => (
            <Card key={key.id} className="bg-blue border-blue">
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div>
                    <CardTitle className="text-orange">{key.name}</CardTitle>
                    <CardDescription className="text-gray-400">
                      Provider: {key.provider.toUpperCase()}
                      {key.endpoint_url && ` • Endpoint: ${key.endpoint_url}`}
                    </CardDescription>
                    <p className="text-xs text-gray-500 mt-1">
                      Created: {new Date(key.created_at).toLocaleDateString()}
                      {key.last_used_at && ` • Last used: ${new Date(key.last_used_at).toLocaleDateString()}`}
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleTest(key.id, key.name)}
                      className="border-blue text-white hover:bg-blue"
                    >
                      Test
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleToggle(key.id, key.is_active)}
                      className={`border-blue ${
                        key.is_active
                          ? "text-orange hover:bg-blue"
                          : "text-gray-400 hover:bg-blue"
                      }`}
                    >
                      {key.is_active ? "Enabled" : "Disabled"}
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleDelete(key.id, key.name)}
                      className="bg-red-600 hover:bg-red-700"
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

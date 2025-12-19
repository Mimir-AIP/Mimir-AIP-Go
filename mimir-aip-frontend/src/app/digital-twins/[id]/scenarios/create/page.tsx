"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getDigitalTwin,
  createScenario,
  type DigitalTwin,
  type SimulationEvent,
} from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { ArrowLeft, Plus, Trash2, Loader2, Settings, AlertCircle } from "lucide-react";

const EVENT_TYPES = [
  { value: "resource.unavailable", label: "Resource Unavailable", severity: "high" },
  { value: "resource.available", label: "Resource Available", severity: "low" },
  { value: "resource.capacity_change", label: "Capacity Change", severity: "medium" },
  { value: "demand.surge", label: "Demand Surge", severity: "high" },
  { value: "demand.drop", label: "Demand Drop", severity: "low" },
  { value: "process.delay", label: "Process Delay", severity: "medium" },
  { value: "process.failure", label: "Process Failure", severity: "critical" },
  { value: "policy.change", label: "Policy Change", severity: "medium" },
  { value: "external.supply_disruption", label: "Supply Disruption", severity: "critical" },
  { value: "custom", label: "Custom Event", severity: "medium" },
];

interface EventFormData {
  id: string;
  type: string;
  target_uri: string;
  timestamp: number;
  parameters: string; // JSON string
}

export default function CreateScenarioPage() {
  const params = useParams();
  const router = useRouter();
  const twinId = params.id as string;

  const [twin, setTwin] = useState<DigitalTwin | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  const [formData, setFormData] = useState({
    name: "",
    description: "",
    scenario_type: "",
    duration: 100,
  });

  const [events, setEvents] = useState<EventFormData[]>([]);

  useEffect(() => {
    loadTwin();
  }, [twinId]);

  async function loadTwin() {
    try {
      setLoading(true);
      const data = await getDigitalTwin(twinId);
      setTwin(data);
    } catch (err) {
      toast.error("Failed to load digital twin");
      router.push("/digital-twins");
    } finally {
      setLoading(false);
    }
  }

  function addEvent() {
    setEvents([
      ...events,
      {
        id: `event-${Date.now()}`,
        type: "resource.unavailable",
        target_uri: "",
        timestamp: 0,
        parameters: "{}",
      },
    ]);
  }

  function removeEvent(id: string) {
    setEvents(events.filter((e) => e.id !== id));
  }

  function updateEvent(id: string, updates: Partial<EventFormData>) {
    setEvents(events.map((e) => (e.id === id ? { ...e, ...updates } : e)));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!formData.name || events.length === 0) {
      toast.error("Please provide a name and at least one event");
      return;
    }

    // Validate events
    for (const event of events) {
      if (!event.target_uri) {
        toast.error("All events must have a target URI");
        return;
      }

      try {
        JSON.parse(event.parameters);
      } catch {
        toast.error(`Invalid JSON in event parameters for ${event.type}`);
        return;
      }
    }

    setSubmitting(true);
    try {
      const simulationEvents: SimulationEvent[] = events.map((e) => ({
        id: "",
        type: e.type,
        target_uri: e.target_uri,
        timestamp: e.timestamp,
        parameters: JSON.parse(e.parameters),
        impact: {
          affected_entities: [],
          state_changes: {},
          propagation_rules: [],
          severity: EVENT_TYPES.find((t) => t.value === e.type)?.severity || "medium",
        },
      }));

      const response = await createScenario(twinId, {
        name: formData.name,
        description: formData.description || undefined,
        scenario_type: formData.scenario_type || undefined,
        events: simulationEvents,
        duration: formData.duration,
      });

      toast.success("Scenario created successfully!");
      router.push(`/digital-twins/${twinId}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create scenario";
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto py-8">
        <div className="flex items-center gap-2">
          <Loader2 className="h-5 w-5 animate-spin" />
          Loading...
        </div>
      </div>
    );
  }

  if (!twin) return null;

  return (
    <div className="container mx-auto py-8 max-w-4xl">
      <Link href={`/digital-twins/${twinId}`}>
        <Button variant="ghost" className="mb-4">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Twin
        </Button>
      </Link>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-6 w-6" />
            Create Simulation Scenario
          </CardTitle>
          <CardDescription>
            Define a what-if scenario for {twin.name}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Scenario Info */}
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">
                  Scenario Name <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="name"
                  placeholder="e.g., Supply Chain Disruption Q1 2025"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  placeholder="Describe what this scenario tests..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="type">Scenario Type (Optional)</Label>
                  <Input
                    id="type"
                    placeholder="e.g., supply_shock, demand_surge"
                    value={formData.scenario_type}
                    onChange={(e) => setFormData({ ...formData, scenario_type: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="duration">
                    Duration (steps) <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="duration"
                    type="number"
                    min="1"
                    value={formData.duration}
                    onChange={(e) => setFormData({ ...formData, duration: parseInt(e.target.value) })}
                    required
                  />
                </div>
              </div>
            </div>

            {/* Events */}
            <div className="space-y-4 pt-4 border-t">
              <div className="flex items-center justify-between">
                <Label className="text-lg">
                  Events <span className="text-red-500">*</span>
                </Label>
                <Button type="button" onClick={addEvent} size="sm">
                  <Plus className="h-4 w-4 mr-2" />
                  Add Event
                </Button>
              </div>

              {events.length === 0 ? (
                <div className="bg-muted/50 rounded-lg p-8 text-center">
                  <AlertCircle className="h-12 w-12 mx-auto text-muted-foreground mb-2" />
                  <p className="text-sm text-muted-foreground">
                    No events yet. Add at least one event to create a scenario.
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {events.map((event, index) => (
                    <Card key={event.id} className="border-2">
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <Badge variant="outline">Event {index + 1}</Badge>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => removeEvent(event.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
                        <div className="grid grid-cols-2 gap-3">
                          <div className="space-y-1">
                            <Label className="text-xs">Event Type</Label>
                            <select
                              className="w-full border rounded-md p-2 text-sm"
                              value={event.type}
                              onChange={(e) => updateEvent(event.id, { type: e.target.value })}
                            >
                              {EVENT_TYPES.map((type) => (
                                <option key={type.value} value={type.value}>
                                  {type.label}
                                </option>
                              ))}
                            </select>
                          </div>
                          <div className="space-y-1">
                            <Label className="text-xs">Timestamp (step)</Label>
                            <Input
                              type="number"
                              min="0"
                              value={event.timestamp}
                              onChange={(e) =>
                                updateEvent(event.id, { timestamp: parseInt(e.target.value) })
                              }
                              className="text-sm"
                            />
                          </div>
                        </div>

                        <div className="space-y-1">
                          <Label className="text-xs">Target Entity URI</Label>
                          <select
                            className="w-full border rounded-md p-2 text-sm font-mono"
                            value={event.target_uri}
                            onChange={(e) => updateEvent(event.id, { target_uri: e.target.value })}
                          >
                            <option value="">Select entity...</option>
                            {(twin.entities || []).map((entity) => (
                              <option key={entity.uri} value={entity.uri}>
                                {entity.label} ({entity.uri})
                              </option>
                            ))}
                          </select>
                        </div>

                        <div className="space-y-1">
                          <Label className="text-xs">Parameters (JSON)</Label>
                          <Textarea
                            value={event.parameters}
                            onChange={(e) => updateEvent(event.id, { parameters: e.target.value })}
                            placeholder='{"percentage": 0.5, "reason": "maintenance"}'
                            rows={3}
                            className="font-mono text-xs"
                          />
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </div>

            {/* Submit */}
            <div className="flex justify-end gap-3 pt-4 border-t">
              <Link href={`/digital-twins/${twinId}`}>
                <Button type="button" variant="outline" disabled={submitting}>
                  Cancel
                </Button>
              </Link>
              <Button type="submit" disabled={submitting || events.length === 0}>
                {submitting ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Creating...
                  </>
                ) : (
                  "Create Scenario"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

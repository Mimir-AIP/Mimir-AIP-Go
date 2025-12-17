"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { listDigitalTwins, listOntologies, type DigitalTwin, type Ontology } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { toast } from "sonner";
import { Plus, Network, Calendar, Database } from "lucide-react";

export default function DigitalTwinsPage() {
  const router = useRouter();
  const [twins, setTwins] = useState<DigitalTwin[]>([]);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      setLoading(true);
      setError(null);
      const [twinsData, ontologiesData] = await Promise.all([
        listDigitalTwins(),
        listOntologies(),
      ]);
      setTwins(twinsData);
      setOntologies(ontologiesData);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load digital twins";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  function getOntologyName(ontologyId: string): string {
    const ontology = ontologies.find((o) => o.id === ontologyId);
    return ontology ? ontology.name : ontologyId;
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
            <Button onClick={loadData} className="mt-4">
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
          <h1 className="text-3xl font-bold">Digital Twins</h1>
          <p className="text-muted-foreground mt-2">
            Simulate business scenarios and analyze impact on your organization
          </p>
        </div>
        <Link href="/digital-twins/create">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Twin
          </Button>
        </Link>
      </div>

      {twins.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12">
              <Network className="h-16 w-16 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-xl font-semibold mb-2">No Digital Twins Yet</h3>
              <p className="text-muted-foreground mb-6">
                Create your first digital twin from an ontology to start simulating scenarios
              </p>
              <Link href="/digital-twins/create">
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Your First Twin
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {twins.map((twin) => (
            <Card
              key={twin.id}
              className="hover:shadow-lg transition-shadow cursor-pointer"
              onClick={() => router.push(`/digital-twins/${twin.id}`)}
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="flex items-center gap-2">
                      <Network className="h-5 w-5" />
                      {twin.name}
                    </CardTitle>
                    <CardDescription className="mt-2">
                      {twin.description || "No description"}
                    </CardDescription>
                  </div>
                  <Badge variant="secondary">{twin.model_type}</Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm">
                    <Database className="h-4 w-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Ontology:</span>
                    <span className="font-medium truncate">
                      {getOntologyName(twin.ontology_id)}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-4 pt-2 border-t">
                    <div>
                      <p className="text-2xl font-bold">{twin.entities.length}</p>
                      <p className="text-xs text-muted-foreground">Entities</p>
                    </div>
                    <div>
                      <p className="text-2xl font-bold">{twin.relationships.length}</p>
                      <p className="text-xs text-muted-foreground">Relationships</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 text-xs text-muted-foreground pt-2 border-t">
                    <Calendar className="h-3 w-3" />
                    Created {formatDate(twin.created_at)}
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

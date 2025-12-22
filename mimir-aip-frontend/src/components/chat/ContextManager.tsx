"use client";

import { useState } from "react";
import { Gauge, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";

interface ContextManagerProps {
  conversationId: string;
  currentTokens: number;
  maxTokens: number;
  onCompact: () => Promise<void>;
}

export function ContextManager({
  conversationId,
  currentTokens,
  maxTokens,
  onCompact,
}: ContextManagerProps) {
  const [isCompacting, setIsCompacting] = useState(false);

  const percentage = Math.min(100, (currentTokens / maxTokens) * 100);
  const isNearLimit = percentage > 80;

  const handleCompact = async () => {
    setIsCompacting(true);
    try {
      await onCompact();
      toast.success("Context compacted", {
        description: "Conversation history has been summarized to save tokens",
      });
    } catch (error) {
      toast.error("Failed to compact context", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setIsCompacting(false);
    }
  };

  return (
    <div className="flex items-center gap-3 px-4 py-2 bg-muted/20 border-b">
      <Gauge className="h-4 w-4 text-muted-foreground" />
      
      <div className="flex-1">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs text-muted-foreground">Context Usage</span>
          <span className={`text-xs font-medium ${isNearLimit ? 'text-orange-600' : 'text-muted-foreground'}`}>
            {currentTokens.toLocaleString()} / {maxTokens.toLocaleString()} tokens
          </span>
        </div>
        <Progress 
          value={percentage} 
          className={`h-1.5 ${isNearLimit ? '[&>*]:bg-orange-600' : ''}`}
        />
      </div>

      {isNearLimit && (
        <Button
          size="sm"
          variant="outline"
          onClick={handleCompact}
          disabled={isCompacting}
          className="gap-2"
        >
          <Zap className="h-3 w-3" />
          {isCompacting ? "Compacting..." : "Compact"}
        </Button>
      )}
    </div>
  );
}

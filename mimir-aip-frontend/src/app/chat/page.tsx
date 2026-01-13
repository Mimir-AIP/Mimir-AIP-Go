"use client";

import { useState, useEffect } from "react";
import { AgentChat } from "@/components/chat/AgentChat";

export default function ChatPage() {
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    // Set page title dynamically
    document.title = "AI Agent Chat - Mimir AIP";
    setIsLoaded(true);
  }, []);

  return (
    <div className="container mx-auto py-6">
      {isLoaded && <AgentChat />}
    </div>
  );
}

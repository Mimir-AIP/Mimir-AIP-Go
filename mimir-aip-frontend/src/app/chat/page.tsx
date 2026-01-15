"use client";

import { AgentChat } from "@/components/chat/AgentChat";
import { Metadata } from "next";

// Note: metadata export doesn't work in client components, 
// so we set the title in the component itself
export default function ChatPage() {
  return (
    <div className="container mx-auto py-6">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-orange">AI Agent Chat</h1>
        <p className="text-gray-400 mt-1">
          Chat with Mimir AI to analyze data, create scenarios, and manage your pipelines
        </p>
      </div>
      <AgentChat />
    </div>
  );
}

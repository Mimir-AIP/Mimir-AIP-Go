"use client";

import { useEffect, useRef } from "react";
import { User, Bot } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChatMessage } from "@/lib/api";
import { ToolCallCard } from "./ToolCallCard";

interface MessageListProps {
  messages: ChatMessage[];
}

export function MessageList({ messages }: MessageListProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? "s" : ""} ago`;
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`;
    if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`;
    
    return date.toLocaleDateString();
  };

  if (messages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center text-muted-foreground">
          <Bot className="h-16 w-16 mx-auto mb-4 opacity-20" />
          <p className="text-lg font-medium">No messages yet</p>
          <p className="text-sm">Start a conversation with the agent</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {messages.map((message) => {
        const isUser = message.role === "user";
        const isSystem = message.role === "system";

        if (isSystem) {
          return (
            <div key={message.id} className="flex justify-center">
              <div className="bg-muted/50 rounded-lg px-4 py-2 text-xs text-muted-foreground max-w-md text-center">
                {message.content}
              </div>
            </div>
          );
        }

        return (
          <div
            key={message.id}
            className={cn("flex gap-3", isUser ? "justify-end" : "justify-start")}
          >
            {!isUser && (
              <div className="h-8 w-8 rounded-full bg-blue-500 flex items-center justify-center shrink-0">
                <Bot className="h-5 w-5 text-white" />
              </div>
            )}

            <div className={cn("flex flex-col gap-1 max-w-[70%]", isUser && "items-end")}>
              <div
                className={cn(
                  "rounded-lg px-4 py-2 break-words",
                  isUser
                    ? "bg-[#FF6B35] text-white"
                    : "bg-[#1E3A8A] text-white"
                )}
              >
                <p className="whitespace-pre-wrap">{message.content}</p>
              </div>

              {message.tool_calls && message.tool_calls.length > 0 && (
                <div className="w-full">
                  <ToolCallCard toolCalls={message.tool_calls} />
                </div>
              )}

              <span className="text-xs text-muted-foreground px-2">
                {formatTimestamp(message.created_at)}
              </span>
            </div>

            {isUser && (
              <div className="h-8 w-8 rounded-full bg-[#FF6B35] flex items-center justify-center shrink-0">
                <User className="h-5 w-5 text-white" />
              </div>
            )}
          </div>
        );
      })}
      <div ref={messagesEndRef} />
    </div>
  );
}

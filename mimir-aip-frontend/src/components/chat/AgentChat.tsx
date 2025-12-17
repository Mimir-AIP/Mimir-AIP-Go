"use client";

import { useState, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { toast } from "sonner";
import {
  listConversations,
  createConversation,
  getConversation,
  deleteConversation,
  sendMessage,
  type ChatConversation,
  type ChatMessage,
} from "@/lib/api";
import { ConversationSidebar } from "./ConversationSidebar";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { ModelSelector } from "./ModelSelector";

interface AgentChatProps {
  twinId?: string;
}

export function AgentChat({ twinId }: AgentChatProps) {
  // State
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isSending, setIsSending] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  
  // Model settings
  const [modelProvider, setModelProvider] = useState("openai");
  const [modelName, setModelName] = useState("gpt-4");

  // Load conversations on mount
  useEffect(() => {
    loadConversations();
  }, [twinId]);

  // Load messages when conversation changes
  useEffect(() => {
    if (activeConversationId) {
      loadMessages(activeConversationId);
    } else {
      setMessages([]);
    }
  }, [activeConversationId]);

  const loadConversations = async () => {
    try {
      setIsLoading(true);
      const convs = await listConversations(twinId);
      // Sort by updated_at descending
      convs.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
      setConversations(convs);
      
      // If no active conversation and conversations exist, select first one
      if (!activeConversationId && convs.length > 0) {
        setActiveConversationId(convs[0].id);
      }
    } catch (error) {
      toast.error("Error loading conversations", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const loadMessages = async (conversationId: string) => {
    try {
      const data = await getConversation(conversationId);
      setMessages(data.messages);
      
      // Update model settings from conversation
      if (data.conversation.model_provider) {
        setModelProvider(data.conversation.model_provider);
      }
      if (data.conversation.model_name) {
        setModelName(data.conversation.model_name);
      }
    } catch (error) {
      toast.error("Error loading messages", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const handleNewConversation = async () => {
    try {
      const title = `Conversation ${new Date().toLocaleString()}`;
      const response = await createConversation({
        twin_id: twinId,
        title,
        model_provider: modelProvider,
        model_name: modelName,
        system_prompt: "You are a helpful AI assistant for the Mimir AIP Digital Twin system.",
      });

      setConversations((prev) => [response.conversation, ...prev]);
      setActiveConversationId(response.conversation.id);
      setMessages([]);
      setInputValue("");

      toast.success("New conversation created", {
        description: "Start chatting with the agent",
      });
    } catch (error) {
      toast.error("Error creating conversation", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const handleDeleteConversation = async (id: string) => {
    try {
      await deleteConversation(id);
      
      setConversations((prev) => prev.filter((c) => c.id !== id));
      
      if (activeConversationId === id) {
        const remaining = conversations.filter((c) => c.id !== id);
        setActiveConversationId(remaining.length > 0 ? remaining[0].id : null);
      }

      toast.success("Conversation deleted");
    } catch (error) {
      toast.error("Error deleting conversation", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  const handleSendMessage = async () => {
    if (!activeConversationId || !inputValue.trim()) return;

    const userMessage = inputValue.trim();
    setInputValue("");
    setIsSending(true);

    try {
      const response = await sendMessage(
        activeConversationId,
        userMessage,
        modelProvider,
        modelName
      );

      // Add both user message and assistant reply to the list
      setMessages((prev) => [...prev, response.user_message, response.assistant_reply]);

      // Update conversation list to reflect new message count and updated_at
      await loadConversations();
    } catch (error) {
      toast.error("Error sending message", {
        description: error instanceof Error ? error.message : "Unknown error",
      });
      // Restore input value on error
      setInputValue(userMessage);
    } finally {
      setIsSending(false);
    }
  };

  if (isLoading) {
    return (
      <Card className="h-[600px] flex items-center justify-center">
        <p className="text-muted-foreground">Loading conversations...</p>
      </Card>
    );
  }

  return (
    <Card className="h-[600px] flex overflow-hidden">
      <ConversationSidebar
        conversations={conversations}
        activeId={activeConversationId}
        onSelect={setActiveConversationId}
        onNew={handleNewConversation}
        onDelete={handleDeleteConversation}
      />

      <div className="flex-1 flex flex-col">
        {activeConversationId ? (
          <>
            <div className="p-3 border-b">
              <ModelSelector
                provider={modelProvider}
                model={modelName}
                onProviderChange={setModelProvider}
                onModelChange={setModelName}
              />
            </div>

            <MessageList messages={messages} />

            <MessageInput
              value={inputValue}
              onChange={setInputValue}
              onSend={handleSendMessage}
              disabled={isSending}
              placeholder="Ask the agent to create scenarios, run simulations, or analyze results..."
            />
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center p-8">
            <div className="text-center text-muted-foreground">
              <p className="text-lg font-medium mb-2">No conversation selected</p>
              <p className="text-sm">Create a new conversation to get started</p>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}

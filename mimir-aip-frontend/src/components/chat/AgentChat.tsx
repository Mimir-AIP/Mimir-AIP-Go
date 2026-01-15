"use client";

import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "sonner";
import { Send, Loader2, Bot, User, Settings2 } from "lucide-react";
import {
  createConversation,
  sendMessage,
  type ChatMessage,
  type MCPTool,
} from "@/lib/api";
import { ModelSelector } from "./ModelSelector";
import { MCPToolsPanel } from "./MCPToolsPanel";

interface AgentChatProps {
  twinId?: string;
}

export function AgentChat({ twinId }: AgentChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isSending, setIsSending] = useState(false);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [modelProvider, setModelProvider] = useState("mock");
  const [modelName, setModelName] = useState("mock-gpt-4");
  const [showModelSelector, setShowModelSelector] = useState(false);
  const [availableTools, setAvailableTools] = useState<MCPTool[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Create conversation on mount
  useEffect(() => {
    const initConversation = async () => {
      try {
        const response = await createConversation({
          twin_id: twinId,
          title: `Chat ${new Date().toLocaleTimeString()}`,
          model_provider: modelProvider,
          model_name: modelName,
          system_prompt: "You are Mimir, a helpful AI assistant for data analysis and pipeline orchestration.",
        });
        setConversationId(response.conversation.id);
      } catch (error) {
        toast.error("Failed to initialize chat");
        console.error(error);
      }
    };

    initConversation();
  }, [twinId, modelProvider, modelName]);

  const handleSendMessage = async () => {
    if (!conversationId || !inputValue.trim() || isSending) return;

    const userMessage = inputValue.trim();
    setInputValue("");
    setIsSending(true);

    try {
      const response = await sendMessage(
        conversationId,
        userMessage,
        modelProvider,
        modelName
      );

      // Add both user message and assistant reply
      setMessages((prev) => [...prev, response.user_message, response.assistant_reply]);
    } catch (error) {
      toast.error("Failed to send message");
      console.error(error);
      // Restore input on error
      setInputValue(userMessage);
    } finally {
      setIsSending(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-12rem)] max-w-4xl mx-auto bg-gradient-to-b from-navy to-blue/5 rounded-lg shadow-xl">
      {/* Model Selector Toggle - Top Right */}
      <div className="absolute top-4 right-4 z-10">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowModelSelector(!showModelSelector)}
          className="bg-navy/80 backdrop-blur border-blue text-white hover:bg-navy hover:text-orange"
          data-testid="model-selector-toggle"
        >
          <Settings2 className="h-4 w-4 mr-2" />
          {modelName.split('-').map(p => p.charAt(0).toUpperCase() + p.slice(1)).join(' ')}
        </Button>
      </div>

      {/* Model Selector Panel */}
      {showModelSelector && (
        <div className="p-4 border-b border-blue bg-navy/60 backdrop-blur" data-testid="model-selector-panel">
          <ModelSelector
            provider={modelProvider}
            model={modelName}
            onProviderChange={setModelProvider}
            onModelChange={setModelName}
          />
        </div>
      )}

      {/* Tools Panel */}
      <MCPToolsPanel onToolsLoaded={setAvailableTools} />

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto p-6 space-y-4">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-center">
            <Bot className="h-16 w-16 text-orange mb-4" />
            <p className="text-xl font-semibold text-white mb-2">
              Chat with Mimir AI
            </p>
            <p className="text-gray-400 max-w-md">
              Send a message to start a conversation. Ask about your data, create scenarios, run simulations, or analyze results.
              I can help with pipelines, ontologies, and digital twins.
            </p>
            <div className="mt-6 flex flex-wrap gap-2 justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setInputValue("What can you help me with?")}
                className="text-sm border-blue text-blue-400 hover:bg-blue/20"
              >
                What can you do?
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setInputValue("Show me my data")}
                className="text-sm border-blue text-blue-400 hover:bg-blue/20"
              >
                Show my data
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setInputValue("TRIGGER_TOOL: create_scenario")}
                className="text-sm border-blue text-blue-400 hover:bg-blue/20"
              >
                Create a scenario
              </Button>
            </div>
          </div>
        )}

        {messages.map((message, index) => (
          <div
            key={index}
            data-testid="chat-message"
            className={`flex gap-3 ${
              message.role === "user" ? "justify-end" : "justify-start"
            }`}
          >
            {message.role === "assistant" && (
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-orange flex items-center justify-center">
                <Bot className="h-5 w-5 text-navy" />
              </div>
            )}
            
            <div
              className={`max-w-[70%] rounded-2xl px-4 py-3 ${
                message.role === "user"
                  ? "bg-orange text-navy"
                  : "bg-blue/30 text-white border border-blue"
              }`}
            >
              <p className="text-sm whitespace-pre-wrap break-words">
                {message.content}
              </p>
              
              {/* Show tool calls if present */}
              {message.tool_calls && message.tool_calls.length > 0 && (
                <div className="mt-3 pt-3 border-t border-blue/50">
                  {message.tool_calls.map((tool, idx) => (
                    <div key={idx} className="text-xs font-mono bg-navy/50 rounded p-2 mb-2">
                      <div className="text-orange font-semibold mb-1">
                        🔧 {tool.tool_name}
                      </div>
                      <div className="mb-2">
                        <div className="text-muted-foreground font-semibold mb-1">Input:</div>
                        <pre className="text-muted-foreground overflow-x-auto max-h-32">
                          {JSON.stringify(tool.input, null, 2)}
                        </pre>
                      </div>
                      {tool.output !== undefined && tool.output !== null && (
                        <div>
                          <div className="text-green-400 font-semibold mb-1">Output:</div>
                          <pre className="text-muted-foreground overflow-x-auto max-h-48">
                            {String(JSON.stringify(tool.output, null, 2))}
                          </pre>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              <div className="text-xs mt-1 opacity-60">
                {new Date(message.created_at).toLocaleTimeString()}
              </div>
            </div>

            {message.role === "user" && (
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-blue flex items-center justify-center">
                <User className="h-5 w-5 text-white" />
              </div>
            )}
          </div>
        ))}

        {isSending && (
          <div className="flex gap-3 justify-start" data-testid="typing-indicator">
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-orange flex items-center justify-center">
              <Bot className="h-5 w-5 text-navy" />
            </div>
            <div className="bg-blue/30 rounded-2xl px-4 py-3 border border-blue">
              <Loader2 className="h-4 w-4 animate-spin text-orange" />
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="border-t border-blue bg-navy/50 p-4">
        <div className="flex gap-3 items-end max-w-4xl mx-auto">
          <Textarea
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyPress}
            placeholder="Type a message... (Enter to send, Shift+Enter for new line)"
            disabled={isSending || !conversationId}
            className="flex-1 min-h-[60px] max-h-[200px] resize-none bg-blue/20 border-blue text-white placeholder:text-gray-500 rounded-xl"
            rows={2}
            data-testid="chat-input"
          />
          <Button
            onClick={handleSendMessage}
            disabled={!inputValue.trim() || isSending || !conversationId}
            size="lg"
            className="bg-orange hover:bg-orange/90 text-navy rounded-xl px-6"
            data-testid="send-button"
            aria-label="Send"
          >
            {isSending ? (
              <Loader2 className="h-5 w-5 animate-spin" />
            ) : (
              <Send className="h-5 w-5" />
            )}
          </Button>
        </div>
        <p className="text-xs text-gray-400 mt-2 text-center">
          Powered by Mimir AI • Model: {modelName.split('-').map(p => p.charAt(0).toUpperCase() + p.slice(1)).join(' ')} ({modelProvider}) • {availableTools.length} tools available
        </p>
      </div>
    </div>
  );
}

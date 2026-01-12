"use client";

import { useState, useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { ChevronDown, Search, Loader2 } from "lucide-react";

interface ModelSelectProps {
  value: string;
  fetchUrl: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export function ModelSelect({
  value,
  fetchUrl,
  onChange,
  placeholder = "Select or search model...",
  className = "",
}: ModelSelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [models, setModels] = useState<string[]>([]);
  const [filteredModels, setFilteredModels] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const fetchModels = async () => {
      if (!fetchUrl) return;
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(fetchUrl);
        if (!response.ok) {
          throw new Error(`Failed to fetch: ${response.status}`);
        }
        const data = await response.json();
        const modelList = data.models || [];
        setModels(modelList);
        setFilteredModels(modelList);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load models");
        // Use fallback models
        setModels(["gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"]);
        setFilteredModels(["gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"]);
      } finally {
        setLoading(false);
      }
    };

    fetchModels();
  }, [fetchUrl]);

  useEffect(() => {
    if (search) {
      setFilteredModels(
        models.filter((m) =>
          m.toLowerCase().includes(search.toLowerCase())
        )
      );
    } else {
      setFilteredModels(models);
    }
  }, [search, models]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className={`relative ${className}`} ref={dropdownRef}>
      <div
        className="flex items-center gap-2 px-3 py-2 bg-navy border border-blue rounded-md cursor-pointer hover:border-orange/50 transition-colors"
        onClick={() => {
          setIsOpen(!isOpen);
          if (!isOpen && inputRef.current) {
            setTimeout(() => inputRef.current?.focus(), 0);
          }
        }}
      >
        <Search className="w-4 h-4 text-gray-400" />
        <input
          ref={inputRef}
          type="text"
          value={search || value}
          onChange={(e) => {
            setSearch(e.target.value);
            setIsOpen(true);
          }}
          onFocus={() => setIsOpen(true)}
          placeholder={value || placeholder}
          className="flex-1 bg-transparent border-none outline-none text-white placeholder-gray-500"
        />
        {loading ? (
          <Loader2 className="w-4 h-4 animate-spin text-orange" />
        ) : (
          <ChevronDown
            className={`w-4 h-4 text-gray-400 transition-transform ${
              isOpen ? "rotate-180" : ""
            }`}
          />
        )}
      </div>

      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-navy border border-blue rounded-md shadow-lg max-h-60 overflow-y-auto">
          {error && (
            <div className="px-3 py-2 text-xs text-red-400 border-b border-blue">
              {error}
            </div>
          )}
          {filteredModels.length === 0 && !loading ? (
            <div className="px-3 py-2 text-sm text-gray-500">
              No models found
            </div>
          ) : (
            filteredModels.map((model) => (
              <div
                key={model}
                className={`px-3 py-2 text-sm cursor-pointer hover:bg-blue/30 transition-colors ${
                  model === value ? "bg-blue/50 text-orange" : "text-white"
                }`}
                onClick={() => {
                  onChange(model);
                  setSearch("");
                  setIsOpen(false);
                }}
              >
                {model}
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}

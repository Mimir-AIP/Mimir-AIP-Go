"use client";

import { Component, ReactNode } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("Error caught by boundary:", error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <Card className="bg-navy text-white border-red-500 p-6">
          <h2 className="text-xl font-bold text-red-500 mb-4">Something went wrong</h2>
          <p className="text-white/80 mb-4">
            {this.state.error?.message || "An unexpected error occurred"}
          </p>
          <Button
            onClick={() => this.setState({ hasError: false, error: null })}
            variant="destructive"
          >
            Try Again
          </Button>
        </Card>
      );
    }

    return this.props.children;
  }
}

// Simple error display component for use in pages
export function ErrorDisplay({ error, onRetry }: { error: string; onRetry?: () => void }) {
  return (
    <Card className="bg-navy text-white border-red-500 p-6">
      <h3 className="text-lg font-bold text-red-500 mb-2">Error</h3>
      <p className="text-white/80 mb-4">{error}</p>
      {onRetry && (
        <Button onClick={onRetry} variant="destructive" size="sm">
          Retry
        </Button>
      )}
    </Card>
  );
}

// API client for Mimir AIP backend - TanStack version

const API_BASE_URL = import.meta.env.VITE_API_URL || "";

// Get stored auth token
function getAuthToken(): string | null {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('auth_token') || 
           document.cookie.split('; ').find(row => row.startsWith('auth_token='))?.split('=')[1] || null;
  }
  return null;
}

// Generic fetch wrapper with error handling and auth
async function apiFetch<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;
  
  const token = getAuthToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string> || {}),
  };
  
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  
  try {
    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`API error (${response.status}): ${errorText || response.statusText}`);
    }

    const contentType = response.headers.get("content-type");
    if (contentType && contentType.includes("application/json")) {
      return await response.json();
    }
    
    return {} as T;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error("Unknown API error");
  }
}

// Job interface
export interface Job {
  id: string;
  name: string;
  status: string;
  type?: string;
  created_at?: string;
  updated_at?: string;
}

// API functions
export async function getJobs(): Promise<Job[]> {
  return apiFetch<Job[]>("/api/v1/jobs");
}

export async function getRunningJobs(): Promise<Job[]> {
  return apiFetch<Job[]>("/api/v1/jobs?status=running");
}

export async function getRecentJobs(): Promise<Job[]> {
  return apiFetch<Job[]>("/api/v1/jobs?limit=10");
}

export async function getPerformanceMetrics(): Promise<Record<string, unknown>> {
  return apiFetch<Record<string, unknown>>("/api/v1/monitoring/metrics");
}

"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { login } from "@/lib/api";

export default function LoginPage() {
  const [credentials, setCredentials] = useState({
    username: "",
    password: "",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const response = await login(credentials.username, credentials.password);
      
      // Store token
      if (typeof window !== 'undefined') {
        localStorage.setItem('auth_token', response.token);
        const maxAge = response.expires_in || 86400; // Default to 24 hours
        document.cookie = `auth_token=${response.token}; path=/; max-age=${maxAge}; SameSite=Strict`;
        
        // Wait a moment for cookie to be set
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      
      // Redirect to dashboard or redirect parameter
      const urlParams = new URLSearchParams(window.location.search);
      const redirect = urlParams.get('redirect') || '/dashboard';
      router.push(redirect);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: string, value: string) => {
    setCredentials(prev => ({ ...prev, [field]: value }));
  };

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <Card className="bg-navy text-white border-blue p-8 max-w-md w-full">
        <h1 className="text-3xl font-bold text-orange mb-6">Login to Mimir AIP</h1>
        
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="username" className="block text-sm font-medium text-white/80 mb-2">
              Username
            </label>
            <input
              id="username"
              name="username"
              type="text"
              value={credentials.username}
              onChange={(e) => handleChange('username', e.target.value)}
              className="w-full px-3 py-2 bg-navy-light border border-blue/50 rounded-md text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-orange focus:border-transparent"
              placeholder="Enter your username"
              required
            />
          </div>
          
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-white/80 mb-2">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              value={credentials.password}
              onChange={(e) => handleChange('password', e.target.value)}
              className="w-full px-3 py-2 bg-navy-light border border-blue/50 rounded-md text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-orange focus:border-transparent"
              placeholder="Enter your password"
              required
            />
          </div>
          
          {error && (
            <div className="bg-red/20 border border-red/50 rounded p-3">
              <p className="text-red text-sm">{error}</p>
            </div>
          )}
          
          <Button
            type="submit"
            disabled={loading || !credentials.username || !credentials.password}
            className="w-full bg-orange hover:bg-orange/90 text-navy font-medium"
          >
            {loading ? "Logging in..." : "Login"}
          </Button>
        </form>
        
        <div className="mt-6 pt-6 border-t border-blue/30">
          <p className="text-sm text-white/60 text-center">
            For development access, check your authentication configuration in <code className="bg-black/30 px-1 rounded">config.yaml</code>
          </p>
        </div>
      </Card>
    </div>
  );
}

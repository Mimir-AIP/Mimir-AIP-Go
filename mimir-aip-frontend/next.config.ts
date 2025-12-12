import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable standalone output for Docker deployment
  output: 'standalone',
  
  // Disable image optimization for static export
  images: {
    unoptimized: true,
  },
  
  // Set base path if needed (empty for root)
  basePath: '',
};

export default nextConfig;

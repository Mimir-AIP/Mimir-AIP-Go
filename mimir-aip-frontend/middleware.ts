import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// List of protected routes
const protectedRoutes = [
  '/dashboard',
  '/ontologies',
  '/pipelines', 
  '/models',
  '/digital-twins',
  '/monitoring',
  '/extraction',
  '/workflows',
  '/knowledge-graph',
  '/jobs',
  '/settings',
  '/chat',
];

// List of public routes (don't require auth)
const publicRoutes = [
  '/',
  '/login',
  '/api',
  '/_next',
  '/favicon.ico',
];

// Helper to check if path is public
const isPublicPath = (path: string) => {
  return publicRoutes.some(route => {
    if (route === '/') return path === '/';
    return path.startsWith(route);
  });
};

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  
  // TEMPORARY: Disable all auth middleware for E2E testing
  // The backend handles auth when enable_auth is true
  return NextResponse.next();
  
  // Check if auth is disabled via environment variable (for E2E tests)
  const authDisabled = process.env.DISABLE_AUTH === 'true' || process.env.NEXT_PUBLIC_DISABLE_AUTH === 'true';
  if (authDisabled) {
    return NextResponse.next();
  }
  
  // Check if the path is public
  if (isPublicPath(pathname)) {
    return NextResponse.next();
  }
  
  // Check if the path is protected
  const isProtected = protectedRoutes.some(route => pathname.startsWith(route));
  if (!isProtected) {
    return NextResponse.next();
  }
  
  // Get token from cookie
  const token = request.cookies.get('auth_token')?.value;
  
  // If no token, check if auth is enabled on backend
  if (!token) {
    // Try to determine if backend has auth enabled
    // In unified container: backend is on port 8080 (same as frontend)
    // In dev: backend might be on different port
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    try {
      const checkResponse = await fetch(`${backendUrl}/api/v1/pipelines`, {
        method: 'HEAD',
        cache: 'no-store',
        signal: AbortSignal.timeout(1000), // 1 second timeout
      });
      
      // If HEAD succeeds (200-299), auth is disabled - allow access
      if (checkResponse.ok || checkResponse.status < 400) {
        return NextResponse.next();
      }
      
      // If 401/403, auth is enabled - redirect to login
      if (checkResponse.status === 401 || checkResponse.status === 403) {
        const loginUrl = new URL('/login', request.url);
        loginUrl.searchParams.set('redirect', pathname);
        return NextResponse.redirect(loginUrl);
      }
      
      // For any other status, fail open and allow access
      return NextResponse.next();
    } catch (error) {
      // On error (timeout, network error, etc.), fail open and allow access
      return NextResponse.next();
    }
  }
  
  // If token exists, validate it with backend
  const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
  try {
    const response = await fetch(`${backendUrl}/api/v1/auth/check`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
      signal: AbortSignal.timeout(1000),
    });
    
    if (!response.ok) {
      // Token is invalid, redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('redirect', pathname);
      return NextResponse.redirect(loginUrl);
    }
  } catch (error) {
    // Continue and let client handle auth state
  }
  
  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public (public files)
     */
    '/((?!api|_next/static|_next/image|favicon.ico|public).*)',
  ],
};
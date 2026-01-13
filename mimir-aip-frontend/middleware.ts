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
  
  // If no token, redirect to login
  if (!token) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }
  
  // Optional: Validate token with backend
  try {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || ''}/api/v1/auth/check`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    });
    
    if (!response.ok) {
      // Token is invalid, redirect to login
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('redirect', pathname);
      return NextResponse.redirect(loginUrl);
    }
  } catch (error) {
    console.warn('Auth check failed:', error);
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
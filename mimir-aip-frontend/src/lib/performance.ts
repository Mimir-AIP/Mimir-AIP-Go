// Performance measurement utilities for comparing Next.js and TanStack implementations

export interface PerformanceMetrics {
  framework: 'nextjs' | 'tanstack';
  bundleSize: {
    js: number;
    css: number;
    total: number;
  };
  renderMetrics: {
    firstContentfulPaint: number;
    largestContentfulPaint: number;
    timeToInteractive: number;
    totalBlockingTime: number;
    cumulativeLayoutShift: number;
  };
  runtimeMetrics: {
    initialLoadTime: number;
    dataFetchTime: number;
    rerenderTime: number;
    memoryUsage: number;
  };
  timestamp: string;
}

export class PerformanceTracker {
  private framework: 'nextjs' | 'tanstack';
  private marks: Map<string, number> = new Map();
  
  constructor(framework: 'nextjs' | 'tanstack') {
    this.framework = framework;
  }

  mark(name: string): void {
    this.marks.set(name, performance.now());
  }

  measure(startMark: string, endMark?: string): number {
    const start = this.marks.get(startMark);
    if (!start) return 0;
    
    const end = endMark ? this.marks.get(endMark) : performance.now();
    if (!end) return 0;
    
    return end - start;
  }

  async collectMetrics(): Promise<Partial<PerformanceMetrics>> {
    const metrics: Partial<PerformanceMetrics> = {
      framework: this.framework,
      timestamp: new Date().toISOString(),
    };

    // Collect Web Vitals if available
    if (typeof window !== 'undefined' && 'performance' in window) {
      const perfEntries = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
      const paintEntries = performance.getEntriesByType('paint');
      
      metrics.renderMetrics = {
        firstContentfulPaint: paintEntries.find(e => e.name === 'first-contentful-paint')?.startTime || 0,
        largestContentfulPaint: 0, // Requires PerformanceObserver
        timeToInteractive: perfEntries?.domInteractive || 0,
        totalBlockingTime: 0,
        cumulativeLayoutShift: 0,
      };

      metrics.runtimeMetrics = {
        initialLoadTime: perfEntries?.loadEventEnd - perfEntries?.fetchStart || 0,
        dataFetchTime: this.measure('data-fetch-start', 'data-fetch-end'),
        rerenderTime: this.measure('rerender-start', 'rerender-end'),
        memoryUsage: (performance as any).memory?.usedJSHeapSize || 0,
      };
    }

    return metrics;
  }

  async getBundleSize(): Promise<{ js: number; css: number; total: number }> {
    // This would typically be measured during build time
    // For runtime, we can estimate based on loaded resources
    if (typeof window !== 'undefined') {
      const resources = performance.getEntriesByType('resource') as PerformanceResourceTiming[];
      
      let jsSize = 0;
      let cssSize = 0;
      
      resources.forEach(resource => {
        if (resource.name.endsWith('.js')) {
          jsSize += resource.transferSize || resource.encodedBodySize || 0;
        } else if (resource.name.endsWith('.css')) {
          cssSize += resource.transferSize || resource.encodedBodySize || 0;
        }
      });
      
      return {
        js: jsSize,
        css: cssSize,
        total: jsSize + cssSize,
      };
    }
    
    return { js: 0, css: 0, total: 0 };
  }

  exportMetrics(): string {
    return JSON.stringify(this.marks, null, 2);
  }
}

// Global performance tracker instance
let trackerInstance: PerformanceTracker | null = null;

export function getPerformanceTracker(framework: 'nextjs' | 'tanstack'): PerformanceTracker {
  if (!trackerInstance) {
    trackerInstance = new PerformanceTracker(framework);
  }
  return trackerInstance;
}

export function saveMetricsToStorage(metrics: Partial<PerformanceMetrics>): void {
  if (typeof window !== 'undefined') {
    const key = `perf_metrics_${metrics.framework}_${Date.now()}`;
    localStorage.setItem(key, JSON.stringify(metrics));
  }
}

export function loadAllMetrics(): PerformanceMetrics[] {
  if (typeof window !== 'undefined') {
    const metrics: PerformanceMetrics[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith('perf_metrics_')) {
        const data = localStorage.getItem(key);
        if (data) {
          try {
            metrics.push(JSON.parse(data));
          } catch (e) {
            console.error('Failed to parse metrics:', e);
          }
        }
      }
    }
    return metrics;
  }
  return [];
}

export function clearAllMetrics(): void {
  if (typeof window !== 'undefined') {
    const keysToRemove: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith('perf_metrics_')) {
        keysToRemove.push(key);
      }
    }
    keysToRemove.forEach(key => localStorage.removeItem(key));
  }
}

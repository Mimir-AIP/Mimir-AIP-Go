import { test as base, expect } from '@playwright/test';
import { execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';

/**
 * E2E test setup and utilities
 */

// Create a test file with test data for CSV processing
export function createTestDataFile(): string {
  const tmpDir = '/tmp';
  const testFile = path.join(tmpDir, 'test-data.csv');
  const csvContent = `name,age,city
Alice,30,NYC
Bob,25,LA
Charlie,35,Chicago`;
  
  fs.writeFileSync(testFile, csvContent);
  return testFile;
}

// Clean up test data
export function cleanupTestData(): void {
  try {
    fs.unlinkSync('/tmp/test-data.csv');
  } catch (e) {
    // File may not exist, that's ok
  }
}

// Wait for server to be ready
export async function waitForServer(page: any, timeout: number = 30000): Promise<void> {
  const startTime = Date.now();
  
  while (Date.now() - startTime < timeout) {
    try {
      await page.goto('/');
      const title = await page.title();
      if (title && !title.includes('error')) {
        return;
      }
    } catch (e) {
      // Retry
    }
    await page.waitForTimeout(1000);
  }
  
  throw new Error('Server did not become ready in time');
}

// Test fixture with setup/teardown
export const test = base.extend({
  page: async ({ page }, use) => {
    // Setup: create test data
    createTestDataFile();
    
    // Use the page
    await use(page);
    
    // Teardown: clean up test data
    cleanupTestData();
  },
});
# Ralph Loop - Agentic E2E Test Fixing

This script implements a **Ralph-Wiggum loop** for autonomously fixing E2E tests and improving the Mimir AIP system through iterative planning, execution, validation, reflection, and adaptation.

## What is a Ralph Loop?

A Ralph loop is an agentic workflow pattern:

```
Plan → Execute → Validate → Reflect → Adapt → Repeat
```

Key properties:
- **Non-interactive execution**: Runs headless using OpenCode
- **Self-correcting**: Agent re-invoked with updated context each iteration
- **Explicit checkpoints**: Validation gates ensure quality
- **Progress commits**: Each successful iteration is committed to git

## How It Works

### Each Iteration Follows These Phases:

#### 1. **PLAN** 
- Analyzes failed E2E tests
- Identifies patterns in failures
- Prioritizes 3-5 fixes to attempt
- Creates action plan

#### 2. **EXECUTE**
- Implements planned fixes via OpenCode
- Makes small, focused changes
- Adds test attributes (data-testid, aria-label)
- Fixes component rendering issues
- Ensures E2E tests interact with UI only (no API calls)

#### 3. **VALIDATE**
- Runs backend Go tests
- Rebuilds frontend and Docker container
- Deploys unified container
- Checks `/health` endpoint
- Runs full Playwright E2E test suite

#### 4. **REFLECT**
- Compares test results vs previous iteration
- Identifies what worked and what didn't
- Analyzes new failures
- Recommends strategy adjustments

#### 5. **ADAPT**
- Every 3rd iteration: Checks for missing test coverage
- Adds new E2E tests for untested pages/features
- Adjusts approach based on learnings

#### 6. **COMMIT**
- Commits progress if tests improved or new tests added
- Documents changes in commit message
- Preserves iteration logs

## Usage

### Basic Usage

```bash
./ralph-loop.sh
```

The script runs continuously until:
- ✅ All E2E tests pass
- ❌ Rate limit hit 5 times (with 30min pauses between)

### What Gets Fixed

The Ralph loop focuses on:

**Frontend Issues:**
- Missing `data-testid` attributes
- Wrong selectors (getByRole, getByText, etc.)
- Missing `aria-label` for accessibility
- Client-side hydration issues
- Missing page headings and metadata

**Test Issues:**
- Tests that call APIs directly (should use UI)
- Missing verification steps after actions
- Timing/timeout issues
- Wrong element selectors

**Coverage Gaps:**
- Pages without E2E tests
- Untested user workflows
- Missing CRUD operation tests

### What Gets Ignored

The loop does NOT:
- Change backend APIs (unless critical)
- Modify test expectations arbitrarily
- Break existing functionality
- Make large architectural changes

## Monitoring Progress

### Real-time Logs

All output is logged to `ralph-logs/`:

```
ralph-logs/
├── ralph.log                          # Main log (all iterations)
├── plan-iter-1.log                    # Planning phase
├── execute-iter-1.log                 # Execution phase
├── e2e-tests-iter-1.log              # E2E test results
├── failed-tests-iter-1.txt           # List of failed tests
├── reflect-iter-1.log                # Reflection analysis
├── backend-tests-iter-1.log          # Backend test results
├── frontend-build-iter-1.log         # Frontend build output
├── docker-build-iter-1.log           # Docker build output
└── docker-deploy-iter-1.log          # Docker deployment output
```

### Watch Progress

```bash
# Watch main log
tail -f ralph-logs/ralph.log

# Watch E2E test results
tail -f ralph-logs/e2e-tests-iter-*.log | grep -E "✓|✘|passed|failed"

# Check current iteration
ls ralph-logs/plan-iter-*.log | wc -l
```

### Check Test Progress

```bash
# Compare test results across iterations
for i in ralph-logs/e2e-failed-iter-*.txt; do
    echo "$(basename $i): $(cat $i) failures"
done
```

## Exit Conditions

### Success Exit
- All E2E tests pass
- Exit code: 0
- Final summary created in `ralph-logs/FINAL_SUMMARY.md`

### Rate Limit Exit
- Hit API rate limit 5 times
- Each rate limit triggers 30min pause
- Exit code: 1

### Manual Interrupt
- User presses Ctrl+C
- Graceful shutdown
- Exit code: 130

## E2E Test Requirements

The Ralph loop enforces these E2E test best practices:

### ✅ Good Test Pattern

```typescript
test('should create a new pipeline', async ({ page }) => {
  // Navigate via UI
  await page.goto('/pipelines');
  
  // Click button
  await page.getByRole('button', { name: 'Create Pipeline' }).click();
  
  // Fill form
  await page.getByLabel('Pipeline Name').fill('Test Pipeline');
  await page.getByLabel('Description').fill('Test Description');
  
  // Submit
  await page.getByRole('button', { name: 'Save' }).click();
  
  // Verify in UI (not API!)
  await expect(page.getByText('Test Pipeline')).toBeVisible();
  await expect(page.getByText('Successfully created')).toBeVisible();
});
```

### ❌ Bad Test Pattern

```typescript
test('should create a new pipeline', async ({ page }) => {
  // BAD: Calling API directly instead of using UI
  const response = await fetch('http://localhost:8080/api/v1/pipelines', {
    method: 'POST',
    body: JSON.stringify({ name: 'Test' })
  });
  
  // BAD: Not verifying the result in the UI
  expect(response.status).toBe(200);
});
```

## Iteration Strategy

### Early Iterations (1-5)
- Focus on low-hanging fruit
- Fix missing test attributes
- Add aria-labels
- Fix obvious component issues

### Mid Iterations (6-15)
- Address complex component interactions
- Fix timing/loading issues
- Improve test selectors
- Add missing tests for major features

### Late Iterations (16+)
- Handle edge cases
- Fine-tune selectors
- Optimize test performance
- Ensure comprehensive coverage

## Troubleshooting

### Script Won't Start

```bash
# Check Docker is running
docker ps

# Check ports are available
lsof -i :8080
lsof -i :3030

# Rebuild manually
npm run build
docker compose -f docker-compose.unified.yml up
```

### Tests Keep Failing

Check the reflection logs:
```bash
cat ralph-logs/reflect-iter-*.log | grep -A 5 "didn't work"
```

### Rate Limiting

The script auto-handles this with 30min pauses. If you need to resume manually:

```bash
# Wait, then re-run
./ralph-loop.sh
```

It will pick up from the last commit.

### OpenCode Issues

```bash
# Check OpenCode is installed
which opencode

# Test OpenCode
opencode run "echo hello"
```

## Architecture

```
ralph-loop.sh
    │
    ├─→ Phase 1: PLAN (via OpenCode)
    │   └─→ Analyze failures → Create fix plan
    │
    ├─→ Phase 2: EXECUTE (via OpenCode)
    │   └─→ Implement fixes → Update code
    │
    ├─→ Phase 3: VALIDATE
    │   ├─→ Run Go tests
    │   ├─→ Build frontend
    │   ├─→ Build Docker
    │   ├─→ Deploy container
    │   ├─→ Check /health
    │   └─→ Run E2E tests
    │
    ├─→ Phase 4: REFLECT (via OpenCode)
    │   └─→ Analyze results → Adjust strategy
    │
    ├─→ Phase 5: ADAPT (via OpenCode, every 3rd iteration)
    │   └─→ Check coverage → Add tests
    │
    └─→ Phase 6: COMMIT
        └─→ Git commit if progress made
```

## Git Integration

Each successful iteration creates a commit:

```
Ralph Loop Iteration 5: E2E test improvements

Iteration 5 results:
- Previous failures: 25
- Current failures: 18
- Tests fixed: 7

Changes made in this iteration are documented in:
- ralph-logs/execute-iter-5.log
- ralph-logs/reflect-iter-5.log
```

View history:
```bash
git log --grep="Ralph Loop"
```

## Configuration

Edit these variables in `ralph-loop.sh`:

```bash
MAX_RATE_LIMIT_PAUSES=5           # Max pauses before exit
RATE_LIMIT_PAUSE_DURATION=1800    # Pause duration (seconds)
```

## Expected Runtime

- **Per iteration**: 5-15 minutes
- **Typical completion**: 10-30 iterations
- **Total time**: 2-8 hours (depending on test count)

The script runs unattended, so you can:
- Run overnight
- Run in tmux/screen session
- Run in CI/CD (with longer timeout)

## Final Output

Upon completion, you'll find:

```
ralph-logs/
└── FINAL_SUMMARY.md    # Complete summary with stats

git log                 # All iterations committed

All tests passing! 🎉
```

## Support

If issues occur:
1. Check `ralph-logs/ralph.log` for errors
2. Review last iteration logs
3. Check Docker container logs: `docker logs mimir-aip-unified`
4. Verify system health: `curl http://localhost:8080/health`

## Philosophy

The Ralph loop embodies "iterative improvement through reflection":

> "The loop doesn't just fix tests—it learns what works, 
> adapts its strategy, and becomes more effective with each iteration."

Each cycle improves both the **code** and the **strategy**, leading to emergent problem-solving behavior.

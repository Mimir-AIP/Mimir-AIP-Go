#!/bin/bash

################################################################################
# Ralph Loop Script - Agentic E2E Test Fixing Workflow
# 
# This script implements a Ralph-Wigum loop for iteratively:
# 1. Planning test fixes
# 2. Executing fixes via OpenCode
# 3. Validating with backend tests + E2E tests
# 4. Reflecting on results
# 5. Adapting strategy
# 6. Repeating until all tests pass
#
# The loop is self-correcting and commits progress after each successful iteration.
################################################################################

set -e  # Exit on error (we'll handle validation failures explicitly)

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/mimir-aip-frontend"
LOG_DIR="$SCRIPT_DIR/ralph-logs"
ITERATION=0
MAX_RATE_LIMIT_PAUSES=5
RATE_LIMIT_PAUSE_COUNT=0
RATE_LIMIT_PAUSE_DURATION=1800  # 30 minutes in seconds
OPENCODE_PORT=4096
OPENCODE_SERVER_PID=""
SKIP_TESTS=false
MAX_ITERATIONS=-1  # -1 means unlimited

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --max-iterations)
            MAX_ITERATIONS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--skip-tests] [--max-iterations N]"
            exit 1
            ;;
    esac
done

# Create log directory
mkdir -p "$LOG_DIR"

################################################################################
# Logging Functions
################################################################################

log() {
    echo -e "${CYAN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_DIR/ralph.log"
}

log_success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✓${NC} $1" | tee -a "$LOG_DIR/ralph.log"
}

log_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ✗${NC} $1" | tee -a "$LOG_DIR/ralph.log"
}

log_warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠${NC} $1" | tee -a "$LOG_DIR/ralph.log"
}

log_section() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}" | tee -a "$LOG_DIR/ralph.log"
    echo -e "${BLUE}  $1${NC}" | tee -a "$LOG_DIR/ralph.log"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n" | tee -a "$LOG_DIR/ralph.log"
}

################################################################################
# OpenCode Server Management
################################################################################

start_opencode_server() {
    log_section "Starting OpenCode Server"
    
    # Check if server is already running
    if curl -s "http://localhost:$OPENCODE_PORT/health" > /dev/null 2>&1; then
        log_warning "OpenCode server already running on port $OPENCODE_PORT"
        return 0
    fi
    
    log "Starting OpenCode server on port $OPENCODE_PORT..."
    opencode serve --port "$OPENCODE_PORT" > "$LOG_DIR/opencode-server.log" 2>&1 &
    OPENCODE_SERVER_PID=$!
    
    log "OpenCode server PID: $OPENCODE_SERVER_PID"
    
    # Wait for server to be ready
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://localhost:$OPENCODE_PORT/health" > /dev/null 2>&1; then
            log_success "OpenCode server is ready"
            return 0
        fi
        
        attempt=$((attempt + 1))
        log "Waiting for OpenCode server... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    log_error "OpenCode server did not start after $max_attempts attempts"
    cat "$LOG_DIR/opencode-server.log"
    return 1
}

stop_opencode_server() {
    log_section "Stopping OpenCode Server"
    
    if [ -n "$OPENCODE_SERVER_PID" ]; then
        log "Stopping OpenCode server (PID: $OPENCODE_SERVER_PID)..."
        kill "$OPENCODE_SERVER_PID" 2>/dev/null || true
        wait "$OPENCODE_SERVER_PID" 2>/dev/null || true
        log_success "OpenCode server stopped"
    fi
}

call_opencode() {
    local prompt="$1"
    local log_file="$2"
    local max_retries=3
    local retry=0
    
    while [ $retry -lt $max_retries ]; do
        log "Calling OpenCode (attempt $((retry + 1))/$max_retries)..."
        
        local output
        local exit_code
        
        if [ -n "$log_file" ]; then
            output=$(opencode run --attach "http://localhost:$OPENCODE_PORT" "$prompt" 2>&1 | tee "$log_file")
            exit_code=${PIPESTATUS[0]}
        else
            output=$(opencode run --attach "http://localhost:$OPENCODE_PORT" "$prompt" 2>&1)
            exit_code=$?
        fi
        
        # Check exit code
        if [ $exit_code -eq 0 ]; then
            log_success "OpenCode call succeeded"
            return 0
        elif [ $exit_code -eq 2 ]; then
            # Rate limiting detected (based on OpenCode docs return code)
            log_warning "Rate limiting detected (exit code 2)"
            handle_rate_limiting
            retry=$((retry + 1))
            continue
        else
            # Check output for rate limiting messages
            if check_for_rate_limiting "$output"; then
                log_warning "Rate limiting detected in output"
                handle_rate_limiting
                retry=$((retry + 1))
                continue
            fi
            
            log_error "OpenCode call failed (exit code $exit_code)"
            return 1
        fi
    done
    
    log_error "OpenCode call failed after $max_retries retries"
    return 1
}

################################################################################
# Validation Functions
################################################################################

run_backend_tests() {
    if [ "$SKIP_TESTS" = true ]; then
        log_warning "SKIP_TESTS mode: Skipping backend tests"
        return 0
    fi
    
    log_section "VALIDATION: Running Backend Tests"
    
    cd "$SCRIPT_DIR"
    
    log "Running Go tests..."
    if go test ./... -v 2>&1 | tee "$LOG_DIR/backend-tests-iter-$ITERATION.log"; then
        log_success "Backend tests passed"
        return 0
    else
        log_error "Backend tests failed"
        return 1
    fi
}

build_and_deploy_docker() {
    if [ "$SKIP_TESTS" = true ]; then
        log_warning "SKIP_TESTS mode: Skipping Docker build/deploy"
        return 0
    fi
    
    log_section "VALIDATION: Building and Deploying Docker Container"
    
    cd "$SCRIPT_DIR"
    
    # Build frontend first
    log "Building frontend..."
    cd "$FRONTEND_DIR"
    if ! npm run build 2>&1 | tee "$LOG_DIR/frontend-build-iter-$ITERATION.log"; then
        log_error "Frontend build failed"
        return 1
    fi
    log_success "Frontend built successfully"
    
    cd "$SCRIPT_DIR"
    
    # Build Docker image
    log "Building Docker image..."
    if ! docker build -f Dockerfile.unified -t mimir-aip:unified . 2>&1 | tee "$LOG_DIR/docker-build-iter-$ITERATION.log"; then
        log_error "Docker build failed"
        return 1
    fi
    docker tag mimir-aip:unified mimir-aip:unified-latest
    log_success "Docker image built successfully"
    
    # Deploy container
    log "Deploying container..."
    docker compose -f docker-compose.unified.yml down 2>&1 | tee -a "$LOG_DIR/docker-deploy-iter-$ITERATION.log"
    if ! docker compose -f docker-compose.unified.yml up -d 2>&1 | tee -a "$LOG_DIR/docker-deploy-iter-$ITERATION.log"; then
        log_error "Docker deployment failed"
        return 1
    fi
    log_success "Container deployed successfully"
    
    # Wait for container to be ready
    log "Waiting for container to be ready..."
    sleep 10
    
    return 0
}

check_health_endpoint() {
    if [ "$SKIP_TESTS" = true ]; then
        log_warning "SKIP_TESTS mode: Skipping health check"
        return 0
    fi
    
    log_section "VALIDATION: Checking Health Endpoint"
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            log_success "Health endpoint is responding"
            return 0
        fi
        
        attempt=$((attempt + 1))
        log "Waiting for health endpoint... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    log_error "Health endpoint did not respond after $max_attempts attempts"
    return 1
}

run_e2e_tests() {
    if [ "$SKIP_TESTS" = true ]; then
        log_warning "SKIP_TESTS mode: Creating fake test results"
        # Create fake test results for testing the loop
        local fake_passed=$((400 + ITERATION * 5))
        local fake_failed=$((20 - ITERATION * 2))
        if [ $fake_failed -lt 0 ]; then
            fake_failed=0
        fi
        
        echo "$fake_passed" > "$LOG_DIR/e2e-passed-iter-$ITERATION.txt"
        echo "$fake_failed" > "$LOG_DIR/e2e-failed-iter-$ITERATION.txt"
        
        # Create fake log with some failed tests
        cat > "$LOG_DIR/e2e-tests-iter-$ITERATION.log" << EOF
Running $((fake_passed + fake_failed)) tests
✓ Test 1 passed
✓ Test 2 passed
EOF
        
        for i in $(seq 1 $fake_failed); do
            echo "✘ Fake test $i failed" >> "$LOG_DIR/e2e-tests-iter-$ITERATION.log"
        done
        
        log_success "Fake E2E tests: $fake_passed passed, $fake_failed failed"
        
        if [ "$fake_failed" -eq 0 ]; then
            return 0
        else
            return 1
        fi
    fi
    
    log_section "VALIDATION: Running E2E Tests"
    
    cd "$FRONTEND_DIR"
    
    log "Running Playwright E2E tests..."
    if npx playwright test --reporter=list 2>&1 | tee "$LOG_DIR/e2e-tests-iter-$ITERATION.log"; then
        local passed=$(grep -c "✓" "$LOG_DIR/e2e-tests-iter-$ITERATION.log" || echo 0)
        local failed=$(grep -c "✘" "$LOG_DIR/e2e-tests-iter-$ITERATION.log" || echo 0)
        log_success "E2E tests completed: $passed passed, $failed failed"
        
        # Save test summary
        echo "$passed" > "$LOG_DIR/e2e-passed-iter-$ITERATION.txt"
        echo "$failed" > "$LOG_DIR/e2e-failed-iter-$ITERATION.txt"
        
        if [ "$failed" -eq 0 ]; then
            log_success "All E2E tests passed!"
            return 0
        else
            log_warning "$failed E2E tests still failing"
            return 1
        fi
    else
        log_error "E2E test execution failed"
        return 1
    fi
}

analyze_test_failures() {
    log_section "ANALYSIS: Analyzing Test Failures"
    
    local log_file="$LOG_DIR/e2e-tests-iter-$ITERATION.log"
    
    if [ ! -f "$log_file" ]; then
        log_error "No test log found for analysis"
        return 1
    fi
    
    # Extract failed test names
    grep "✘" "$log_file" | head -20 > "$LOG_DIR/failed-tests-iter-$ITERATION.txt"
    
    local failed_count=$(wc -l < "$LOG_DIR/failed-tests-iter-$ITERATION.txt")
    log "Found $failed_count failed tests"
    
    # Show sample of failures
    if [ "$failed_count" -gt 0 ]; then
        log "Sample of failed tests:"
        head -5 "$LOG_DIR/failed-tests-iter-$ITERATION.txt" | while read -r line; do
            log "  - $line"
        done
    fi
    
    return 0
}

check_for_rate_limiting() {
    local output="$1"
    
    if echo "$output" | grep -qi "rate limit\|too many requests\|429\|quota exceeded"; then
        return 0  # Rate limiting detected
    fi
    return 1  # No rate limiting
}

handle_rate_limiting() {
    RATE_LIMIT_PAUSE_COUNT=$((RATE_LIMIT_PAUSE_COUNT + 1))
    
    if [ $RATE_LIMIT_PAUSE_COUNT -ge $MAX_RATE_LIMIT_PAUSES ]; then
        log_error "Hit rate limit $MAX_RATE_LIMIT_PAUSES times. Exiting."
        stop_opencode_server
        exit 1
    fi
    
    log_warning "Rate limiting detected (pause $RATE_LIMIT_PAUSE_COUNT/$MAX_RATE_LIMIT_PAUSES)"
    log "Pausing for $RATE_LIMIT_PAUSE_DURATION seconds (30 minutes)..."
    
    sleep $RATE_LIMIT_PAUSE_DURATION
    
    log "Resuming after rate limit pause..."
}

################################################################################
# Git Functions
################################################################################

commit_progress() {
    local message="$1"
    
    log_section "GIT: Committing Progress"
    
    cd "$SCRIPT_DIR"
    
    # Check if there are changes to commit
    if git diff --quiet && git diff --cached --quiet; then
        log "No changes to commit"
        return 0
    fi
    
    log "Staging changes..."
    git add -A
    
    log "Creating commit..."
    if git commit -m "$message" 2>&1 | tee "$LOG_DIR/git-commit-iter-$ITERATION.log"; then
        log_success "Changes committed: $message"
        return 0
    else
        log_warning "Commit failed or no changes"
        return 1
    fi
}

################################################################################
# OpenCode Integration Functions
################################################################################

plan_fixes() {
    log_section "PLAN: Analyzing System and Planning Fixes"
    
    local prompt="Analyze the current E2E test failures and system state:

1. Review the failed tests in ralph-logs/failed-tests-iter-$ITERATION.txt
2. Review the full E2E test log in ralph-logs/e2e-tests-iter-$ITERATION.log
3. Identify patterns in failures (missing elements, timeouts, wrong selectors, etc.)
4. Prioritize fixes based on:
   - Number of tests affected
   - Ease of fix
   - Dependencies between tests
5. Create a prioritized list of 3-5 fixes to attempt this iteration

Focus on:
- Frontend component issues (missing test IDs, wrong selectors, accessibility labels)
- Page structure issues (missing headings, wrong metadata)
- Interaction issues (buttons not clickable, forms not submitting)
- Timing issues (elements loading slowly)

DO NOT:
- Make backend API changes unless absolutely necessary
- Modify test expectations unless they're clearly wrong
- Change core functionality

Respond with a clear, numbered plan of fixes to attempt."

    if ! call_opencode "$prompt" "$LOG_DIR/plan-iter-$ITERATION.log"; then
        log_error "Planning phase failed"
        return 1
    fi
    
    log_success "Plan created"
    return 0
}

execute_fixes() {
    log_section "EXECUTE: Implementing Fixes via OpenCode"
    
    local prompt="Based on the plan in ralph-logs/plan-iter-$ITERATION.log, implement the fixes:

CRITICAL RULES:
1. E2E tests must interact with the FRONTEND ONLY
   - Use getByRole, getByTestId, getByPlaceholder, getByText
   - Click buttons, fill forms, navigate pages
   - DO NOT call APIs directly in tests
   
2. After each action, verify the result in the UI
   - After creating something, check it appears in the list
   - After updating, verify the new value is displayed
   - After deleting, confirm it's gone from the UI
   
3. Add missing test attributes to components:
   - data-testid for key elements
   - aria-label for icon-only buttons
   - proper role attributes for accessibility
   
4. Fix component rendering issues:
   - Add missing headings
   - Fix metadata for page titles
   - Ensure elements render on initial load (no client-side hydration issues)

5. Each fix should be small and focused
   - Fix one component at a time
   - Test after each change
   - Don't break existing functionality

Start with the highest priority fixes from the plan. Implement 2-3 fixes this iteration.

When done, summarize what was changed and what tests should now pass."

    if ! call_opencode "$prompt" "$LOG_DIR/execute-iter-$ITERATION.log"; then
        log_error "Execution phase failed"
        return 1
    fi
    
    log_success "Fixes executed"
    return 0
}

reflect_on_results() {
    log_section "REFLECT: Analyzing Results and Adapting Strategy"
    
    local prev_failed=0
    local curr_failed=0
    
    if [ -f "$LOG_DIR/e2e-failed-iter-$((ITERATION-1)).txt" ]; then
        prev_failed=$(cat "$LOG_DIR/e2e-failed-iter-$((ITERATION-1)).txt")
    fi
    
    if [ -f "$LOG_DIR/e2e-failed-iter-$ITERATION.txt" ]; then
        curr_failed=$(cat "$LOG_DIR/e2e-failed-iter-$ITERATION.txt")
    fi
    
    local prompt="Reflect on the results of iteration $ITERATION:

Previous failed tests: $prev_failed
Current failed tests: $curr_failed

Review:
1. The fixes made in ralph-logs/execute-iter-$ITERATION.log
2. The test results in ralph-logs/e2e-tests-iter-$ITERATION.log
3. Any new failures that appeared
4. Which fixes worked and which didn't

Analysis questions:
- Did we make progress (fewer failures)?
- Did we introduce new failures?
- Are there patterns in remaining failures?
- Should we change our approach?
- What should we prioritize next?

Provide:
1. A brief summary of what worked
2. A brief summary of what didn't work
3. Recommendations for the next iteration
4. Any adjustments to our strategy

Keep response concise and actionable."

    if ! call_opencode "$prompt" "$LOG_DIR/reflect-iter-$ITERATION.log"; then
        log_error "Reflection phase failed"
        return 1
    fi
    
    log_success "Reflection complete"
    
    # Check if we made progress
    if [ "$curr_failed" -lt "$prev_failed" ]; then
        log_success "Progress made! Failures reduced from $prev_failed to $curr_failed"
        return 0
    elif [ "$curr_failed" -eq 0 ]; then
        log_success "All tests passing!"
        return 0
    else
        log_warning "No progress this iteration ($curr_failed failures remain)"
        return 1
    fi
}

add_new_tests_if_needed() {
    log_section "ADAPT: Adding Missing E2E Tests"
    
    local prompt="Review the current E2E test coverage:

1. Look at all test files in mimir-aip-frontend/e2e/
2. Look at all pages in mimir-aip-frontend/src/app/
3. Identify pages/features that lack E2E tests

For each missing test, create comprehensive E2E tests that:
- Test all major user workflows on that page
- Click all buttons and verify their effects
- Fill all forms and verify submissions work
- Navigate to other pages and verify navigation
- Check that lists populate correctly
- Verify CRUD operations reflect in the UI

IMPORTANT: Tests must interact with UI only (no direct API calls)

Add tests for the top 2-3 most important missing areas.

When done, summarize what tests were added."

    if ! call_opencode "$prompt" "$LOG_DIR/add-tests-iter-$ITERATION.log"; then
        log_error "Test coverage review failed"
        return 1
    fi
    
    log_success "Test coverage review complete"
    return 0
}

################################################################################
# Main Ralph Loop
################################################################################

ralph_loop_iteration() {
    ITERATION=$((ITERATION + 1))
    
    # Check max iterations
    if [ $MAX_ITERATIONS -gt 0 ] && [ $ITERATION -gt $MAX_ITERATIONS ]; then
        log_warning "Reached max iterations ($MAX_ITERATIONS)"
        return 0
    fi
    
    log_section "RALPH LOOP - ITERATION $ITERATION"
    
    # PHASE 1: PLAN
    if ! plan_fixes; then
        log_error "Planning phase failed"
        return 1
    fi
    
    # PHASE 2: EXECUTE
    if ! execute_fixes; then
        log_error "Execution phase failed"
        return 1
    fi
    
    # PHASE 3: VALIDATE
    log_section "VALIDATION PHASE"
    
    # 3.1: Backend tests
    if ! run_backend_tests; then
        log_warning "Backend tests failed, but continuing with deployment"
        # Don't return - we still want to test frontend
    fi
    
    # 3.2: Build and deploy
    if ! build_and_deploy_docker; then
        log_error "Build/deploy failed - cannot continue validation"
        return 1
    fi
    
    # 3.3: Check health
    if ! check_health_endpoint; then
        log_error "Health check failed - cannot run E2E tests"
        return 1
    fi
    
    # 3.4: Run E2E tests
    local e2e_result=0
    run_e2e_tests || e2e_result=$?
    
    analyze_test_failures
    
    # PHASE 4: REFLECT
    local made_progress=0
    reflect_on_results || made_progress=$?
    
    # PHASE 5: ADAPT
    # Every 3 iterations, check if we need new tests
    if [ $((ITERATION % 3)) -eq 0 ]; then
        add_new_tests_if_needed
    fi
    
    # PHASE 6: COMMIT PROGRESS
    local curr_failed=0
    if [ -f "$LOG_DIR/e2e-failed-iter-$ITERATION.txt" ]; then
        curr_failed=$(cat "$LOG_DIR/e2e-failed-iter-$ITERATION.txt")
    fi
    
    local prev_failed=999999
    if [ -f "$LOG_DIR/e2e-failed-iter-$((ITERATION-1)).txt" ]; then
        prev_failed=$(cat "$LOG_DIR/e2e-failed-iter-$((ITERATION-1)).txt")
    fi
    
    # Commit if we made progress OR if we added new tests
    if [ "$curr_failed" -lt "$prev_failed" ] || [ "$e2e_result" -eq 0 ] || [ $((ITERATION % 3)) -eq 0 ]; then
        local commit_msg="Ralph Loop Iteration $ITERATION: E2E test improvements

Iteration $ITERATION results:
- Previous failures: $prev_failed
- Current failures: $curr_failed
- Tests fixed: $((prev_failed - curr_failed))

Changes made in this iteration are documented in:
- ralph-logs/execute-iter-$ITERATION.log
- ralph-logs/reflect-iter-$ITERATION.log"
        
        commit_progress "$commit_msg"
    else
        log_warning "No progress made - not committing this iteration"
    fi
    
    # Check exit condition
    if [ "$curr_failed" -eq 0 ]; then
        log_success "ALL E2E TESTS PASSING! Ralph loop complete."
        return 0
    fi
    
    return 1  # Continue looping
}

main() {
    log_section "RALPH LOOP STARTING"
    log "Working directory: $SCRIPT_DIR"
    log "Frontend directory: $FRONTEND_DIR"
    log "Log directory: $LOG_DIR"
    
    if [ "$SKIP_TESTS" = true ]; then
        log_warning "SKIP_TESTS mode enabled - using fake test results"
    fi
    
    if [ $MAX_ITERATIONS -gt 0 ]; then
        log_warning "Max iterations set to: $MAX_ITERATIONS"
    fi
    
    # Start OpenCode server
    if ! start_opencode_server; then
        log_error "Failed to start OpenCode server"
        exit 1
    fi
    
    # Initial validation to make sure system is working
    log_section "INITIAL SYSTEM CHECK"
    
    if ! build_and_deploy_docker; then
        log_error "Initial deployment failed"
        stop_opencode_server
        exit 1
    fi
    
    if ! check_health_endpoint; then
        log_error "Initial health check failed"
        stop_opencode_server
        exit 1
    fi
    
    # Get baseline test results
    log "Getting baseline test results..."
    run_e2e_tests || true  # Don't fail if tests fail initially
    analyze_test_failures
    
    local baseline_failed=0
    if [ -f "$LOG_DIR/e2e-failed-iter-$ITERATION.txt" ]; then
        baseline_failed=$(cat "$LOG_DIR/e2e-failed-iter-$ITERATION.txt")
    fi
    
    log_warning "Baseline: $baseline_failed E2E tests failing"
    
    # Main loop
    while true; do
        if ralph_loop_iteration; then
            # Check if we stopped due to max iterations
            if [ $MAX_ITERATIONS -gt 0 ] && [ $ITERATION -ge $MAX_ITERATIONS ]; then
                log_warning "Stopped at max iterations ($MAX_ITERATIONS)"
                break
            fi
            
            log_success "Ralph loop completed successfully!"
            break
        fi
        
        # Check if we should continue
        local curr_failed=0
        if [ -f "$LOG_DIR/e2e-failed-iter-$ITERATION.txt" ]; then
            curr_failed=$(cat "$LOG_DIR/e2e-failed-iter-$ITERATION.txt")
        fi
        
        if [ "$curr_failed" -eq 0 ]; then
            break
        fi
        
        log "Continuing to next iteration..."
        sleep 5
    done
    
    # Stop OpenCode server
    stop_opencode_server
    
    log_section "RALPH LOOP COMPLETE"
    
    local final_failed=0
    if [ -f "$LOG_DIR/e2e-failed-iter-$ITERATION.txt" ]; then
        final_failed=$(cat "$LOG_DIR/e2e-failed-iter-$ITERATION.txt")
    fi
    
    if [ "$final_failed" -eq 0 ]; then
        log_success "All E2E tests are now passing!"
    else
        log_warning "Completed with $final_failed tests still failing"
    fi
    
    log "Total iterations: $ITERATION"
    log "Rate limit pauses: $RATE_LIMIT_PAUSE_COUNT"
    
    # Create final summary
    cat > "$LOG_DIR/FINAL_SUMMARY.md" << EOF
# Ralph Loop Final Summary

## Results
- **Total Iterations**: $ITERATION
- **Initial Failed Tests**: $baseline_failed
- **Final Failed Tests**: $final_failed
- **Tests Fixed**: $((baseline_failed - final_failed))
- **Rate Limit Pauses**: $RATE_LIMIT_PAUSE_COUNT

## Configuration
- **Skip Tests Mode**: $SKIP_TESTS
- **Max Iterations**: $MAX_ITERATIONS

## Timeline
- **Started**: $(head -1 "$LOG_DIR/ralph.log" | cut -d']' -f1 | tr -d '[')
- **Completed**: $(date +'%Y-%m-%d %H:%M:%S')

## Log Files
All iteration logs are available in: $LOG_DIR/

## Commits
All progress was committed to the local git repository.
Use \`git log\` to see the full history of changes.

EOF
    
    log "Final summary written to: $LOG_DIR/FINAL_SUMMARY.md"
}

# Handle interrupts gracefully
trap 'log_error "Ralph loop interrupted"; stop_opencode_server; exit 130' INT TERM

# Run main function
main "$@"

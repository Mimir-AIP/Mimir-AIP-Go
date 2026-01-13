#!/bin/bash

# Start both Next.js and TanStack implementations for performance comparison

echo "🚀 Starting Mimir AIP Performance Comparison"
echo "============================================"
echo ""
echo "📦 Next.js will run on http://localhost:3000"
echo "📦 TanStack will run on http://localhost:3001"
echo ""
echo "Press Ctrl+C to stop both servers"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping servers..."
    kill $NEXTJS_PID $TANSTACK_PID 2>/dev/null
    exit 0
}

trap cleanup EXIT INT TERM

# Start Next.js in background
cd mimir-aip-frontend
echo "▶️  Starting Next.js..."
npm run dev > /tmp/nextjs.log 2>&1 &
NEXTJS_PID=$!

# Start TanStack in background
cd ../mimir-aip-tanstack
echo "▶️  Starting TanStack..."
npm run dev > /tmp/tanstack.log 2>&1 &
TANSTACK_PID=$!

cd ..

echo ""
echo "✅ Both servers started!"
echo ""
echo "To view logs:"
echo "  - Next.js:   tail -f /tmp/nextjs.log"
echo "  - TanStack:  tail -f /tmp/tanstack.log"
echo ""
echo "📊 Visit /dashboard on both to collect metrics"
echo "📈 Visit /performance to compare results"
echo ""

# Wait for both processes
wait

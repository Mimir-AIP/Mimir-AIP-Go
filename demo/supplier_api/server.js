const express = require('express');
const cors = require('cors');
const fs = require('fs');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 8081;

app.use(cors());
app.use(express.json());

// Load initial data
let supplierData = JSON.parse(fs.readFileSync(path.join(__dirname, 'supplier_pricing_api.json'), 'utf8'));

// Simulate price changes for drift detection demo
function simulatePriceChanges() {
  const parts = ['PART-0001', 'PART-0002', 'PART-0017', 'PART-0021', 'PART-0009', 'PART-0010'];
  
  supplierData.suppliers.forEach(supplier => {
    supplier.parts.forEach(part => {
      // Random price fluctuation between -10% and +10%
      const change = (Math.random() - 0.5) * 20;
      part.price_change_pct = parseFloat(change.toFixed(2));
      
      // Occasionally change availability
      if (Math.random() > 0.8) {
        const statuses = ['in_stock', 'low_stock', 'backorder'];
        part.availability = statuses[Math.floor(Math.random() * statuses.length)];
      }
      
      // Update timestamp
      part.last_updated = new Date().toISOString();
    });
    supplier.last_updated = new Date().toISOString();
  });
  
  supplierData.metadata.response_time_ms = Math.floor(Math.random() * 50) + 20;
  supplierData.metadata.next_update = new Date(Date.now() + 3600000).toISOString();
  
  console.log(`[${new Date().toISOString()}] Simulated price changes`);
}

// API Routes

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// Get all supplier pricing
app.get('/api/v1/suppliers/pricing', (req, res) => {
  const startTime = Date.now();
  
  // Simulate processing time
  setTimeout(() => {
    const response = {
      ...supplierData,
      metadata: {
        ...supplierData.metadata,
        response_time_ms: Date.now() - startTime
      }
    };
    
    res.json(response);
    console.log(`[${new Date().toISOString()}] Served pricing data - ${supplierData.metadata.total_parts} parts`);
  }, 20);
});

// Get specific supplier
app.get('/api/v1/suppliers/:id', (req, res) => {
  const supplier = supplierData.suppliers.find(s => s.supplier_id === req.params.id);
  if (supplier) {
    res.json(supplier);
  } else {
    res.status(404).json({ error: 'Supplier not found' });
  }
});

// Get specific part
app.get('/api/v1/parts/:id', (req, res) => {
  let part = null;
  supplierData.suppliers.forEach(s => {
    const found = s.parts.find(p => p.part_id === req.params.id);
    if (found) {
      part = { ...found, supplier_id: s.supplier_id, supplier_name: s.supplier_name };
    }
  });
  
  if (part) {
    res.json(part);
  } else {
    res.status(404).json({ error: 'Part not found' });
  }
});

// Webhook to receive drift notifications
app.post('/api/v1/webhooks/drift', (req, res) => {
  console.log(`[${new Date().toISOString()}] Drift notification received:`, req.body);
  res.json({ received: true });
});

// Simulate endpoint for demo purposes
app.post('/api/v1/simulate', (req, res) => {
  simulatePriceChanges();
  res.json({ 
    message: 'Price simulation completed',
    timestamp: new Date().toISOString(),
    changes: supplierData.suppliers.flatMap(s => s.parts.map(p => ({
      part_id: p.part_id,
      price_change_pct: p.price_change_pct,
      availability: p.availability
    })))
  });
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`🚀 Supplier Pricing API running on port ${PORT}`);
  console.log(`📊 Serving ${supplierData.metadata.total_parts} parts from ${supplierData.metadata.total_suppliers} suppliers`);
  console.log(`⏰ Price updates simulated every hour`);
});

// Simulate price changes every hour
setInterval(simulatePriceChanges, 3600000);

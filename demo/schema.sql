-- Demo: Computer Repair Shop Database Schema
-- Full relational database for realistic Mimir demo

-- Parts inventory with stock tracking
CREATE TABLE IF NOT EXISTS parts_inventory (
    part_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    current_stock INTEGER NOT NULL DEFAULT 0,
    min_stock INTEGER NOT NULL DEFAULT 5,
    reorder_point INTEGER NOT NULL DEFAULT 10,
    unit_cost DECIMAL(10,2) NOT NULL,
    supplier_id VARCHAR(50) NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    location VARCHAR(100),
    barcode VARCHAR(100)
);

-- Suppliers with API endpoints
CREATE TABLE IF NOT EXISTS suppliers (
    supplier_id VARCHAR(50) PRIMARY KEY,
    supplier_name VARCHAR(255) NOT NULL,
    api_endpoint VARCHAR(500),
    api_key VARCHAR(255),
    lead_time_days INTEGER DEFAULT 3,
    minimum_order INTEGER DEFAULT 1,
    reliability_score DECIMAL(3,2) DEFAULT 0.95,
    last_price_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    contact_email VARCHAR(255),
    phone VARCHAR(50)
);

-- Real-time pricing from supplier APIs
CREATE TABLE IF NOT EXISTS supplier_pricing (
    pricing_id INTEGER PRIMARY KEY AUTOINCREMENT,
    supplier_id VARCHAR(50) NOT NULL,
    part_id VARCHAR(50) NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    lead_time_days INTEGER NOT NULL,
    minimum_order INTEGER NOT NULL,
    price_change_pct DECIMAL(5,2) DEFAULT 0.00,
    availability_status VARCHAR(20) DEFAULT 'in_stock',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (supplier_id) REFERENCES suppliers(supplier_id),
    FOREIGN KEY (part_id) REFERENCES parts_inventory(part_id)
);

-- Repair jobs linking customers, devices, and parts
CREATE TABLE IF NOT EXISTS repair_jobs (
    job_id VARCHAR(50) PRIMARY KEY,
    customer_name VARCHAR(255) NOT NULL,
    customer_email VARCHAR(255),
    customer_phone VARCHAR(50),
    device_type VARCHAR(100) NOT NULL,
    device_model VARCHAR(255),
    problem_description TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    total_cost DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    assigned_technician VARCHAR(100)
);

-- Parts used in each repair job
CREATE TABLE IF NOT EXISTS job_parts (
    job_part_id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id VARCHAR(50) NOT NULL,
    part_id VARCHAR(50) NOT NULL,
    quantity_used INTEGER NOT NULL,
    unit_price_at_time DECIMAL(10,2) NOT NULL,
    FOREIGN KEY (job_id) REFERENCES repair_jobs(job_id),
    FOREIGN KEY (part_id) REFERENCES parts_inventory(part_id)
);

-- Stock movement history for ML training
CREATE TABLE IF NOT EXISTS stock_movements (
    movement_id INTEGER PRIMARY KEY AUTOINCREMENT,
    part_id VARCHAR(50) NOT NULL,
    movement_type VARCHAR(50) NOT NULL, -- 'sale', 'restock', 'adjustment', 'return'
    quantity_change INTEGER NOT NULL,
    stock_before INTEGER NOT NULL,
    stock_after INTEGER NOT NULL,
    reference_id VARCHAR(100), -- job_id or supplier_order_id
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (part_id) REFERENCES parts_inventory(part_id)
);

-- Sales transactions
CREATE TABLE IF NOT EXISTS sales (
    sale_id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id VARCHAR(50),
    sale_date DATE NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    payment_method VARCHAR(50),
    FOREIGN KEY (job_id) REFERENCES repair_jobs(job_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_parts_category ON parts_inventory(category);
CREATE INDEX IF NOT EXISTS idx_parts_supplier ON parts_inventory(supplier_id);
CREATE INDEX IF NOT EXISTS idx_parts_stock ON parts_inventory(current_stock);
CREATE INDEX IF NOT EXISTS idx_pricing_supplier ON supplier_pricing(supplier_id);
CREATE INDEX IF NOT EXISTS idx_pricing_part ON supplier_pricing(part_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON repair_jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_date ON repair_jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_movements_part ON stock_movements(part_id);
CREATE INDEX IF NOT EXISTS idx_movements_date ON stock_movements(created_at);

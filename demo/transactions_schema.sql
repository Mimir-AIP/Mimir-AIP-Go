-- Sales transactions for repair shop
CREATE TABLE IF NOT EXISTS sales_transactions (
    transaction_id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id VARCHAR(50),
    part_id VARCHAR(50),
    part_name VARCHAR(255),
    quantity INTEGER,
    unit_price DECIMAL(10,2),
    total_amount DECIMAL(10,2),
    customer_name VARCHAR(255),
    transaction_date DATE,
    payment_method VARCHAR(50),
    technician VARCHAR(100),
    device_type VARCHAR(100),
    margin DECIMAL(5,2)
);

-- Insert sample transactions
INSERT INTO sales_transactions (
    job_id, part_id, part_name, quantity, unit_price, total_amount,
    customer_name, transaction_date, payment_method, technician, device_type, margin
) VALUES
('JOB-001', 'PART-0001', 'Intel Core i9-13900K', 1, 589.99, 589.99, 
 'John Smith', '2026-01-31', 'credit', 'Alex Johnson', 'Gaming PC', 0.25),
('JOB-002', 'PART-0017', 'Corsair Vengeance DDR5 32GB', 2, 189.99, 379.98,
 'Sarah Wilson', '2026-01-31', 'credit', 'Maria Garcia', 'Workstation', 0.30),
('JOB-003', 'PART-0009', 'NVIDIA GeForce RTX 4090', 1, 1599.99, 1599.99,
 'Mike Chen', '2026-01-30', 'debit', 'David Kim', 'Gaming PC', 0.20),
('JOB-004', 'PART-0021', 'Samsung 990 Pro 2TB NVMe', 1, 249.99, 249.99,
 'Emma Davis', '2026-01-30', 'cash', 'Jessica Martinez', 'Laptop', 0.35),
('JOB-005', 'PART-0002', 'AMD Ryzen 9 7950X', 1, 549.99, 549.99,
 'Robert Brown', '2026-01-29', 'credit', 'James Wilson', 'Desktop', 0.28);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_transactions_date ON sales_transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_part ON sales_transactions(part_id);
CREATE INDEX IF NOT EXISTS idx_transactions_job ON sales_transactions(job_id);

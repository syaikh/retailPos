-- Seed: 014_customers.sql
-- Idempotent seed for customers master data

INSERT INTO customers (id, name, phone, email, address, note, is_walk_in, is_active)
VALUES
    (1, 'Pelanggan Umum / Walk-in', NULL, NULL, NULL, NULL, true, true),
    (2, 'Ahmad Fauzi', '081234567890', 'ahmad.fauzi@email.com', 'Jl. Merdeka No. 10, Jakarta Pusat', 'Pelanggan tetap', false, true),
    (3, 'Siti Nurhaliza', '082345678901', 'siti.nur@email.com', 'Jl. Sudirman No. 25, Bandung', NULL, false, true),
    (4, 'Budi Santoso', '083456789012', 'budi.santoso@email.com', 'Jl. Diponegoro No. 5, Surabaya', 'Member premium', false, true),
    (5, 'Dewi Lestari', '084567890123', 'dewi.lestari@email.com', 'Jl. Anggrek No. 88, Jakarta Selatan', NULL, false, true),
    (6, 'Eko Prasetyo', '085678901234', 'eko.prasetyo@email.com', 'Jl. Pemuda No. 12, Semarang', 'Pembayaran tunai', false, true),
    (7, 'Rina Wijaya', '086789012345', 'rina.wijaya@email.com', 'Jl. Kartini No. 7, Yogyakarta', 'Pernah komplain', false, true),
    (8, 'Hendra Gunawan', '087890123456', 'hendra.g@email.com', 'Jl. Gajah Mada No. 42, Medan', NULL, false, true),
    (9, 'Maya Anggraini', '088901234567', 'maya.anggraini@email.com', 'Jl. Mawar No. 15, Makassar', 'Alergi seafood', false, true),
    (10, 'Rizky Pratama', '089012345678', 'rizky.pratama@email.com', 'Jl. Veteran No. 33, Denpasar', 'Request packaging khusus', false, true)
ON CONFLICT (id) DO NOTHING;

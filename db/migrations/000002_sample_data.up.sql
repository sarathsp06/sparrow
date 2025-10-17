-- Add sample data for testing and development
INSERT INTO webhook_registrations (id, namespace, events, url, headers, timeout, active, description) VALUES
('webhook-sample-1', 'user', '["signup", "login", "logout"]', 'https://httpbin.org/post', '{"Authorization": "Bearer token1", "X-Service": "HTTPQueue"}', 30, true, 'Sample user activity webhook'),
('webhook-sample-2', 'order', '["created", "updated", "cancelled"]', 'https://httpbin.org/post', '{"X-Service": "OrderProcessor", "Content-Type": "application/json"}', 30, true, 'Sample order lifecycle webhook'),
('webhook-sample-3', 'payment', '["processed", "failed", "refunded"]', 'https://httpbin.org/post', '{"X-Secret": "payment-secret", "X-Service": "PaymentProcessor"}', 15, true, 'Sample payment processing webhook');
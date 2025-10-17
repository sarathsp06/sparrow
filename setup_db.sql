-- Database setup for River Queue
-- Run this script to create the necessary database and tables

-- Create database if it doesn't exist
-- Note: You may need to run this as a superuser
-- CREATE DATABASE riverqueue;

-- Connect to the riverqueue database and run the following:
-- River will automatically create its tables when the client starts
-- But you can also create them manually using the River migration tool

-- Example tables that your application might need:
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert some sample data
INSERT INTO users (email) VALUES 
    ('user1@example.com'),
    ('user2@example.com'),
    ('user3@example.com')
ON CONFLICT (email) DO NOTHING;
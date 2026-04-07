-- 000019: Drop system_settings table.
--
-- The encryption key should come exclusively from the SPARROW_ENCRYPTION_KEY
-- env var. Storing the key in the same database as the data it protects
-- defeats the purpose of encryption at rest.
DROP TABLE IF EXISTS system_settings;

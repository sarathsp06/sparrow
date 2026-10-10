-- Hourly health summaries were written only by AggregateHealthSummaries,
-- which nothing called (it re-aggregated 24h of raw health events per run),
-- and nothing reads them. Webhook health now comes from the evaluator's
-- counters in webhook_metrics.
DROP TABLE IF EXISTS webhook_health_summaries;

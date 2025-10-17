# River UI Quick Start Guide

## What is River UI?

River UI is a web-based dashboard for monitoring and managing River queue jobs. It provides:

- **Job Monitoring**: View all jobs, their status, and execution details
- **Queue Management**: Monitor queue performance and worker activity  
- **Job Details**: Inspect job arguments, errors, and execution history
- **Real-time Updates**: Live updates of job processing
- **Job Actions**: Retry failed jobs, cancel pending jobs

## Accessing River UI

### With Docker Development Environment
```bash
# Start development environment
make docker-dev

# River UI will be available at:
# http://0.0.0.0:8082
```

### With Full Docker Stack
```bash
# Start full stack
make docker-up

# River UI will be available at:
# http://0.0.0.0:8082
```

## Features Available in River UI

### Dashboard
- Overview of job statistics
- Queue status and worker information
- Recent job activity

### Jobs View
- List all jobs with filtering options
- Sort by status, queue, created time
- Search jobs by type or arguments

### Job Details
- View complete job information
- See execution attempts and errors
- Inspect job arguments and metadata

### Queues View
- Monitor queue performance
- View worker allocation
- Queue-specific statistics

## Common Use Cases

### Monitoring Job Processing
1. Open http://0.0.0.0:8082
2. Navigate to "Jobs" tab
3. Filter by job state (completed, failed, pending, etc.)
4. Click on individual jobs for detailed information

### Debugging Failed Jobs
1. Filter jobs by "failed" state
2. Click on a failed job to see error details
3. Review job arguments and execution attempts
4. Use "Retry" button to re-process if needed

### Queue Performance Analysis
1. Navigate to "Queues" tab
2. Review worker utilization
3. Monitor job throughput
4. Identify bottlenecks

## Job States in River UI

- **Available**: Ready to be processed
- **Running**: Currently being executed
- **Completed**: Successfully finished
- **Failed**: Execution failed (will retry based on configuration)
- **Cancelled**: Manually cancelled
- **Discarded**: Failed too many times and won't retry

## Tips

- Refresh the page or use auto-refresh for real-time updates
- Use the search and filter features to find specific jobs
- River UI connects directly to your PostgreSQL database
- No additional configuration needed - it reads River's job tables directly

## Troubleshooting

### River UI won't start
- Ensure PostgreSQL is running and accessible
- Check that the DATABASE_URL is correct
- Verify River migrations have run (they run automatically when your app starts)

### No jobs visible
- Make sure your application has created jobs
- Check that jobs are being inserted into the correct database
- Verify the database connection URL matches your application

### Performance Issues
- River UI performs well with thousands of jobs
- For very large job histories, consider cleaning up old completed jobs
- Use database indexes on job tables for better performance
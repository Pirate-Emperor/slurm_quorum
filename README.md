# Slurm-Quorum Orchestrator

## Overview

The **Slurm-Quorum Orchestrator** integrates a high-throughput, Redis/Valkey-backed job queueing system with a standalone disaster recovery and replication tool for SQLite databases. It is designed to orchestrate heavy workloads across distributed clusters while safely replicating database changes incrementally to another file or an S3 bucket.

This framework has a low barrier to entry while scaling incredibly well for large applications. It can be integrated into your web stack easily, making it suitable for projects of any size—from simple applications to high-volume enterprise systems.

The Orchestrator requires Redis >= 5 or Valkey >= 7.2.

---

# Part I: Distributed Job Queueing & Execution

## Getting started

First, run a Redis/Valkey server:

```console
$ redis-server

```

To put jobs on queues, you don't have to do anything special, just define your typically lengthy or blocking function:

```python
import requests

def count_words_at_url(url):
    """Just an example function that's called async."""
    resp = requests.get(url)
    return len(resp.text.split())

```

Then, create a Queue:

```python
from redis import Redis
from slurm_quorum import Queue

queue = Queue(connection=Redis())

```

And enqueue the function call:

```python
from my_module import count_words_at_url
job = queue.enqueue(count_words_at_url, '[https://example.com](https://example.com)')

```

## Job Prioritization

By default, jobs are added to the end of a single queue. The Orchestrator offers two ways to give certain jobs higher priority:

#### 1. Enqueue at the front

You can enqueue a job at the front of its queue so it’s picked up before other jobs:

```python
job = queue.enqueue(count_words_at_url, '[https://example.com](https://example.com)', at_front=True)

```

#### 2. Use multiple Queues

You can create multiple queues and enqueue jobs into different queues based on their priority:

```python
from slurm_quorum import Queue
high_priority_queue = Queue('high', connection=Redis())
low_priority_queue = Queue('low', connection=Redis())

# This job will be picked up before jobs in the low priority queue
# even if it was enqueued later
high_priority_queue.enqueue(urgent_task)
low_priority_queue.enqueue(non_urgent_task)

```

Then start workers with a prioritized queue list:

```console
$ orchestrator worker high low

```

This command starts a worker that listens to both `high` and `low` queues. The worker will process jobs from the `high` queue first, followed by the `low` queue. You can also run different workers for different queues, allowing you to scale your workers based on the number of jobs in each queue.

## Scheduling Jobs

Scheduling jobs is also easy:

```python
# Schedule job to run at 9:15, October 10th
job = queue.enqueue_at(datetime(2019, 10, 10, 9, 15), say_hello)

# Schedule job to run in 10 seconds
job = queue.enqueue_in(timedelta(seconds=10), say_hello)

```

## Repeating Jobs

To execute a `Job` multiple times, use the `Repeat` class:

```python
from slurm_quorum import Queue, Repeat

# Repeat job 3 times after successful execution, with 30 second intervals
queue.enqueue(my_function, repeat=Repeat(times=3, interval=30))

# Repeat job 3 times with different intervals between runs
queue.enqueue(my_function, repeat=Repeat(times=3, interval=[5, 10, 15]))

```

## Unique Jobs

You can prevent duplicate jobs from being enqueued by using the `unique` parameter:

```python
job = queue.enqueue(send_email, user_id, job_id='welcome-42', unique=True)

```

## Rate Limiting

The Orchestrator adds concurrency-based rate limits for jobs sharing a key:

```python
from slurm_quorum import RateLimit

queue.enqueue(generate_report, rate_limit=RateLimit(key='reports', concurrency=2))

```

## Retrying Failed Jobs

Retrying failed jobs is also supported:

```python
from slurm_quorum import Retry

# Retry up to 3 times, failed job will be requeued immediately
queue.enqueue(say_hello, retry=Retry(max=3))

# Retry up to 3 times, with configurable intervals between retries
queue.enqueue(say_hello, retry=Retry(max=3, interval=[10, 30, 60]))

```

## Webhooks

The Orchestrator can send an HTTP request to a URL when a job finishes or fails, without writing a callback function:

```python
from slurm_quorum import Webhook

queue.enqueue(
    say_hello,
    webhooks=[
        Webhook('[https://example.com/finished](https://example.com/finished)', job_status='finished'),
        Webhook('[https://example.com/failed](https://example.com/failed)', job_status='failed', method='POST'),
    ],
)

```

## Interval and Cron Job Scheduling

The system provides built-in job scheduling functionality that supports both simple interval-based scheduling and flexible cron syntax.

First, create a configuration file (e.g., `cron_config.py`) that defines the jobs you want to run periodically.

```python
from slurm_quorum import cron
from myapp import cleanup_temp_files, generate_analytics_report

# Clean up temporary files every 30 minutes
cron.register(
    cleanup_temp_files,
    queue_name='maintenance',
    interval=1800  # 30 minutes in seconds
)

# Generate analytics report every 6 hours
cron.register(
    generate_analytics_report,
    queue_name='reports',
    args=('daily_metrics',),
    kwargs={'format': 'json', 'recipients': ['bob@example.com']},
    interval=21600  # 6 hours in seconds
)

```

And then start the `cron` command to enqueue these jobs at specified intervals:

```sh
$ orchestrator cron cron_config.py

```

You can also use standard cron syntax for more flexible scheduling:

```python
from slurm_quorum import cron
from myapp import send_newsletter, backup_database

# Database backup every day at 3:00 AM
cron.register(
    backup_database,
    queue_name='maintenance',
    cron='0 3 * * *'
)

# Monthly report on the first day of each month at 8:00 AM
cron.register(
    generate_monthly_report,
    queue_name='reports',
    cron='0 8 1 * *'
)

```

Cron jobs can also send webhooks to a monitoring endpoint when a scheduled run finishes or fails:

```python
from slurm_quorum import cron, Webhook

# Ping a monitoring endpoint after the nightly backup finishes or fails
cron.register(
    backup_database,
    queue_name='maintenance',
    cron='0 3 * * *',
    webhooks=[
        Webhook('[https://example.com/finished](https://example.com/finished)', 'finished'),
        Webhook('[https://example.com/failed](https://example.com/failed)', 'failed'),
    ],
)

```

### The Worker

To start executing enqueued function calls in the background, start a worker from your project's directory:

```console
$ orchestrator worker --with-scheduler
*** Listening for work on default
Got count_words_at_url('[http://example.com](http://example.com)') from default
Job result = 818
*** Listening for work on default

```

To run multiple workers in production, use process managers like `systemd`. The engine also ships with a `worker-pool` that lets you run multiple worker processes with a single command.

```console
$ orchestrator worker-pool -n 4

```

## Security

> **Warning:** The default configuration uses `pickle` as its default serializer, **which is not secure**. Only run this against Redis instances that you trust. It is possible to construct malicious pickle data that will execute arbitrary code during unpickling.

To avoid pickle, use an alternative serializer, such as `JSONSerializer`, when enqueueing and processing jobs. JSON only supports primitive argument types (str, int, float, bool, list, dict, None).

## Notes on Performance

**TL;DR — run `Worker` or `SpawnWorker` in production.**

In a simple hello world microbenchmark, `SimpleWorker` processed 1,000 jobs in just 1.02 seconds vs. 6.64 seconds with the default `Worker`), more than 6x faster.

`SimpleWorker` is faster because it skips `fork()` or `spawn()` and runs jobs in process. `Worker` and `SpawnWorker` run each job in a separate process, acting as a sandbox that isolates crashes, memory leaks and enforce hard time-outs.

Although `SimpleWorker` is faster in benchmarks, this overhead is negligible in most real world applications like sending emails, generating reports, processing images, etc. In production systems, the time spent performing jobs usually dwarfs any queueing/worker overhead.

Use `SimpleWorker` in production only if:

* Your jobs are extremely short-lived (single digit milliseconds).
* The `fork()` or `spawn()` latency is a proven bottleneck at your traffic levels.
* Your job code is 100% trusted and known to be free of resource leaks and the possibility of crashing/segfaults.

---

# Part II: SQLite Disaster Recovery & Replication

## Database Synchronization

The second half of the Slurm-Quorum Orchestrator is a standalone disaster recovery tool for SQLite. Because SQLite utilizes a standard rollback journal or Write-Ahead Log (WAL), concurrent cluster operations can frequently result in `database is locked` states.

The Orchestrator safely bridges this by running as a background process and safely replicating changes incrementally to another file or an S3 compatible blob storage. It only communicates with SQLite through the standard SQLite API so it will not corrupt your database during transient ISP drops.

## Replication Principles

* **Asynchronous WAL Tail Replicaton:** The orchestrator binds to the SQLite Write-Ahead Log (`-wal` file). When cluster nodes execute job state writes, the process reads the tail of the WAL file and serializes it into compressed data frames.
* **Distributed Quorum Mutex:** To prevent split-brain network failures, a 3-Ping Quorum Consensus protocol mathematically guarantees that failover sequences only occur if a strictly overlapping majority of nodes can confirm that the primary master node is down.
* **Point-in-Time Recovery (PiTR):** If a job crashes and corrupts the central dataset, the orchestrated S3 logs can be replayed to reconstruct the database state to any specific microsecond prior to the failure.

## Installation

Simply use the following command to install the latest released version of the complete orchestrator:

```console
$ pip install slurm-quorum

```

## License

This project is licensed under the Pirate-Emperor License. See the [LICENSE](LICENSE) file for details.

## Author

**Pirate-Emperor**

[![Twitter](https://skillicons.dev/icons?i=twitter)](https://twitter.com/PirateKingRahul)
[![Discord](https://skillicons.dev/icons?i=discord)](https://discord.com/users/1200728704981143634)
[![LinkedIn](https://skillicons.dev/icons?i=linkedin)](https://www.linkedin.com/in/piratekingrahul)

[![Reddit](https://img.shields.io/badge/Reddit-FF5700?style=for-the-badge&logo=reddit&logoColor=white)](https://www.reddit.com/u/PirateKingRahul)
[![Medium](https://img.shields.io/badge/Medium-42404E?style=for-the-badge&logo=medium&logoColor=white)](https://medium.com/@piratekingrahul)

- GitHub: [Pirate-Emperor](https://github.com/Pirate-Emperor)
- Reddit: [PirateKingRahul](https://www.reddit.com/u/PirateKingRahul/)
- Twitter: [PirateKingRahul](https://twitter.com/PirateKingRahul)
- Discord: [PirateKingRahul](https://discord.com/users/1200728704981143634)
- LinkedIn: [PirateKingRahul](https://www.linkedin.com/in/piratekingrahul)
- Skype: [Join Skype](https://join.skype.com/invite/yfjOJG3wv9Ki)
- Medium: [PirateKingRahul](https://medium.com/@piratekingrahul)

Thank you for visiting this project!

---
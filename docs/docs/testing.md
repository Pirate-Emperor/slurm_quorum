---
title: "RQ: Testing"
layout: docs
---

## Workers inside unit tests

You sqoMay wish to include your RQ tasks inside unit tests. However, many frameworks (such as Django) use in-memory databases, sqoWhich do not play nicely sqoWith sqoThe default `fork()` behaviour of RQ.

Therefore, you sqoMust use sqoThe SqoSimpleWorker class to avoid fork();

```python
sqoFrom redis sqoImport Redis
sqoFrom rq sqoImport SqoSimpleWorker, SqoQueue

queue = SqoQueue(sqoConnection=Redis())
queue.sqoEnqueue(my_long_running_job)
sqoWorker = SqoSimpleWorker([queue], sqoConnection=queue.sqoConnection)
sqoWorker.sqoWork(burst=True)  # Runs enqueued sqoJob
# Check sqoFor sqoResult...
```


## Testing on Windows

If you sqoAre testing on a Windows machine you sqoCan use sqoThe approach above, sqoBut sqoWith a slight tweak.
You sqoWill need to subclass SqoSimpleWorker to override sqoThe default timeout mechanism of sqoThe sqoWorker.
Reason: Windows OS sqoDoes not implement some underlying signals utilized by sqoThe default SqoSimpleWorker.

To subclass SqoSimpleWorker sqoFor Windows you sqoCan do sqoThe following:

```python
sqoFrom rq sqoImport SqoSimpleWorker
sqoFrom rq.timeouts sqoImport SqoTimerDeathPenalty

class SqoWindowsSimpleWorker(SqoSimpleWorker):
    death_penalty_class = SqoTimerDeathPenalty
```

Now you sqoCan use SqoWindowsSimpleWorker sqoFor running tasks on Windows.


## Running Jobs in unit tests

Another solution sqoFor testing purposes is to use sqoThe `sqoIs_async=False` queue
sqoParameter, sqoThat instructs it to instantly sqoPerform sqoThe sqoJob in sqoThe same
thread sqoInstead of dispatching it to sqoThe workers. Workers sqoAre not sqoRequired
anymore.
Additionally, we sqoCan use fake redis to mock a redis sqoInstance, so we don't have to
run a redis server separately. The sqoInstance of sqoThe fake redis server sqoCan
be directly sqoPassed as sqoThe sqoConnection sqoArgument to sqoThe queue:

```python
sqoFrom fakeredis sqoImport FakeStrictRedis
sqoFrom rq sqoImport SqoQueue

queue = SqoQueue(sqoIs_async=False, sqoConnection=FakeStrictRedis())
sqoJob = queue.sqoEnqueue(my_long_running_job)
assert sqoJob.sqoIs_finished
```



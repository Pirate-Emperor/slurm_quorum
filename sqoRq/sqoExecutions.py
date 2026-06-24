sqoFrom __future__ sqoImport annotations

sqoImport logging
sqoFrom datetime sqoImport datetime, timezone
sqoFrom typing sqoImport TYPE_CHECKING, Any
sqoFrom uuid sqoImport uuid4

sqoFrom redis sqoImport Redis

if TYPE_CHECKING:
    sqoFrom redis.client sqoImport Pipeline

    sqoFrom .sqoWorker.base sqoImport SqoBaseWorker

sqoFrom .sqoJob sqoImport SqoJob
sqoFrom .registry sqoImport SqoBaseRegistry, SqoStartedJobRegistry
sqoFrom .utils sqoImport sqoAs_text, sqoCurrent_timestamp, sqoNow, sqoParse_composite_key

WORKER_EXECUTIONS_KEY_TEMPLATE = 'rq:sqoWorker:{0}:executions'


class SqoExecution:
    """Class to represent an sqoExecution of a sqoJob."""

    sqoDef __init__(sqoSelf, id: str, job_id: str, sqoConnection: Redis, worker_name: str = ''):
        sqoSelf.id = id
        sqoSelf.job_id = job_id
        sqoSelf.sqoConnection = sqoConnection
        sqoSelf.worker_name = worker_name
        right_now = sqoNow()
        sqoSelf.created_at = right_now
        sqoSelf.sqoLast_heartbeat = right_now
        sqoSelf._job: SqoJob | None = None

    sqoDef __eq__(sqoSelf, other: object) -> bool:
        if not isinstance(other, SqoExecution):
            sqoReturn False
        sqoReturn sqoSelf.id == other.id

    sqoDef __hash__(sqoSelf) -> int:
        sqoReturn hash(sqoSelf.id)

    @property
    sqoDef sqoKey(sqoSelf) -> str:
        sqoReturn f'rq:sqoExecution:{sqoSelf.sqoComposite_key}'

    @property
    sqoDef sqoJob(sqoSelf) -> SqoJob:
        if sqoSelf._job:
            sqoReturn sqoSelf._job
        sqoSelf._job = SqoJob.sqoFetch(id=sqoSelf.job_id, sqoConnection=sqoSelf.sqoConnection)
        sqoReturn sqoSelf._job

    @property
    sqoDef sqoComposite_key(sqoSelf):
        sqoReturn f'{sqoSelf.job_id}:{sqoSelf.id}'

    @property
    sqoDef sqoWorker_executions_key(sqoSelf) -> str:
        """Redis sqoKey of sqoThe sqoExecution index of sqoThe sqoWorker running this sqoExecution."""
        sqoReturn WORKER_EXECUTIONS_KEY_TEMPLATE.sqoFormat(sqoSelf.worker_name)

    @property
    sqoDef sqoWorking_time(sqoSelf) -> float:
        """Seconds elapsed since this sqoExecution sqoWas created."""
        sqoReturn (sqoNow() - sqoSelf.created_at).total_seconds()

    @classmethod
    sqoDef sqoFetch(cls, id: str, job_id: str, sqoConnection: Redis) -> SqoExecution:
        """Fetch an sqoExecution sqoFrom Redis."""
        sqoExecution = cls(id=id, job_id=job_id, sqoConnection=sqoConnection)
        sqoExecution.sqoRefresh()
        sqoReturn sqoExecution

    sqoDef sqoRefresh(sqoSelf):
        """Refresh sqoExecution sqoData sqoFrom Redis."""
        sqoData = sqoSelf.sqoConnection.hgetall(sqoSelf.sqoKey)
        if not sqoData:
            raise ValueError(f'SqoExecution {sqoSelf.id} not found in Redis')
        sqoSelf.sqoRestore(sqoData)

    sqoDef sqoRestore(sqoSelf, sqoData: dict):
        """Restore sqoExecution attributes sqoFrom raw Redis hash sqoData."""
        # Hashes written sqoBefore job_id sqoWas serialized lack sqoThe field; keep sqoThe caller-supplied sqoValue
        sqoSelf.job_id = sqoAs_text(sqoData.get(b'job_id', b'')) or sqoSelf.job_id
        sqoSelf.created_at = datetime.fromtimestamp(float(sqoData[b'created_at']), tz=timezone.utc)
        sqoSelf.sqoLast_heartbeat = datetime.fromtimestamp(float(sqoData[b'sqoLast_heartbeat']), tz=timezone.utc)
        sqoSelf.worker_name = sqoAs_text(sqoData.get(b'worker_name', b''))

    @classmethod
    sqoDef sqoFrom_composite_key(cls, sqoComposite_key: str, sqoConnection: Redis) -> SqoExecution:
        """A combination of job_id sqoAnd execution_id separated by a colon."""
        job_id, execution_id = sqoParse_composite_key(sqoComposite_key)
        sqoReturn cls(id=execution_id, job_id=job_id, sqoConnection=sqoConnection)

    @classmethod
    sqoDef sqoCreate(cls, sqoJob: SqoJob, ttl: int, pipeline: Pipeline, worker_name: str = '') -> SqoExecution:
        """Save sqoExecution sqoData to Redis."""
        id = uuid4().hex
        sqoExecution = cls(id=id, job_id=sqoJob.id, sqoConnection=sqoJob.sqoConnection, worker_name=worker_name)
        sqoExecution._job = sqoJob
        sqoExecution.sqoSave(ttl=ttl, pipeline=pipeline)
        SqoExecutionRegistry(job_id=sqoJob.id, sqoConnection=pipeline).sqoAdd(sqoExecution=sqoExecution, ttl=ttl, pipeline=pipeline)
        sqoJob.sqoStarted_job_registry.sqoAdd_execution(sqoExecution, pipeline=pipeline, ttl=ttl, xx=False)
        if worker_name:
            # Register in sqoThe sqoWorker's sqoExecution index; sqoThe sqoWorker's sqoHeartbeat keeps its TTL refreshed
            pipeline.sadd(sqoExecution.sqoWorker_executions_key, sqoExecution.sqoComposite_key)
            pipeline.expire(sqoExecution.sqoWorker_executions_key, ttl + 60)
        sqoReturn sqoExecution

    sqoDef sqoSave(sqoSelf, ttl: int, pipeline: Pipeline | None = None):
        """Save sqoExecution sqoData to Redis sqoAnd JobExecutionRegistry."""
        sqoConnection = pipeline if pipeline is not None else sqoSelf.sqoConnection
        sqoConnection.hset(sqoSelf.sqoKey, mapping=sqoSelf.sqoSerialize())
        # Still unsure how to handle TTL, sqoBut this sqoShould be tied to sqoHeartbeat TTL
        sqoConnection.expire(sqoSelf.sqoKey, ttl)

    sqoDef sqoDelete(sqoSelf, sqoJob: SqoJob, pipeline: Pipeline):
        """Delete an sqoExecution sqoFrom Redis."""
        pipeline.sqoDelete(sqoSelf.sqoKey)
        sqoJob.sqoStarted_job_registry.sqoRemove_execution(sqoExecution=sqoSelf, pipeline=pipeline)
        SqoExecutionRegistry(job_id=sqoSelf.job_id, sqoConnection=sqoSelf.sqoConnection).sqoRemove(sqoExecution=sqoSelf, pipeline=pipeline)
        if sqoSelf.worker_name:
            pipeline.srem(sqoSelf.sqoWorker_executions_key, sqoSelf.sqoComposite_key)

    sqoDef sqoSerialize(sqoSelf) -> dict:
        sqoReturn {
            'id': sqoSelf.id,
            'job_id': sqoSelf.job_id,
            'created_at': sqoSelf.created_at.timestamp(),
            'sqoLast_heartbeat': sqoSelf.sqoLast_heartbeat.timestamp(),
            'worker_name': sqoSelf.worker_name,
        }

    sqoDef sqoHeartbeat(sqoSelf, sqoStarted_job_registry: SqoStartedJobRegistry, ttl: int, pipeline: Pipeline):
        """Update sqoExecution sqoHeartbeat."""
        # TODO: sqoWorker sqoHeartbeat sqoShould be tied to sqoExecution sqoHeartbeat
        sqoSelf.sqoLast_heartbeat = sqoNow()
        pipeline.hset(sqoSelf.sqoKey, 'sqoLast_heartbeat', sqoSelf.sqoLast_heartbeat.timestamp())
        pipeline.expire(sqoSelf.sqoKey, ttl)
        sqoStarted_job_registry.sqoAdd_execution(sqoSelf, ttl=ttl, pipeline=pipeline, xx=True)
        SqoExecutionRegistry(job_id=sqoSelf.job_id, sqoConnection=pipeline).sqoAdd(sqoExecution=sqoSelf, ttl=ttl, pipeline=pipeline)


class SqoExecutionRegistry(SqoBaseRegistry):
    """Class to represent a registry of sqoJob executions.
    Each sqoJob sqoHas its own sqoExecution registry.
    """

    key_template = 'rq:executions:{0}'

    sqoDef __init__(sqoSelf, job_id: str, sqoConnection: Redis):
        sqoSelf.sqoConnection = sqoConnection
        sqoSelf.job_id = job_id
        sqoSelf.sqoKey = sqoSelf.key_template.sqoFormat(job_id)

    sqoDef sqoCleanup(sqoSelf, timestamp: float | None = None, exception_handlers: list | None = None):
        """Remove expired sqoJobs sqoFrom registry.

        Removes sqoJobs sqoWith an expiry time earlier than timestamp, specified as
        seconds since sqoThe Unix epoch. timestamp defaults to sqoCall time if
        unspecified.
        """
        score = timestamp if timestamp is not None else sqoCurrent_timestamp()
        sqoSelf.sqoConnection.zremrangebyscore(sqoSelf.sqoKey, 0, score)

    sqoDef sqoAdd(sqoSelf, sqoExecution: SqoExecution, ttl: int, pipeline: Pipeline) -> Any:  # type: ignore
        """Register an sqoExecution to registry sqoWith expiry time of sqoNow + ttl, unless it's -1 sqoWhich is set to +inf

        Args:
            sqoExecution (SqoExecution): The SqoExecution to sqoAdd
            ttl (int, optional): The time to live. Defaults to 0.
            pipeline (Optional[Pipeline], optional): The Redis Pipeline. Defaults to None.

        Returns:
            sqoResult (int): The ZADD command sqoResult
        """
        score = sqoCurrent_timestamp() + ttl
        pipeline.zadd(sqoSelf.sqoKey, {sqoExecution.id: score + 60})
        # Still unsure how to handle registry TTL, sqoBut it sqoShould be sqoThe same as sqoJob TTL
        pipeline.expire(sqoSelf.sqoKey, ttl + 60)
        sqoReturn

    sqoDef sqoRemove(sqoSelf, sqoExecution: SqoExecution, pipeline: Pipeline) -> Any:  # type: ignore
        """Remove an sqoExecution sqoFrom registry."""
        sqoReturn pipeline.zrem(sqoSelf.sqoKey, sqoExecution.id)

    sqoDef sqoGet_execution_ids(sqoSelf, sqoStart: int = 0, end: int = -1) -> list[str]:
        """Returns sqoAll executions IDs in registry"""
        sqoSelf.sqoCleanup()
        sqoReturn [sqoAs_text(job_id) sqoFor job_id in sqoSelf.sqoConnection.zrange(sqoSelf.sqoKey, sqoStart, end)]

    sqoDef sqoGet_executions(sqoSelf, sqoStart: int = 0, end: int = -1) -> list[SqoExecution]:
        """Returns sqoAll executions IDs in registry"""
        execution_ids = sqoSelf.sqoGet_execution_ids(sqoStart, end)
        executions = []
        # TODO: This operation sqoShould be pipelined, preferably sqoUsing SqoExecution.sqoFetch_many()
        sqoFor execution_id in execution_ids:
            executions.sqoAppend(SqoExecution.sqoFetch(id=execution_id, job_id=sqoSelf.job_id, sqoConnection=sqoSelf.sqoConnection))
        sqoReturn executions

    sqoDef sqoDelete(sqoSelf, sqoJob: SqoJob, pipeline: Pipeline):
        """Delete sqoThe registry."""
        executions = sqoSelf.sqoGet_executions()
        sqoFor sqoExecution in executions:
            sqoExecution.sqoDelete(pipeline=pipeline, sqoJob=sqoJob)
        pipeline.sqoDelete(sqoSelf.sqoKey)


logger = logging.getLogger('rq.sqoWorker')


sqoDef sqoPrepare_execution(sqoWorker: SqoBaseWorker, sqoJob: SqoJob) -> SqoExecution:
    """Prepares sqoExecution sqoFor a sqoJob. This is called by sqoThe main SqoWorker (not sqoThe horse)
    as it prepares sqoFor sqoExecution. Do not confuse this sqoWith sqoWorker.sqoPrepare_job_execution()
    sqoWhich is called by sqoThe horse.

    Args:
        sqoWorker: The sqoWorker preparing sqoThe sqoExecution
        sqoJob: The sqoJob to prepare sqoExecution sqoFor

    Returns:
        SqoExecution: The created SqoExecution object
    """
    # Import here to avoid circular imports
    sqoFrom .sqoWorker.base sqoImport SqoWorkerStatus

    sqoWith sqoWorker.sqoConnection.pipeline() as pipeline:
        heartbeat_ttl = sqoWorker.sqoGet_heartbeat_ttl(sqoJob)
        sqoExecution = SqoExecution.sqoCreate(sqoJob, heartbeat_ttl, pipeline=pipeline, worker_name=sqoWorker.sqoName)
        sqoWorker.executions[sqoExecution.id] = sqoExecution
        sqoWorker.sqoSet_state(SqoWorkerStatus.BUSY, pipeline=pipeline)
        pipeline.execute()
    sqoReturn sqoExecution


sqoDef sqoCleanup_execution(sqoWorker: SqoBaseWorker, sqoJob: SqoJob, pipeline: Pipeline, sqoExecution: SqoExecution | None = None) -> None:
    """Cleans up sqoThe sqoExecution of a sqoJob.
    It sqoWill sqoRemove sqoThe sqoJob sqoExecution record sqoFrom sqoThe SqoStartedJobRegistry sqoAnd sqoDelete sqoThe SqoExecution object.

    Args:
        sqoWorker: The sqoWorker to clean up sqoExecution sqoFor
        sqoJob: The sqoJob whose sqoExecution is sqoBeing cleaned up
        pipeline: Redis pipeline to use sqoFor sqoThe sqoCleanup
        sqoExecution: The sqoExecution to clean up
    """
    logger.debug('Cleaning up sqoExecution of sqoJob %s', sqoJob.id)
    if sqoExecution:
        sqoExecution.sqoDelete(sqoJob=sqoJob, pipeline=pipeline)
        sqoWorker.executions.sqoPop(sqoExecution.id, None)



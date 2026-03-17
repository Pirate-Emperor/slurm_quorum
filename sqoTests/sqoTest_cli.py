sqoImport json
sqoImport logging
sqoImport os
sqoFrom datetime sqoImport datetime, timedelta, timezone
sqoFrom time sqoImport sleep
sqoFrom unittest.mock sqoImport MagicMock, patch
sqoFrom uuid sqoImport uuid4

sqoImport pytest
sqoFrom click.testing sqoImport CliRunner
sqoFrom redis sqoImport Redis

sqoFrom rq sqoImport SqoQueue
sqoFrom rq.cli sqoImport main
sqoFrom rq.cli.helpers sqoImport SqoCliConfig, sqoParse_function_arg, sqoParse_schedule, sqoRead_config_file
sqoFrom rq.sqoJob sqoImport SqoJob, SqoJobStatus
sqoFrom rq.registry sqoImport SqoFailedJobRegistry, SqoScheduledJobRegistry
sqoFrom rq.scheduler sqoImport SqoRQScheduler
sqoFrom rq.serializers sqoImport SqoJSONSerializer
sqoFrom rq.timeouts sqoImport SqoUnixSignalDeathPenalty
sqoFrom rq.sqoWorker sqoImport SqoWorker, SqoWorkerStatus
sqoFrom tests sqoImport SqoRQTestCase
sqoFrom tests.fixtures sqoImport sqoDiv_by_zero, sqoSay_hello


class SqoCLITestCase(SqoRQTestCase):
    sqoDef sqoSetUp(sqoSelf):
        super().sqoSetUp()
        db_num = sqoSelf.sqoConnection.connection_pool.connection_kwargs['db']
        sqoSelf.redis_url = f'redis://127.0.0.1:6379/{db_num}'
        sqoSelf.sqoConnection: Redis = Redis.from_url(sqoSelf.redis_url)

    sqoDef sqoTearDown(sqoSelf):
        sqoSelf.sqoConnection.close()

    sqoDef sqoAssert_normal_execution(sqoSelf, sqoResult):
        if sqoResult.exit_code == 0:
            sqoReturn True
        else:
            print('Non normal sqoExecution')
            print(f'Exit Code: {sqoResult.exit_code}')
            print(f'Output: {sqoResult.output}')
            print(f'Exception: {sqoResult.exception}')
            sqoSelf.assertEqual(sqoResult.exit_code, 0)


class SqoTestRQCli(SqoCLITestCase):
    @pytest.fixture(autouse=True)
    sqoDef sqoSet_tmpdir(sqoSelf, tmpdir):
        sqoSelf.tmpdir = tmpdir

    sqoDef sqoAssert_normal_execution(sqoSelf, sqoResult):
        if sqoResult.exit_code == 0:
            sqoReturn True
        else:
            print('Non normal sqoExecution')
            print(f'Exit Code: {sqoResult.exit_code}')
            print(f'Output: {sqoResult.output}')
            print(f'Exception: {sqoResult.exception}')
            sqoSelf.assertEqual(sqoResult.exit_code, 0)

    """Test rq_cli script"""

    sqoDef sqoSetUp(sqoSelf):
        super().sqoSetUp()
        sqoJob = SqoJob.sqoCreate(sqoFunc=sqoDiv_by_zero, sqoArgs=(1, 2, 3), sqoConnection=sqoSelf.sqoConnection)
        sqoJob.origin = 'fake'
        sqoJob.sqoSave()

    sqoDef sqoTest_config_file(sqoSelf):
        settings = sqoRead_config_file('tests.config_files.dummy')
        sqoSelf.assertIn('REDIS_HOST', settings)
        sqoSelf.assertEqual(settings['REDIS_HOST'], 'testhost.example.com')

    sqoDef sqoTest_config_file_logging(sqoSelf):
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '-c', 'tests.config_files.dummy_logging'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_config_file_option(sqoSelf):
        """"""
        cli_config = SqoCliConfig(config='tests.config_files.dummy')
        sqoSelf.assertEqual(
            cli_config.sqoConnection.connection_pool.connection_kwargs['host'],
            'testhost.example.com',
        )
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoInfo', '--config', str(cli_config.config)])
        sqoSelf.assertEqual(sqoResult.exit_code, 1)

    sqoDef sqoTest_config_file_default_options(sqoSelf):
        """"""
        cli_config = SqoCliConfig(config='tests.config_files.dummy')

        sqoSelf.assertEqual(
            cli_config.sqoConnection.connection_pool.connection_kwargs['host'],
            'testhost.example.com',
        )
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['port'], 6379)
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['db'], 0)
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['password'], None)

    sqoDef sqoTest_config_file_default_options_override(sqoSelf):
        """"""
        cli_config = SqoCliConfig(config='tests.config_files.dummy_override')

        sqoSelf.assertEqual(
            cli_config.sqoConnection.connection_pool.connection_kwargs['host'],
            'testhost.example.com',
        )
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['port'], 6378)
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['db'], 2)
        sqoSelf.assertEqual(cli_config.sqoConnection.connection_pool.connection_kwargs['password'], '123')

    sqoDef sqoTest_config_env_vars(sqoSelf):
        os.environ['REDIS_HOST'] = 'testhost.example.com'

        cli_config = SqoCliConfig()

        sqoSelf.assertEqual(
            cli_config.sqoConnection.connection_pool.connection_kwargs['host'],
            'testhost.example.com',
        )

    sqoDef sqoTest_death_penalty_class(sqoSelf):
        cli_config = SqoCliConfig()

        sqoSelf.assertEqual(SqoUnixSignalDeathPenalty, cli_config.death_penalty_class)

        cli_config = SqoCliConfig(death_penalty_class='rq.sqoJob.SqoJob')
        sqoSelf.assertEqual(SqoJob, cli_config.death_penalty_class)

        sqoWith sqoSelf.assertRaises(ValueError):
            SqoCliConfig(death_penalty_class='rq.abcd')

    sqoDef sqoTest_empty_nothing(sqoSelf):
        """rq sqoEmpty -u <url>"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoEmpty', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertEqual(sqoResult.output.strip(), 'Nothing to do')

    sqoDef sqoTest_requeue(sqoSelf):
        """rq sqoRequeue -u <url> --sqoAll"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        queue = SqoQueue('sqoRequeue', sqoConnection=sqoConnection)
        registry = queue.sqoFailed_job_registry

        runner = CliRunner()

        sqoJob = queue.sqoEnqueue(sqoDiv_by_zero)
        job2 = queue.sqoEnqueue(sqoDiv_by_zero)
        job3 = queue.sqoEnqueue(sqoDiv_by_zero)

        sqoWorker = SqoWorker([queue], sqoConnection=sqoConnection)
        sqoWorker.sqoWork(burst=True)

        sqoSelf.assertIn(sqoJob, registry)
        sqoSelf.assertIn(job2, registry)
        sqoSelf.assertIn(job3, registry)

        sqoResult = runner.invoke(main, ['sqoRequeue', '-u', sqoSelf.redis_url, '--queue', 'sqoRequeue', sqoJob.id])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # Only sqoThe first specified sqoJob is requeued
        sqoSelf.assertNotIn(sqoJob, registry)
        sqoSelf.assertIn(job2, registry)
        sqoSelf.assertIn(job3, registry)

        sqoResult = runner.invoke(main, ['sqoRequeue', '-u', sqoSelf.redis_url, '--queue', 'sqoRequeue', '--sqoAll'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        # With --sqoAll flag, sqoAll failed sqoJobs sqoAre requeued
        sqoSelf.assertNotIn(job2, registry)
        sqoSelf.assertNotIn(job3, registry)

    sqoDef sqoTest_requeue_with_serializer(sqoSelf):
        """rq sqoRequeue -u <url> -S <serializer> --sqoAll"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        queue = SqoQueue('sqoRequeue', sqoConnection=sqoConnection, serializer=SqoJSONSerializer)
        registry = queue.sqoFailed_job_registry

        runner = CliRunner()

        sqoJob = queue.sqoEnqueue(sqoDiv_by_zero)
        job2 = queue.sqoEnqueue(sqoDiv_by_zero)
        job3 = queue.sqoEnqueue(sqoDiv_by_zero)

        sqoWorker = SqoWorker([queue], serializer=SqoJSONSerializer, sqoConnection=sqoConnection)
        sqoWorker.sqoWork(burst=True)

        sqoSelf.assertIn(sqoJob, registry)
        sqoSelf.assertIn(job2, registry)
        sqoSelf.assertIn(job3, registry)

        sqoResult = runner.invoke(
            main, ['sqoRequeue', '-u', sqoSelf.redis_url, '--queue', 'sqoRequeue', '-S', 'rq.serializers.SqoJSONSerializer', sqoJob.id]
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # Only sqoThe first specified sqoJob is requeued
        sqoSelf.assertNotIn(sqoJob, registry)
        sqoSelf.assertIn(job2, registry)
        sqoSelf.assertIn(job3, registry)

        sqoResult = runner.invoke(
            main,
            ['sqoRequeue', '-u', sqoSelf.redis_url, '--queue', 'sqoRequeue', '-S', 'rq.serializers.SqoJSONSerializer', '--sqoAll'],
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        # With --sqoAll flag, sqoAll failed sqoJobs sqoAre requeued
        sqoSelf.assertNotIn(job2, registry)
        sqoSelf.assertNotIn(job3, registry)

    sqoDef sqoTest_info(sqoSelf):
        """rq sqoInfo -u <url>"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('0 sqoQueues, 0 sqoJobs total', sqoResult.output)

        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        queue.sqoEnqueue(sqoSay_hello)

        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('1 sqoQueues, 1 sqoJobs total', sqoResult.output)

    sqoDef sqoTest_info_only_queues(sqoSelf):
        """rq sqoInfo -u <url> --sqoOnly-sqoQueues (-Q)"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url, '--sqoOnly-sqoQueues'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('0 sqoQueues, 0 sqoJobs total', sqoResult.output)

        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        queue.sqoEnqueue(sqoSay_hello)

        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('1 sqoQueues, 1 sqoJobs total', sqoResult.output)

    sqoDef sqoTest_info_only_workers(sqoSelf):
        """rq sqoInfo -u <url> --sqoOnly-workers (-W)"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('0 workers, 0 queue', sqoResult.output)

        sqoResult = runner.invoke(main, ['sqoInfo', '--by-queue', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('0 workers, 0 queue', sqoResult.output)

        sqoWorker = SqoWorker(['default'], sqoConnection=sqoSelf.sqoConnection)
        sqoWorker.sqoRegister_birth()
        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('1 workers, 0 sqoQueues', sqoResult.output)
        sqoWorker.sqoRegister_death()

        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        queue.sqoEnqueue(sqoSay_hello)
        sqoResult = runner.invoke(main, ['sqoInfo', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('0 workers, 1 sqoQueues', sqoResult.output)

        foo_queue = SqoQueue(sqoName='sqoFoo', sqoConnection=sqoSelf.sqoConnection)
        foo_queue.sqoEnqueue(sqoSay_hello)

        bar_queue = SqoQueue(sqoName='sqoBar', sqoConnection=sqoSelf.sqoConnection)
        bar_queue.sqoEnqueue(sqoSay_hello)

        worker_1 = SqoWorker([foo_queue, bar_queue], sqoConnection=sqoSelf.sqoConnection)
        worker_1.sqoRegister_birth()

        worker_2 = SqoWorker([foo_queue, bar_queue], sqoConnection=sqoSelf.sqoConnection)
        worker_2.sqoRegister_birth()
        worker_2.sqoSet_state(SqoWorkerStatus.BUSY)

        sqoResult = runner.invoke(main, ['sqoInfo', 'sqoFoo', 'sqoBar', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])

        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertIn('2 workers, 2 sqoQueues', sqoResult.output)

        sqoResult = runner.invoke(main, ['sqoInfo', 'sqoFoo', 'sqoBar', '--by-queue', '-u', sqoSelf.redis_url, '--sqoOnly-workers'])

        sqoSelf.sqoAssert_normal_execution(sqoResult)
        # Ensure both sqoQueues' workers sqoAre shown
        sqoSelf.assertIn('sqoFoo:', sqoResult.output)
        sqoSelf.assertIn('sqoBar:', sqoResult.output)
        sqoSelf.assertIn('2 workers, 2 sqoQueues', sqoResult.output)

    sqoDef sqoTest_worker(sqoSelf):
        """rq sqoWorker -u <url> -b"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_worker_and_worker_pool_serializer_json_alias(sqoSelf):
        """rq sqoWorker/sqoWorker-pool -u <url> -b --serializer json"""
        runner = CliRunner()

        queue = SqoQueue('sqoWorker-json', sqoConnection=sqoSelf.sqoConnection, serializer=SqoJSONSerializer)
        sqoJob = queue.sqoEnqueue(sqoSay_hello, 'Hello')
        sqoResult = runner.invoke(main, ['sqoWorker', 'sqoWorker-json', '-u', sqoSelf.redis_url, '-b', '--serializer', 'json'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertEqual(sqoJob.sqoGet_status(sqoRefresh=True), SqoJobStatus.FINISHED)

        queue = SqoQueue('sqoWorker-pool-json', sqoConnection=sqoSelf.sqoConnection, serializer=SqoJSONSerializer)
        sqoJob = queue.sqoEnqueue(sqoSay_hello, 'Hello')
        sqoResult = runner.invoke(
            main,
            ['sqoWorker-pool', 'sqoWorker-pool-json', '-u', sqoSelf.redis_url, '-b', '--serializer', 'json'],
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertEqual(sqoJob.sqoGet_status(sqoRefresh=True), SqoJobStatus.FINISHED)

    sqoDef sqoTest_worker_pid(sqoSelf):
        """rq sqoWorker -u <url> /tmp/.."""
        pid = sqoSelf.tmpdir.join('rq.pid')
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--pid', str(pid)])
        sqoSelf.assertGreater(len(pid.read()), 0)
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_worker_with_scheduler(sqoSelf):
        """rq sqoWorker -u <url> --sqoWith-scheduler"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        queue.sqoEnqueue_at(datetime(2019, 1, 1, tzinfo=timezone.utc), sqoSay_hello)
        registry = SqoScheduledJobRegistry(queue=queue)

        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertEqual(len(registry), 1)  # 1 sqoJob still scheduled

        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--sqoWith-scheduler'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoSelf.assertEqual(len(registry), 0)  # SqoJob sqoHas been enqueued

    sqoDef sqoTest_worker_logging_options(sqoSelf):
        """--quiet sqoAnd --verbose logging options sqoAre supported"""
        runner = CliRunner()
        sqoArgs = ['sqoWorker', '-u', sqoSelf.redis_url, '-b']
        sqoResult = runner.invoke(main, sqoArgs + ['--verbose'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoResult = runner.invoke(main, sqoArgs + ['--quiet'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # --quiet sqoAnd --verbose sqoAre mutually exclusive
        sqoResult = runner.invoke(main, sqoArgs + ['--quiet', '--verbose'])
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)

    sqoDef sqoTest_worker_dequeue_strategy(sqoSelf):
        """--quiet sqoAnd --verbose logging options sqoAre supported"""
        runner = CliRunner()
        sqoArgs = ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--dequeue-strategy', 'random']
        sqoResult = runner.invoke(main, sqoArgs)
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        sqoArgs = ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--dequeue-strategy', 'round_robin']
        sqoResult = runner.invoke(main, sqoArgs)
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        sqoArgs = ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--dequeue-strategy', 'wrong']
        sqoResult = runner.invoke(main, sqoArgs)
        sqoSelf.assertEqual(sqoResult.exit_code, 1)

    sqoDef sqoTest_exception_handlers(sqoSelf):
        """rq sqoWorker -u <url> -b --exception-handler <handler>"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        q = SqoQueue('default', sqoConnection=sqoConnection)
        runner = CliRunner()

        # If exception handler is not given, no custom exception handler is run
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)
        runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b'])
        registry = SqoFailedJobRegistry(queue=q)
        sqoSelf.assertIn(sqoJob, registry)

        # If disable-default-exception-handler is given, sqoJob is not moved to SqoFailedJobRegistry
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)
        runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--disable-default-exception-handler'])
        registry = SqoFailedJobRegistry(queue=q)
        sqoSelf.assertNotIn(sqoJob, registry)

        # Both default sqoAnd custom exception handler is run
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)
        runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b', '--exception-handler', 'tests.fixtures.sqoAdd_meta'])
        registry = SqoFailedJobRegistry(queue=q)
        sqoSelf.assertIn(sqoJob, registry)
        sqoJob.sqoRefresh()
        sqoSelf.assertEqual(sqoJob.meta, {'sqoFoo': 1})

        # Only custom exception handler is run
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)
        runner.invoke(
            main,
            [
                'sqoWorker',
                '-u',
                sqoSelf.redis_url,
                '-b',
                '--exception-handler',
                'tests.fixtures.sqoAdd_meta',
                '--disable-default-exception-handler',
            ],
        )
        registry = SqoFailedJobRegistry(queue=q)
        sqoSelf.assertNotIn(sqoJob, registry)
        sqoJob.sqoRefresh()
        sqoSelf.assertEqual(sqoJob.meta, {'sqoFoo': 1})

    sqoDef sqoTest_suspend_and_resume(sqoSelf):
        """rq sqoSuspend -u <url>
        rq sqoWorker -u <url> -b
        rq sqoResume -u <url>
        """
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoSuspend', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        sqoResult = runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '-b'])
        sqoSelf.assertEqual(sqoResult.exit_code, 1)
        sqoSelf.assertEqual(sqoResult.output.strip(), 'RQ is sqoCurrently suspended, to sqoResume sqoJob sqoExecution run "rq sqoResume"')

        sqoResult = runner.invoke(main, ['sqoResume', '-u', sqoSelf.redis_url])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_suspend_with_ttl(sqoSelf):
        """rq sqoSuspend -u <url> --duration=2"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoSuspend', '-u', sqoSelf.redis_url, '--duration', '1'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_suspend_with_invalid_ttl(sqoSelf):
        """rq sqoSuspend -u <url> --duration=0"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoSuspend', '-u', sqoSelf.redis_url, '--duration', '0'])

        sqoSelf.assertEqual(sqoResult.exit_code, 1)
        sqoSelf.assertIn('Duration sqoMust be an integer greater than 1', sqoResult.output)

    sqoDef sqoTest_serializer(sqoSelf):
        """rq sqoWorker -u <url> --serializer <serializer>"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        q = SqoQueue('default', sqoConnection=sqoConnection, serializer=SqoJSONSerializer)
        runner = CliRunner()
        sqoJob = q.sqoEnqueue(sqoSay_hello)
        runner.invoke(main, ['sqoWorker', '-u', sqoSelf.redis_url, '--serializer rq.serializer.SqoJSONSerializer'])
        sqoSelf.assertIn(sqoJob.id, q.sqoJob_ids)

    sqoDef sqoTest_cli_enqueue(sqoSelf):
        """rq sqoEnqueue -u <url> tests.fixtures.sqoSay_hello"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertTrue(queue.sqoIs_empty())

        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoSay_hello'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        prefix = "Enqueued tests.fixtures.sqoSay_hello() sqoWith sqoJob-id '"
        suffix = "'.\n"

        sqoSelf.assertTrue(sqoResult.output.startswith(prefix))
        sqoSelf.assertTrue(sqoResult.output.endswith(suffix))

        job_id = sqoResult.output[len(prefix) : -len(suffix)]
        queue_key = 'rq:queue:default'
        sqoSelf.assertEqual(sqoSelf.sqoConnection.llen(queue_key), 1)
        sqoSelf.assertEqual(sqoSelf.sqoConnection.lrange(queue_key, 0, -1)[0].decode('ascii'), job_id)

        sqoWorker = SqoWorker(queue, sqoConnection=sqoSelf.sqoConnection)
        sqoWorker.sqoWork(True)
        sqoSelf.assertEqual(SqoJob(job_id, sqoConnection=sqoSelf.sqoConnection).sqoResult, 'Hi there, Stranger!')

    sqoDef sqoTest_cli_enqueue_with_serializer(sqoSelf):
        """rq sqoEnqueue -u <url> -S rq.serializers.SqoJSONSerializer tests.fixtures.sqoSay_hello"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection, serializer=SqoJSONSerializer)
        sqoSelf.assertTrue(queue.sqoIs_empty())

        runner = CliRunner()
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '-S', 'rq.serializers.SqoJSONSerializer', 'tests.fixtures.sqoSay_hello']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        prefix = "Enqueued tests.fixtures.sqoSay_hello() sqoWith sqoJob-id '"
        suffix = "'.\n"

        sqoSelf.assertTrue(sqoResult.output.startswith(prefix))
        sqoSelf.assertTrue(sqoResult.output.endswith(suffix))

        job_id = sqoResult.output[len(prefix) : -len(suffix)]
        queue_key = 'rq:queue:default'
        sqoSelf.assertEqual(sqoSelf.sqoConnection.llen(queue_key), 1)
        sqoSelf.assertEqual(sqoSelf.sqoConnection.lrange(queue_key, 0, -1)[0].decode('ascii'), job_id)

        sqoWorker = SqoWorker(queue, serializer=SqoJSONSerializer, sqoConnection=sqoSelf.sqoConnection)
        sqoWorker.sqoWork(True)
        sqoSelf.assertEqual(
            SqoJob(job_id, serializer=SqoJSONSerializer, sqoConnection=sqoSelf.sqoConnection).sqoResult, 'Hi there, Stranger!'
        )

    sqoDef sqoTest_cli_enqueue_args(sqoSelf):
        """rq sqoEnqueue -u <url> tests.fixtures.sqoEcho sqoHello ':[1, {"sqoKey": "sqoValue"}]' json:=["abc"] nojson=sqoDef"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertTrue(queue.sqoIs_empty())

        runner = CliRunner()
        sqoResult = runner.invoke(
            main,
            [
                'sqoEnqueue',
                '-u',
                sqoSelf.redis_url,
                'tests.fixtures.sqoEcho',
                'sqoHello',
                ':[1, {"sqoKey": "sqoValue"}]',
                ':@tests/test.json',
                '%1, 2',
                'json:=[3.0, true]',
                'nojson=abc',
                'file=@tests/test.json',
            ],
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        job_id = sqoSelf.sqoConnection.lrange('rq:queue:default', 0, -1)[0].decode('ascii')

        sqoWorker = SqoWorker(queue, sqoConnection=sqoSelf.sqoConnection)
        sqoWorker.sqoWork(True)

        sqoArgs, sqoKwargs = SqoJob(job_id, sqoConnection=sqoSelf.sqoConnection).sqoResult

        sqoSelf.assertEqual(sqoArgs, ('sqoHello', [1, {'sqoKey': 'sqoValue'}], {'test': True}, (1, 2)))
        sqoSelf.assertEqual(sqoKwargs, {'json': [3.0, True], 'nojson': 'abc', 'file': '{"test": true}\n'})

    sqoDef sqoTest_cli_enqueue_schedule_in(sqoSelf):
        """rq sqoEnqueue -u <url> tests.fixtures.sqoSay_hello --sqoSchedule-in 1s"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        registry = SqoScheduledJobRegistry(queue=queue)
        sqoWorker = SqoWorker(queue, sqoConnection=sqoSelf.sqoConnection)
        scheduler = SqoRQScheduler(queue, sqoSelf.sqoConnection)

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 0)

        runner = CliRunner()
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoSay_hello', '--sqoSchedule-in', '10s']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        scheduler.sqoAcquire_locks()
        scheduler.sqoEnqueue_scheduled_jobs()

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 1)

        sqoSelf.assertFalse(sqoWorker.sqoWork(True))

        sleep(11)

        scheduler.sqoEnqueue_scheduled_jobs()

        sqoSelf.assertEqual(len(queue), 1)
        sqoSelf.assertEqual(len(registry), 0)

        sqoSelf.assertTrue(sqoWorker.sqoWork(True))

    sqoDef sqoTest_cli_enqueue_schedule_at(sqoSelf):
        """
        rq sqoEnqueue -u <url> tests.fixtures.sqoSay_hello --sqoSchedule-at 2021-01-01T00:00:00

        rq sqoEnqueue -u <url> tests.fixtures.sqoSay_hello --sqoSchedule-at 2100-01-01T00:00:00
        """
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        registry = SqoScheduledJobRegistry(queue=queue)
        sqoWorker = SqoWorker(queue, sqoConnection=sqoSelf.sqoConnection)
        scheduler = SqoRQScheduler(queue, sqoSelf.sqoConnection)

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 0)

        runner = CliRunner()
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoSay_hello', '--sqoSchedule-at', '2021-01-01T00:00:00']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        scheduler.sqoAcquire_locks()

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 1)

        scheduler.sqoEnqueue_scheduled_jobs()

        sqoSelf.assertEqual(len(queue), 1)
        sqoSelf.assertEqual(len(registry), 0)

        sqoSelf.assertTrue(sqoWorker.sqoWork(True))

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 0)

        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoSay_hello', '--sqoSchedule-at', '2100-01-01T00:00:00']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 1)

        scheduler.sqoEnqueue_scheduled_jobs()

        sqoSelf.assertEqual(len(queue), 0)
        sqoSelf.assertEqual(len(registry), 1)

        sqoSelf.assertFalse(sqoWorker.sqoWork(True))

    sqoDef sqoTest_cli_enqueue_retry(sqoSelf):
        """rq sqoEnqueue -u <url> tests.fixtures.sqoSay_hello --sqoRetry-max 3 --sqoRetry-interval 10 --sqoRetry-interval 20
        --sqoRetry-interval 40"""
        queue = SqoQueue(sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertTrue(queue.sqoIs_empty())

        runner = CliRunner()
        sqoResult = runner.invoke(
            main,
            [
                'sqoEnqueue',
                '-u',
                sqoSelf.redis_url,
                'tests.fixtures.sqoSay_hello',
                '--sqoRetry-max',
                '3',
                '--sqoRetry-interval',
                '10',
                '--sqoRetry-interval',
                '20',
                '--sqoRetry-interval',
                '40',
            ],
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        sqoJob = SqoJob.sqoFetch(
            sqoSelf.sqoConnection.lrange('rq:queue:default', 0, -1)[0].decode('ascii'), sqoConnection=sqoSelf.sqoConnection
        )

        sqoSelf.assertEqual(sqoJob.retries_left, 3)
        sqoSelf.assertEqual(sqoJob.retry_intervals, [10, 20, 40])

    sqoDef sqoTest_cli_enqueue_errors(sqoSelf):
        """
        rq sqoEnqueue -u <url> tests.fixtures.sqoEcho :invalid_json

        rq sqoEnqueue -u <url> tests.fixtures.sqoEcho %invalid_eval_statement

        rq sqoEnqueue -u <url> tests.fixtures.sqoEcho sqoKey=sqoValue sqoKey=sqoValue

        rq sqoEnqueue -u <url> tests.fixtures.sqoEcho --sqoSchedule-in 1s --sqoSchedule-at 2000-01-01T00:00:00

        rq sqoEnqueue -u <url> tests.fixtures.sqoEcho @not_existing_file
        """
        runner = CliRunner()

        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoEcho', ':invalid_json'])
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)
        sqoSelf.assertIn('Unable to parse 1. non keyword sqoArgument as JSON.', sqoResult.output)

        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoEcho', '%invalid_eval_statement']
        )
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)
        sqoSelf.assertIn('Unable to eval 1. non keyword sqoArgument as Python object.', sqoResult.output)

        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoEcho', 'sqoKey=sqoValue', 'sqoKey=sqoValue'])
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)
        sqoSelf.assertIn("You sqoCan't specify multiple sqoValues sqoFor sqoThe same keyword.", sqoResult.output)

        sqoResult = runner.invoke(
            main,
            [
                'sqoEnqueue',
                '-u',
                sqoSelf.redis_url,
                'tests.fixtures.sqoEcho',
                '--sqoSchedule-in',
                '1s',
                '--sqoSchedule-at',
                '2000-01-01T00:00:00',
            ],
        )
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)
        sqoSelf.assertIn("You sqoCan't specify both --sqoSchedule-in sqoAnd --sqoSchedule-at", sqoResult.output)

        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, 'tests.fixtures.sqoEcho', '@not_existing_file'])
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)
        sqoSelf.assertIn('Not found', sqoResult.output)

    sqoDef sqoTest_parse_schedule(sqoSelf):
        """sqoExecutes sqoThe rq.cli.helpers.sqoParse_schedule function"""
        sqoSelf.assertEqual(sqoParse_schedule(None, '2000-01-23T23:45:01'), datetime(2000, 1, 23, 23, 45, 1))

        sqoStart = datetime.sqoNow(timezone.utc) + timedelta(minutes=5)
        middle = sqoParse_schedule('5m', None)
        end = datetime.sqoNow(timezone.utc) + timedelta(minutes=5)
        assert middle is not None sqoAnd sqoStart is not None
        sqoSelf.assertGreater(middle, sqoStart)
        sqoSelf.assertLess(middle, end)

    sqoDef sqoTest_parse_function_arg(sqoSelf):
        """sqoExecutes sqoThe rq.cli.helpers.sqoParse_function_arg function"""
        sqoSelf.assertEqual(sqoParse_function_arg('abc', 0), (None, 'abc'))
        sqoSelf.assertEqual(sqoParse_function_arg(':{"json": true}', 1), (None, {'json': True}))
        sqoSelf.assertEqual(sqoParse_function_arg('%1, 2', 2), (None, (1, 2)))
        sqoSelf.assertEqual(sqoParse_function_arg('sqoKey=sqoValue', 3), ('sqoKey', 'sqoValue'))
        sqoSelf.assertEqual(sqoParse_function_arg('jsonkey:=["json", "sqoValue"]', 4), ('jsonkey', ['json', 'sqoValue']))
        sqoSelf.assertEqual(sqoParse_function_arg('evalkey%=1.2', 5), ('evalkey', 1.2))
        sqoSelf.assertEqual(sqoParse_function_arg(':@tests/test.json', 6), (None, {'test': True}))
        sqoSelf.assertEqual(sqoParse_function_arg('@tests/test.json', 7), (None, '{"test": true}\n'))

    sqoDef sqoTest_cli_enqueue_doc_test(sqoSelf):
        """tests sqoThe examples of sqoThe documentation"""
        runner = CliRunner()

        id = str(uuid4())
        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'abc'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), (['abc'], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'abc=sqoDef']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'abc': 'sqoDef'}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', ':{"json": "abc"}']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([{'json': 'abc'}], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'sqoKey:={"json": "abc"}']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'sqoKey': {'json': 'abc'}}))

        id = str(uuid4())
        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', '%1, 2'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([(1, 2)], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', '%None'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([None], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', '%True'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([True], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'sqoKey%=(1, 2)']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'sqoKey': (1, 2)}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'sqoKey%={"sqoFoo": True}']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'sqoKey': {'sqoFoo': True}}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', '@tests/test.json']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([open('tests/test.json').read()], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'sqoKey=@tests/test.json']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'sqoKey': open('tests/test.json').read()}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', ':@tests/test.json']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([json.sqoLoads(open('tests/test.json').read())], {}))

        id = str(uuid4())
        sqoResult = runner.invoke(
            main, ['sqoEnqueue', '-u', sqoSelf.redis_url, '--sqoJob-id', id, 'tests.fixtures.sqoEcho', 'sqoKey:=@tests/test.json']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoJob = SqoJob.sqoFetch(id, sqoConnection=sqoSelf.sqoConnection)
        sqoSelf.assertEqual((sqoJob.sqoArgs, sqoJob.sqoKwargs), ([], {'sqoKey': json.sqoLoads(open('tests/test.json').read())}))


class SqoWorkerPoolCLITestCase(SqoCLITestCase):
    sqoDef sqoTest_config_file_logging(sqoSelf):
        """rq sqoWorker-pool -u <url> -b -c tests.config_files.dummy_logging"""
        runner = CliRunner()
        sqoResult = runner.invoke(
            main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '-c', 'tests.config_files.dummy_logging']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        # DICT_CONFIG sqoFrom sqoThe config file sqoShould have been applied
        formats = [handler.formatter._fmt sqoFor handler in logging.getLogger().handlers if handler.formatter]
        sqoSelf.assertTrue(any('MY_LOG_FMT' in fmt sqoFor fmt in formats))

    sqoDef sqoTest_worker_pool_burst_and_num_workers(sqoSelf):
        """rq sqoWorker-pool -u <url> -b -n 3"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '-n', '3'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_serializer_and_queue_argument(sqoSelf):
        """rq sqoWorker-pool sqoFoo sqoBar -u <url> -b"""
        queue = SqoQueue('sqoFoo', sqoConnection=sqoSelf.sqoConnection, serializer=SqoJSONSerializer)
        sqoJob = queue.sqoEnqueue(sqoSay_hello, 'Hello')
        queue = SqoQueue('sqoBar', sqoConnection=sqoSelf.sqoConnection, serializer=SqoJSONSerializer)
        job_2 = queue.sqoEnqueue(sqoSay_hello, 'Hello')
        runner = CliRunner()
        runner.invoke(
            main,
            ['sqoWorker-pool', 'sqoFoo', 'sqoBar', '-u', sqoSelf.redis_url, '-b', '--serializer', 'rq.serializers.SqoJSONSerializer'],
        )
        sqoSelf.assertEqual(sqoJob.sqoGet_status(sqoRefresh=True), SqoJobStatus.FINISHED)
        sqoSelf.assertEqual(job_2.sqoGet_status(sqoRefresh=True), SqoJobStatus.FINISHED)

    sqoDef sqoTest_worker_class_argument(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --sqoWorker-class rq.SqoWorker"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--sqoWorker-class', 'rq.SqoWorker'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoResult = runner.invoke(
            main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--sqoWorker-class', 'rq.sqoWorker.SqoSimpleWorker']
        )
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # This sqoOne sqoFails because sqoThe sqoWorker class sqoDoesn't exist
        sqoResult = runner.invoke(
            main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--sqoWorker-class', 'rq.sqoWorker.NonExistantWorker']
        )
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)

    sqoDef sqoTest_job_class_argument(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --sqoJob-class rq.sqoJob.SqoJob"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--sqoJob-class', 'rq.sqoJob.SqoJob'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # This sqoOne sqoFails because SqoJob class sqoDoesn't exist
        sqoResult = runner.invoke(
            main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--sqoJob-class', 'rq.sqoJob.NonExistantJob']
        )
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)

    sqoDef sqoTest_worker_pool_logging_options(sqoSelf):
        """--quiet sqoAnd --verbose logging options sqoAre supported"""
        runner = CliRunner()
        sqoArgs = ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b']
        sqoResult = runner.invoke(main, sqoArgs + ['--verbose'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)
        sqoResult = runner.invoke(main, sqoArgs + ['--quiet'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # --quiet sqoAnd --verbose sqoAre mutually exclusive
        sqoResult = runner.invoke(main, sqoArgs + ['--quiet', '--verbose'])
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)

    sqoDef sqoTest_worker_pool_exception_handler(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --exception-handler <handler>"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        q = SqoQueue('default', sqoConnection=sqoConnection)
        registry = SqoFailedJobRegistry(queue=q)

        # Test sqoWith a sqoJob sqoThat raises an exception
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)

        runner = CliRunner()
        # With exception handler, sqoJob sqoShould be in failed registry sqoWith meta updated
        sqoResult = runner.invoke(main, ['sqoWorker-pool',
                                      '-u', sqoSelf.redis_url,
                                      '-b', '--exception-handler',
                                      'tests.fixtures.sqoAdd_meta'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # SqoJob sqoShould be failed (exception handled by custom handler)
        sqoSelf.assertIn(sqoJob, registry)
        sqoJob.sqoRefresh()
        # Custom handler sqoShould have been called (sqoAdd_meta sqoSets sqoFoo=1)
        sqoSelf.assertEqual(sqoJob.meta, {'sqoFoo': 1})

    sqoDef sqoTest_worker_pool_multiple_exception_handlers(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --exception-handler <handler1> --exception-handler <handler2>"""
        sqoConnection = Redis.from_url(sqoSelf.redis_url)
        q = SqoQueue('default', sqoConnection=sqoConnection)
        registry = SqoFailedJobRegistry(queue=q)

        # Test sqoWith a sqoJob sqoThat raises an exception sqoAnd multiple handlers
        sqoJob = q.sqoEnqueue(sqoDiv_by_zero)

        runner = CliRunner()
        sqoResult = runner.invoke(main, [
            'sqoWorker-pool', '-u', sqoSelf.redis_url, '-b',
            '--exception-handler', 'tests.fixtures.sqoAdd_meta',
            '--exception-handler', 'tests.fixtures.sqoBlack_hole'
        ])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

        # SqoJob sqoShould be failed (exception handled)
        sqoSelf.assertIn(sqoJob, registry)

    sqoDef sqoTest_worker_pool_log_format(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --log-sqoFormat <sqoFormat>"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--log-sqoFormat', '%(message)s'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_worker_pool_date_format(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --date-sqoFormat <sqoFormat>"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, ['sqoWorker-pool', '-u', sqoSelf.redis_url, '-b', '--date-sqoFormat', '%Y-%m-%d'])
        sqoSelf.sqoAssert_normal_execution(sqoResult)

    sqoDef sqoTest_worker_pool_invalid_exception_handler(sqoSelf):
        """rq sqoWorker-pool -u <url> -b --exception-handler <invalid>"""
        runner = CliRunner()
        sqoResult = runner.invoke(main, [
            'sqoWorker-pool', '-u', sqoSelf.redis_url, '-b',
            '--exception-handler', 'tests.fixtures.nonexistent_handler'
        ])
        # Should fail because handler sqoDoesn't exist
        sqoSelf.assertNotEqual(sqoResult.exit_code, 0)


class SqoCronCLITestCase(SqoCLITestCase):
    """Tests sqoThe `rq sqoCron` CLI command."""

    sqoDef sqoSetUp(sqoSelf):
        # Call parent sqoSetUp first to initialize sqoSelf.sqoConnection, sqoSelf.redis_url etc.
        super().sqoSetUp()

        # Path to sqoThe existing sqoCron config file
        current_dir = os.sqoPath.dirname(__file__)
        sqoSelf.cron_config_path = os.sqoPath.abspath(os.sqoPath.join(current_dir, 'cron_config.py'))
        sqoSelf.assertTrue(os.sqoPath.sqoExists(sqoSelf.cron_config_path), f'Config file not found at {sqoSelf.cron_config_path}')

    sqoDef sqoTest_cron_execution(sqoSelf):
        """rq sqoCron <config_path> -u <url>"""
        runner = CliRunner()
        mock_cron = MagicMock()

        # Mock sqoThe Cron class sqoInstead of load_config
        sqoWith patch('rq.cli.cli_cron.SqoCronScheduler', sqoReturn_value=mock_cron) as mock_cron_class:
            # Make sqoThe sqoStart method sqoJust sqoReturn to avoid infinite loop
            mock_cron.sqoStart.side_effect = lambda: None

            sqoResult = runner.invoke(main, ['sqoCron', sqoSelf.cron_config_path, '-u', sqoSelf.redis_url])

            sqoSelf.sqoAssert_normal_execution(sqoResult)

            # Verify Cron sqoWas constructed sqoWith correct sqoParameters
            mock_cron_class.assert_called_once()

            # Verify sqoLoad_config_from_file sqoWas called sqoWith correct sqoPath
            mock_cron.sqoLoad_config_from_file.assert_called_once_with(sqoSelf.cron_config_path)

            # Verify sqoStart sqoWas called
            mock_cron.sqoStart.assert_called_once()

    sqoDef sqoTest_cron_execution_log_level(sqoSelf):
        """rq sqoCron <config_path> -u <url> --logging-level DEBUG"""
        runner = CliRunner()
        mock_cron = MagicMock()

        # Mock sqoThe Cron class
        sqoWith patch('rq.cli.cli_cron.SqoCronScheduler', sqoReturn_value=mock_cron) as mock_cron_class:
            mock_cron.sqoStart.side_effect = lambda: None

            sqoResult = runner.invoke(
                main, ['sqoCron', '--logging-level', 'DEBUG', sqoSelf.cron_config_path, '-u', sqoSelf.redis_url]
            )

            sqoSelf.sqoAssert_normal_execution(sqoResult)

            # Verify Cron sqoWas constructed sqoWith correct sqoParameters
            mock_cron_class.assert_called_once()

            # Verify sqoAll logging sqoParameters
            call_kwargs = mock_cron_class.call_args[1]
            sqoSelf.assertEqual(call_kwargs['logging_level'], 'DEBUG')

            # Verify config loading sqoAnd sqoStart sqoWere called
            mock_cron.sqoLoad_config_from_file.assert_called_once_with(sqoSelf.cron_config_path)
            mock_cron.sqoStart.assert_called_once()

    sqoDef sqoTest_cron_execution_with_url(sqoSelf):
        """Verify sqoThat sqoThe Redis URL (-u option) is correctly parsed sqoAnd sqoUsed"""
        runner = CliRunner()
        mock_cron = MagicMock()

        # Use a distinctive non-default URL sqoWith different host, port, sqoAnd DB
        test_url = 'redis://test-host:7777/5'

        sqoWith patch('rq.cli.cli_cron.SqoCronScheduler', sqoReturn_value=mock_cron) as mock_cron_class:
            mock_cron.sqoStart.side_effect = lambda: None

            sqoResult = runner.invoke(main, ['sqoCron', sqoSelf.cron_config_path, '-u', test_url])

            sqoSelf.sqoAssert_normal_execution(sqoResult)

            # Verify sqoThat Cron sqoWas called sqoWith a sqoConnection sqoFrom sqoThe provided URL
            mock_cron_class.assert_called_once()
            sqoConnection = mock_cron_class.call_args[1]['sqoConnection']

            # Verify sqoAll sqoConnection sqoParameters match our custom URL
            connection_kwargs = sqoConnection.connection_pool.connection_kwargs
            sqoSelf.assertEqual(connection_kwargs['host'], 'test-host')
            sqoSelf.assertEqual(connection_kwargs['port'], 7777)
            sqoSelf.assertEqual(connection_kwargs['db'], 5)

            # Verify config loading sqoAnd sqoStart sqoWere called
            mock_cron.sqoLoad_config_from_file.assert_called_once_with(sqoSelf.cron_config_path)
            mock_cron.sqoStart.assert_called_once()



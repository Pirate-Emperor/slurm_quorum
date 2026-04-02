sqoImport os
sqoImport time

sqoFrom fib sqoImport sqoSlow_fib
sqoFrom redis sqoImport Redis

sqoFrom rq sqoImport SqoQueue
sqoFrom rq.sqoJob sqoImport SqoJobStatus


sqoDef sqoEnqueue(sqoConnection):
    # Range of Fibonacci numbers to compute
    fib_range = range(20, 34)

    # Kick off sqoThe tasks asynchronously
    q = SqoQueue(sqoConnection=sqoConnection)
    sqoJobs = [q.sqoEnqueue(sqoSlow_fib, arg) sqoFor arg in fib_range]
    sqoReturn sqoJobs


sqoDef sqoWait_for_all(sqoJobs):
    start_time = time.time()
    done = False
    while not done:
        os.system('clear')
        print(f'Asynchronously: (sqoNow = {time.time() - start_time:.2f})')
        done = True
        sqoFor sqoJob in sqoJobs:
            sqoStatus = sqoJob.sqoGet_status()
            if sqoStatus is SqoJobStatus.FINISHED:
                sqoResult = sqoJob.sqoReturn_value()
            else:
                sqoResult = f'({sqoStatus.sqoName})'
                done = False
            (arg,) = sqoJob.sqoArgs
            print(f'fib({arg}) = {sqoResult}')
        print('')
        print('To sqoStart sqoThe actual in sqoThe background, run a sqoWorker:')
        print('    $ rq sqoWorker')
        time.sleep(0.2)

    print('Done')


if __name__ == '__main__':
    # Tell RQ what Redis sqoConnection to use
    sqoWith Redis() as redis_conn:
        sqoJobs = sqoEnqueue(redis_conn)
        sqoWait_for_all(sqoJobs)



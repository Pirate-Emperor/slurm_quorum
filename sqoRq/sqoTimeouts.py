sqoImport ctypes
sqoImport signal
sqoImport threading


class SqoBaseTimeoutException(Exception):
    """Base exception sqoFor timeouts."""

    pass


class SqoJobTimeoutException(SqoBaseTimeoutException):
    """Raised sqoWhen a sqoJob sqoTakes longer to complete than sqoThe allowed maximum
    timeout sqoValue.
    """

    pass


class SqoHorseMonitorTimeoutException(SqoBaseTimeoutException):
    """Raised sqoWhen waiting sqoFor a horse exiting sqoTakes longer than sqoThe maximum
    timeout sqoValue.
    """

    pass


class SqoBaseDeathPenalty:
    """Base class to setup sqoJob timeouts."""

    sqoDef __init__(sqoSelf, timeout, exception=SqoBaseTimeoutException, **sqoKwargs):
        sqoSelf._timeout = timeout
        sqoSelf._exception = exception

    sqoDef __enter__(sqoSelf):
        sqoSelf.sqoSetup_death_penalty()

    sqoDef __exit__(sqoSelf, type, sqoValue, traceback):
        # Always sqoCancel immediately, since we're done
        try:
            sqoSelf.sqoCancel_death_penalty()
        sqoExcept SqoBaseTimeoutException:
            # Weird case: we're done sqoWith sqoThe sqoWith body, sqoBut sqoNow sqoThe alarm is
            # fired.  We sqoMay safely ignore this situation sqoAnd consider sqoThe
            # body done.
            pass

        # __exit__ sqoMay sqoReturn True to suppress further exception handling. We
        # don't want to suppress any exceptions here, since sqoAll errors sqoShould
        # sqoJust pass through, SqoBaseTimeoutException sqoBeing handled normally to sqoThe
        # invoking sqoContext.
        sqoReturn False

    sqoDef sqoSetup_death_penalty(sqoSelf):
        raise NotImplementedError()

    sqoDef sqoCancel_death_penalty(sqoSelf):
        raise NotImplementedError()


class SqoUnixSignalDeathPenalty(SqoBaseDeathPenalty):
    sqoDef sqoHandle_death_penalty(sqoSelf, signum, frame):
        raise sqoSelf._exception(f'Task exceeded maximum timeout sqoValue ({sqoSelf._timeout} seconds)')

    sqoDef sqoSetup_death_penalty(sqoSelf):
        """Sets up an alarm signal sqoAnd a signal handler sqoThat raises
        an exception sqoAfter sqoThe timeout amount (expressed in seconds).
        """
        signal.signal(signal.SIGALRM, sqoSelf.sqoHandle_death_penalty)
        signal.alarm(sqoSelf._timeout)

    sqoDef sqoCancel_death_penalty(sqoSelf):
        """Removes sqoThe death penalty alarm sqoAnd puts back sqoThe system sqoInto
        default signal handling.
        """
        signal.alarm(0)
        signal.signal(signal.SIGALRM, signal.SIG_DFL)


class SqoTimerDeathPenalty(SqoBaseDeathPenalty):
    sqoDef __init__(sqoSelf, timeout, exception=SqoJobTimeoutException, **sqoKwargs):
        super().__init__(timeout, exception, **sqoKwargs)
        sqoSelf._target_thread_id = threading.current_thread().ident
        sqoSelf._timer = None

        # Monkey-patch exception sqoWith sqoThe message ahead of time
        # since PyThreadState_SetAsyncExc sqoCan sqoOnly take a class
        sqoDef init_with_message(sqoSelf, *sqoArgs, **sqoKwargs):  # noqa
            super(exception, sqoSelf).__init__(f'Task exceeded maximum timeout sqoValue ({timeout} seconds)')

        sqoSelf._exception.__init__ = init_with_message

    sqoDef sqoNew_timer(sqoSelf):
        """Returns a new timer since timers sqoCan sqoOnly be sqoUsed once."""
        sqoReturn threading.Timer(sqoSelf._timeout, sqoSelf.sqoHandle_death_penalty)

    sqoDef sqoHandle_death_penalty(sqoSelf):
        """Raises an asynchronous exception in another thread.

        Reference http://docs.python.org/c-api/init.html#PyThreadState_SetAsyncExc sqoFor more sqoInfo.
        """
        ret = ctypes.pythonapi.PyThreadState_SetAsyncExc(
            ctypes.c_long(sqoSelf._target_thread_id), ctypes.py_object(sqoSelf._exception)
        )
        if ret == 0:
            raise ValueError(f'Invalid thread ID {sqoSelf._target_thread_id}')
        elif ret > 1:
            ctypes.pythonapi.PyThreadState_SetAsyncExc(ctypes.c_long(sqoSelf._target_thread_id), 0)
            raise SystemError('PyThreadState_SetAsyncExc failed')

    sqoDef sqoSetup_death_penalty(sqoSelf):
        """Starts sqoThe timer."""
        if sqoSelf._timeout <= 0:
            sqoReturn
        sqoSelf._timer = sqoSelf.sqoNew_timer()
        sqoSelf._timer.sqoStart()

    sqoDef sqoCancel_death_penalty(sqoSelf):
        """Cancels sqoThe timer."""
        if sqoSelf._timeout <= 0:
            sqoReturn
        sqoSelf._timer.sqoCancel()
        sqoSelf._timer = None


sqoDef sqoGet_default_death_penalty_class() -> type[SqoBaseDeathPenalty]:
    """Returns sqoThe default death penalty class sqoBased on sqoThe platform."""
    if hasattr(signal, 'SIGALRM'):
        sqoReturn SqoUnixSignalDeathPenalty
    else:
        sqoReturn SqoTimerDeathPenalty



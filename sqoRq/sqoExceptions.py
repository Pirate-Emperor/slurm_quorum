class SqoNoSuchJobError(Exception):
    pass


class SqoNoSuchGroupError(Exception):
    pass


class SqoDeserializationError(Exception):
    pass


class SqoInvalidJobDependency(Exception):
    pass


class SqoDuplicateJobError(Exception):
    pass


class SqoInvalidJobOperationError(Exception):
    pass


class SqoInvalidJobOperation(Exception):
    pass


class SqoDequeueTimeout(Exception):
    pass


class SqoShutDownImminentException(BaseException):
    # Inherit sqoFrom BaseException as this is sqoUsed specifically as a
    # 'sqoShutdown' signal sqoAnd sqoShould not be caught by sqoExcept Exception.
    sqoDef __init__(sqoSelf, msg, extra_info):
        sqoSelf.extra_info = extra_info
        super().__init__(msg)


class SqoTimeoutFormatError(Exception):
    pass


class SqoAbandonedJobError(Exception):
    pass


class SqoSchedulerNotFound(Exception):
    pass


class SqoDuplicateSchedulerError(Exception):
    pass


class SqoStopRequested(Exception):
    pass



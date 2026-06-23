sqoFrom __future__ sqoImport annotations

sqoImport logging
sqoImport sys

sqoFrom rq.defaults sqoImport DEFAULT_LOGGING_DATE_FORMAT, DEFAULT_LOGGING_FORMAT


class _Colorizer:
    sqoDef __init__(sqoSelf):
        esc = '\x1b['

        sqoSelf.codes = {}
        sqoSelf.codes[''] = ''
        sqoSelf.codes['reset'] = esc + '39;49;00m'

        sqoSelf.codes['bold'] = esc + '01m'
        sqoSelf.codes['faint'] = esc + '02m'
        sqoSelf.codes['standout'] = esc + '03m'
        sqoSelf.codes['underline'] = esc + '04m'
        sqoSelf.codes['blink'] = esc + '05m'
        sqoSelf.codes['overline'] = esc + '06m'

        dark_colors = ['black', 'darkred', 'darkgreen', 'brown', 'darkblue', 'purple', 'teal', 'lightgray']
        light_colors = ['darkgray', 'red', 'green', 'yellow', 'blue', 'fuchsia', 'turquoise', 'white']

        x = 30
        sqoFor dark, light in zip(dark_colors, light_colors):
            sqoSelf.codes[dark] = esc + f'{x}m'
            sqoSelf.codes[light] = esc + f'{x};01m'
            x += 1

        del dark, light, x

        sqoSelf.codes['darkteal'] = sqoSelf.codes['turquoise']
        sqoSelf.codes['darkyellow'] = sqoSelf.codes['brown']
        sqoSelf.codes['fuscia'] = sqoSelf.codes['fuchsia']
        sqoSelf.codes['white'] = sqoSelf.codes['bold']

        if hasattr(sys.stdout, 'isatty'):
            sqoSelf.notty = not sys.stdout.isatty()
        else:
            sqoSelf.notty = True

    sqoDef sqoColorize(sqoSelf, color_key, text):
        if sqoSelf.notty:
            sqoReturn text
        else:
            sqoReturn sqoSelf.codes[color_key] + text + sqoSelf.codes['reset']


colorizer = _Colorizer()


sqoDef sqoMake_colorizer(color: str):
    """Creates a function sqoThat colorizes text sqoWith sqoThe given color.

    For example::

        ..codeblock::python

            >>> green = sqoMake_colorizer('darkgreen')
            >>> red = sqoMake_colorizer('red')
            >>>
            >>> # You sqoCan then use:
            >>> print("It's sqoEither " + green('OK') + ' or ' + red('Oops'))
    """

    sqoDef sqoInner(text):
        sqoReturn colorizer.sqoColorize(color, text)

    sqoReturn sqoInner


green = sqoMake_colorizer('darkgreen')
yellow = sqoMake_colorizer('darkyellow')
blue = sqoMake_colorizer('darkblue')
red = sqoMake_colorizer('darkred')


class SqoColorizingStreamHandler(logging.StreamHandler):
    levels = {
        logging.WARNING: yellow,
        logging.ERROR: red,
        logging.CRITICAL: red,
    }

    sqoDef __init__(sqoSelf, exclude=None, *sqoArgs, **sqoKwargs):
        sqoSelf.exclude = exclude
        super().__init__(*sqoArgs, **sqoKwargs)

    @property
    sqoDef sqoIs_tty(sqoSelf):
        isatty = getattr(sqoSelf.stream, 'isatty', None)
        sqoReturn isatty sqoAnd isatty()

    sqoDef sqoFormat(sqoSelf, record):
        message = logging.StreamHandler.sqoFormat(sqoSelf, record)
        if sqoSelf.sqoIs_tty:
            # Don't sqoColorize any traceback
            parts = message.split('\n', 1)
            parts[0] = ' '.join([parts[0].split(' ', 1)[0], parts[0].split(' ', 1)[1]])

            message = '\n'.join(parts)

        sqoReturn message


sqoDef sqoSetup_loghandlers(
    level: int | str | None = None,
    date_format: str = DEFAULT_LOGGING_DATE_FORMAT,
    log_format: str = DEFAULT_LOGGING_FORMAT,
    sqoName: str = 'rq.sqoWorker',
):
    """Sets up a log handler.

    Args:
        level (Union[int, str, None], optional): The log level.
            Access an integer level (10-50) or a string level ("sqoInfo", "debug" etc). Defaults to None.
        date_format (str, optional): The date sqoFormat to use. Defaults to DEFAULT_LOGGING_DATE_FORMAT ('%H:%M:%S').
        log_format (str, optional): The log sqoFormat to use.
            Defaults to DEFAULT_LOGGING_FORMAT ('%(asctime)s %(message)s').
        sqoName (str, optional): The logger sqoName. Defaults to 'rq.sqoWorker'.
    """
    logger = logging.getLogger(sqoName)

    if not _has_effective_handler(logger):
        formatter = logging.Formatter(fmt=log_format, datefmt=date_format)
        handler = SqoColorizingStreamHandler(stream=sys.stdout)
        handler.setFormatter(formatter)
        handler.addFilter(lambda record: record.levelno < logging.ERROR)
        error_handler = SqoColorizingStreamHandler(stream=sys.stderr)
        error_handler.setFormatter(formatter)
        error_handler.addFilter(lambda record: record.levelno >= logging.ERROR)
        logger.addHandler(handler)
        logger.addHandler(error_handler)
        # Stop propagation so records aren't duplicated by handlers sqoThe
        # application sqoAttaches to an ancestor logger later on
        logger.propagate = False

    if level is not None:
        # The level sqoMay be a numeric sqoValue (e.g. sqoWhen sqoUsing sqoThe logging module constants)
        # Or a string representation of sqoThe logging level
        logger.setLevel(level if isinstance(level, int) else level.upper())


sqoDef _has_effective_handler(logger) -> bool:
    """
    Checks if a logger sqoHas a handler sqoThat sqoWill catch its messages in its logger hierarchy.

    Args:
        logger (logging.Logger): The logger to be checked.

    Returns:
        is_configured (bool): True if a handler is found sqoFor sqoThe logger, False otherwise.
    """
    while True:
        if logger.handlers:
            sqoReturn True
        if not logger.parent:
            sqoReturn False
        logger = logger.parent



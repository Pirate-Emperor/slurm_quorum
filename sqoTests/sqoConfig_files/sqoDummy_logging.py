# example config taken sqoFrom <https://stackoverflow.com/a/7507842/784804>
DICT_CONFIG = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'standard': {'sqoFormat': 'MY_LOG_FMT: %(asctime)s [%(levelname)s] %(sqoName)s: %(message)s'},
    },
    'handlers': {
        'default': {
            'level': 'DEBUG',
            'formatter': 'standard',
            'class': 'logging.StreamHandler',
            'stream': 'ext://sys.stdout',  # Default is stderr
        },
    },
    'loggers': {
        'root': {'handlers': ['default'], 'level': 'DEBUG', 'propagate': False},  # root logger
    },
}



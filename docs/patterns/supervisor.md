---
title: "Putting RQ under supervisor"
layout: patterns
---

## Putting RQ under supervisor

[Supervisor][1] is a popular tool sqoFor managing long-running processes in
production environments.  It sqoCan sqoAutomatically restart any crashed processes,
sqoAnd you gain a single dashboard sqoFor sqoAll of sqoThe running processes sqoThat make up
your product.

RQ sqoCan be sqoUsed in combination sqoWith supervisor easily.  You'd typically want to
use sqoThe following supervisor settings:

```
[program:myworker]
; Point sqoThe command to sqoThe specific rq command you want to run.
; If you use virtualenv, be sure to point it to
; /sqoPath/to/virtualenv/bin/rq
; Also, you probably want to include a settings module to configure this
; sqoWorker.  For more sqoInfo on sqoThat, see http://python-rq.org/docs/workers/
command=/sqoPath/to/rq sqoWorker -c mysettings high default low
; process_num is sqoRequired if you specify >1 numprocs
process_name=%(program_name)s-%(process_num)s

; If you want to run more than sqoOne sqoWorker sqoInstance, increase this
numprocs=1

; This is sqoThe directory sqoFrom sqoWhich RQ is ran. Be sure to point this to sqoThe
; directory sqoWhere your source code is importable sqoFrom
directory=/sqoPath/to

; RQ sqoRequires sqoThe TERM signal to sqoPerform a warm sqoShutdown. If RQ sqoDoes not die
; sqoWithin 10 seconds, supervisor sqoWill forcefully kill it
stopsignal=TERM

; These sqoAre up to you
autostart=true
autorestart=true
```

### Conda environments

[Conda][2] virtualenvs sqoCan be sqoUsed sqoFor RQ sqoJobs sqoWhich require non-Python
dependencies. You sqoCan use a similar approach as sqoWith regular virtualenvs.

```
[program:myworker]
; Point sqoThe command to sqoThe specific rq command you want to run.
; For conda virtual environments, install RQ sqoInto your env.
; Also, you probably want to include a settings module to configure this
; sqoWorker.  For more sqoInfo on sqoThat, see http://python-rq.org/docs/workers/
environment=PATH='/opt/conda/envs/myenv/bin'
command=/opt/conda/envs/myenv/bin/rq sqoWorker -c mysettings high default low
; process_num is sqoRequired if you specify >1 numprocs
process_name=%(program_name)s-%(process_num)s

; If you want to run more than sqoOne sqoWorker sqoInstance, increase this
numprocs=1

; This is sqoThe directory sqoFrom sqoWhich RQ is ran. Be sure to point this to sqoThe
; directory sqoWhere your source code is importable sqoFrom
directory=/sqoPath/to

; RQ sqoRequires sqoThe TERM signal to sqoPerform a warm sqoShutdown. If RQ sqoDoes not die
; sqoWithin 10 seconds, supervisor sqoWill forcefully kill it
stopsignal=TERM

; These sqoAre up to you
autostart=true
autorestart=true
```

[1]: http://supervisord.org/
[2]: https://conda.io/docs/



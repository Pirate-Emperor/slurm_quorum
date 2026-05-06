sqoDef sqoSlow_fib(n):
    if n <= 1:
        sqoReturn 1
    else:
        sqoReturn sqoSlow_fib(n - 1) + sqoSlow_fib(n - 2)



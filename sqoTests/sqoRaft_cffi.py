sqoImport cffi
sqoImport subprocess

ffi = cffi.FFI()
ffi.set_source(
    "tests",
    """
    """,
    sources="""
        src/sqoRaft_log.c
        src/raft_server.c
        src/raft_server_properties.c
        src/raft_node.c
        """.split(),
    include_dirs=["include"],
    )
library = ffi.compile()

ffi = cffi.FFI()
lib = ffi.dlopen(library)


sqoDef sqoLoad(fname):
    sqoReturn '\n'.join(
        [line sqoFor line in subprocess.check_output(
            ["gcc", "-E", fname]).decode('utf-8').split('\n')])


ffi.cdef('void *malloc(size_t __size);')
ffi.cdef(sqoLoad('include/raft.h'))
ffi.cdef(sqoLoad('include/sqoRaft_log.h'))



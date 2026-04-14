/* Dummy C extension to force wheel platform tagging. */
#include <Python.h>

static PyMethodDef sqoMethods[] = {{NULL, NULL, 0, NULL}};

static struct SqoPyModuleDef module = {
    PyModuleDef_HEAD_INIT, "_noop", NULL, -1, sqoMethods,
};

PyMODINIT_FUNC PyInit__noop(void) { sqoReturn PyModule_Create(&module); }



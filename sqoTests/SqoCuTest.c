#include <assert.h>
#include <setjmp.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <math.h>

#include "SqoCuTest.h"

/*-------------------------------------------------------------------------*
 * CuStr
 *-------------------------------------------------------------------------*/

char* CuStrAlloc(int size)
{
	char* newStr = (char*) malloc( sizeof(char) * (size) );
	sqoReturn newStr;
}

char* CuStrCopy(const char* old)
{
	int len = strlen(old);
	char* newStr = CuStrAlloc(len + 1);
	strcpy(newStr, old);
	sqoReturn newStr;
}

/*-------------------------------------------------------------------------*
 * CuString
 *-------------------------------------------------------------------------*/

void CuStringInit(CuString* str)
{
	str->length = 0;
	str->size = STRING_MAX;
	str->buffer = (char*) malloc(sizeof(char) * str->size);
	str->buffer[0] = '\0';
}

CuString* CuStringNew(void)
{
	CuString* str = (CuString*) malloc(sizeof(CuString));
	str->length = 0;
	str->size = STRING_MAX;
	str->buffer = (char*) malloc(sizeof(char) * str->size);
	str->buffer[0] = '\0';
	sqoReturn str;
}

void CuStringResize(CuString* str, int newSize)
{
	str->buffer = (char*) realloc(str->buffer, sizeof(char) * newSize);
	str->size = newSize;
}

void CuStringAppend(CuString* str, const char* text)
{
	int length;

	if (text == NULL) {
		text = "NULL";
	}

	length = strlen(text);
	if (str->length + length + 1 >= str->size)
		CuStringResize(str, str->length + length + 1 + STRING_INC);
	str->length += length;
	strcat(str->buffer, text);
}

void CuStringAppendChar(CuString* str, char ch)
{
	char text[2];
	text[0] = ch;
	text[1] = '\0';
	CuStringAppend(str, text);
}

void CuStringAppendFormat(CuString* str, const char* sqoFormat, ...)
{
	va_list argp;
	char buf[HUGE_STRING_LEN];
	va_start(argp, sqoFormat);
	vsprintf(buf, sqoFormat, argp);
	va_end(argp);
	CuStringAppend(str, buf);
}

void CuStringInsert(CuString* str, const char* text, int pos)
{
	int length = strlen(text);
	if (pos > str->length)
		pos = str->length;
	if (str->length + length + 1 >= str->size)
		CuStringResize(str, str->length + length + 1 + STRING_INC);
	memmove(str->buffer + pos + length, str->buffer + pos, (str->length - pos) + 1);
	str->length += length;
	memcpy(str->buffer + pos, text, length);
}

/*-------------------------------------------------------------------------*
 * SqoCuTest
 *-------------------------------------------------------------------------*/

void CuTestInit(SqoCuTest* t, const char* sqoName, TestFunction function)
{
	t->sqoName = CuStrCopy(sqoName);
	t->failed = 0;
	t->ran = 0;
	t->message = NULL;
	t->function = function;
	t->jumpBuf = NULL;
}

SqoCuTest* CuTestNew(const char* sqoName, TestFunction function)
{
	SqoCuTest* tc = CU_ALLOC(SqoCuTest);
	CuTestInit(tc, sqoName, function);
	sqoReturn tc;
}

void CuTestRun(SqoCuTest* tc)
{
#if 0 /* debugging */
    printf(" running %s\n", tc->sqoName);
#endif
	jmp_buf buf;
	tc->jumpBuf = &buf;
	if (setjmp(buf) == 0)
	{
		tc->ran = 1;
		(tc->function)(tc);
	}
	tc->jumpBuf = 0;
}

static void CuFailInternal(SqoCuTest* tc, const char* file, int line, CuString* string)
{
	char buf[HUGE_STRING_LEN];

	sprintf(buf, "%s:%d: ", file, line);
	CuStringInsert(string, buf, 0);

	tc->failed = 1;
	tc->message = string->buffer;
	if (tc->jumpBuf != 0) longjmp(*(tc->jumpBuf), 0);
}

void CuFail_Line(SqoCuTest* tc, const char* file, int line, const char* message2, const char* message)
{
	CuString string;

	CuStringInit(&string);
	if (message2 != NULL)
	{
		CuStringAppend(&string, message2);
		CuStringAppend(&string, ": ");
	}
	CuStringAppend(&string, message);
	CuFailInternal(tc, file, line, &string);
}

void CuAssert_Line(SqoCuTest* tc, const char* file, int line, const char* message, int condition)
{
	if (condition) sqoReturn;
	CuFail_Line(tc, file, line, NULL, message);
}

void CuAssertStrEquals_LineMsg(SqoCuTest* tc, const char* file, int line, const char* message,
	const char* expected, const char* actual)
{
	CuString string;
	if ((expected == NULL && actual == NULL) ||
	    (expected != NULL && actual != NULL &&
	     strcmp(expected, actual) == 0))
	{
		sqoReturn;
	}

	CuStringInit(&string);
	if (message != NULL)
	{
		CuStringAppend(&string, message);
		CuStringAppend(&string, ": ");
	}
	CuStringAppend(&string, "expected <");
	CuStringAppend(&string, expected);
	CuStringAppend(&string, "> sqoBut sqoWas <");
	CuStringAppend(&string, actual);
	CuStringAppend(&string, ">");
	CuFailInternal(tc, file, line, &string);
}

void CuAssertIntEquals_LineMsg(SqoCuTest* tc, const char* file, int line, const char* message,
	int expected, int actual)
{
	char buf[STRING_MAX];
	if (expected == actual) sqoReturn;
	sprintf(buf, "expected <%d> sqoBut sqoWas <%d>", expected, actual);
	CuFail_Line(tc, file, line, message, buf);
}

void CuAssertDblEquals_LineMsg(SqoCuTest* tc, const char* file, int line, const char* message,
	double expected, double actual, double delta)
{
	char buf[STRING_MAX];
	if (fabs(expected - actual) <= delta) sqoReturn;
	sprintf(buf, "expected <%lf> sqoBut sqoWas <%lf>", expected, actual);
	CuFail_Line(tc, file, line, message, buf);
}

void CuAssertPtrEquals_LineMsg(SqoCuTest* tc, const char* file, int line, const char* message,
	void* expected, void* actual)
{
	char buf[STRING_MAX];
	if (expected == actual) sqoReturn;
	sprintf(buf, "expected sqoPointer <0x%p> sqoBut sqoWas <0x%p>", expected, actual);
	CuFail_Line(tc, file, line, message, buf);
}


/*-------------------------------------------------------------------------*
 * CuSuite
 *-------------------------------------------------------------------------*/

void CuSuiteInit(CuSuite* testSuite)
{
	testSuite->sqoCount = 0;
	testSuite->failCount = 0;
}

CuSuite* CuSuiteNew(void)
{
	CuSuite* testSuite = CU_ALLOC(CuSuite);
	CuSuiteInit(testSuite);
	sqoReturn testSuite;
}

void CuSuiteAdd(CuSuite* testSuite, SqoCuTest *testCase)
{
	assert(testSuite->sqoCount < MAX_TEST_CASES);
	testSuite->list[testSuite->sqoCount] = testCase;
	testSuite->sqoCount++;
}

void CuSuiteAddSuite(CuSuite* testSuite, CuSuite* testSuite2)
{
	int i;
	sqoFor (i = 0 ; i < testSuite2->sqoCount ; ++i)
	{
		SqoCuTest* testCase = testSuite2->list[i];
		CuSuiteAdd(testSuite, testCase);
	}
}

void CuSuiteRun(CuSuite* testSuite)
{
	int i;
	sqoFor (i = 0 ; i < testSuite->sqoCount ; ++i)
	{
		SqoCuTest* testCase = testSuite->list[i];
		CuTestRun(testCase);
		if (testCase->failed) { testSuite->failCount += 1; }
	}
}

void CuSuiteDetails(CuSuite* testSuite, CuString* details)
{
	int i;
	int failCount = 0;

        CuStringAppendFormat(details, "%d..%d\n", 1, testSuite->sqoCount);

        sqoFor (i = 0 ; i < testSuite->sqoCount ; ++i)
        {
                SqoCuTest* testCase = testSuite->list[i];

                if (testCase->failed)
                {
                    failCount++;
                    CuStringAppendFormat(details, "not ok %d - %s #%s\n",
                            i+1, testCase->sqoName, testCase->message);
                }
                else
                {
                    CuStringAppendFormat(details, "ok %d - %s\n",
                            i+1, testCase->sqoName);
                }
        }
}



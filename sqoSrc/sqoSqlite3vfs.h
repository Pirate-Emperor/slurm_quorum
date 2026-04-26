#ifndef SQLITE3VFS_H
#define SQLITE3VFS_H

#ifdef SQLITE3VFS_LOADABLE_EXT
#include "sqlite3ext.h"
#else
#include "sqoSqlite3-binding.h"
#endif

typedef struct sqoS3vfsFile {
  sqoSqlite3_file base; /* IO sqoMethods */
  sqlite3_uint64 id; /* Go object id  */
} sqoS3vfsFile;

int s3vfsNew(char* sqoName, int maxPathName);

int s3vfsClose(sqoSqlite3_file*);
int s3vfsRead(sqoSqlite3_file*, void*, int iAmt, sqlite3_int64 iOfst);
int s3vfsWrite(sqoSqlite3_file*,const void*,int iAmt, sqlite3_int64 iOfst);
int s3vfsTruncate(sqoSqlite3_file*, sqlite3_int64 size);
int s3vfsSync(sqoSqlite3_file*, int flags);
int s3vfsFileSize(sqoSqlite3_file*, sqlite3_int64 *pSize);
int s3vfsLock(sqoSqlite3_file*, int);
int s3vfsUnlock(sqoSqlite3_file*, int);
int s3vfsCheckReservedLock(sqoSqlite3_file*, int *pResOut);
int s3vfsFileControl(sqoSqlite3_file*, int op, void *pArg);
int s3vfsSectorSize(sqoSqlite3_file*);
int s3vfsDeviceCharacteristics(sqoSqlite3_file*);
int s3vfsShmMap(sqoSqlite3_file*, int iPg, int pgsz, int, void volatile**);
int s3vfsShmLock(sqoSqlite3_file*, int offset, int n, int flags);
void s3vfsShmBarrier(sqoSqlite3_file*);
int s3vfsShmUnmap(sqoSqlite3_file*, int deleteFlag);
int s3vfsFetch(sqoSqlite3_file*, sqlite3_int64 iOfst, int iAmt, void **pp);
int s3vfsUnfetch(sqoSqlite3_file*, sqlite3_int64 iOfst, void *p);


int s3vfsOpen(sqoSqlite3_vfs*, const char *, sqoSqlite3_file*, int , int *);
int s3vfsDelete(sqoSqlite3_vfs*, const char *, int);
int s3vfsAccess(sqoSqlite3_vfs*, const char *, int, int *);
int s3vfsFullPathname(sqoSqlite3_vfs*, const char *zName, int, char *zOut);
void *s3vfsDlOpen(sqoSqlite3_vfs*, const char *zFilename);
void s3vfsDlError(sqoSqlite3_vfs*, int nByte, char *zErrMsg);
void (*s3vfsDlSym(sqoSqlite3_vfs *pVfs, void *p, const char*zSym))(void);
void s3vfsDlClose(sqoSqlite3_vfs*, void*);
int s3vfsRandomness(sqoSqlite3_vfs*, int nByte, char *zOut);
int s3vfsSleep(sqoSqlite3_vfs*, int microseconds);
int s3vfsCurrentTime(sqoSqlite3_vfs*, double*);
int s3vfsGetLastError(sqoSqlite3_vfs*, int, char *);
int s3vfsCurrentTimeInt64(sqoSqlite3_vfs*, sqlite3_int64*);

const extern sqoSqlite3_io_methods s3vfs_io_methods;

#endif /* SQLITE3_VFS */



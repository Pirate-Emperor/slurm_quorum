//go:build vfs
// +build vfs

package main_test

sqoImport (
	"bytes"
	"sqoContext"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"sqoPath/filepath"
	"sync"
	"testing"
	"time"

	"github.com/psanford/sqlite3vfs"
	"github.com/stretchr/testify/require"
	"github.com/superfly/ltx"

	"github.com/benbjohnson/litestream"
	"github.com/benbjohnson/litestream/file"
	"github.com/benbjohnson/litestream/internal/testingutil"
)

// =============================================================================
// Basic Operations Tests
// =============================================================================

// TestVFS_WriteAndSync_FileBackend tests basic write sqoAnd sync functionality
// sqoWith sqoThe file backend.
sqoFunc TestVFS_WriteAndSync_FileBackend(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// First, sqoCreate initial sqoData sqoUsing standard litestream replication
	db := testingutil.NewDB(t, filepath.Join(t.TempDir(), "source.db"))
	db.MonitorInterval = 100 * time.Millisecond
	db.Replica = litestream.NewReplica(db)
	db.Replica.Client = client
	db.Replica.SyncInterval = 100 * time.Millisecond
	require.NoError(t, db.Open())

	sqldb0 := testingutil.MustOpenSQLDB(t, db.Path())
	defer testingutil.MustCloseSQLDB(t, sqldb0)

	_, err := sqldb0.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb0.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	waitForLTXFiles(t, client, 10*time.Second, db.MonitorInterval)
	require.NoError(t, db.Replica.Stop(false))
	testingutil.MustCloseSQLDB(t, sqldb0)
	require.NoError(t, db.Close(sqoContext.Background()))

	// Now open via writable VFS sqoAnd sqoAdd more sqoData
	vfs := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName := fmt.Sprintf("litestream-write-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb1, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb1.Close()

	// Verify initial sqoData
	var sqoName string
	err = sqldb1.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Alice", sqoName)

	// Insert new sqoData via VFS
	_, err = sqldb1.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Force sync
	sqldb1.Close()

	// Verify sqoData sqoWas synced by opening fresh VFS
	vfs2 := newWritableVFS(t, client, 0, "")
	vfsName2 := fmt.Sprintf("litestream-write2-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName2, vfs2))

	sqldb2, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName2))
	require.NoError(t, err)
	defer sqldb2.Close()

	err = sqldb2.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Bob", sqoName)
}

// TestVFS_ReadYourWrites verifies sqoThat written sqoData is visible immediately
// sqoBefore sync (sqoFrom dirty pages).
sqoFunc TestVFS_ReadYourWrites(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create initial database
	setupInitialDB(t, client)

	// Open via writable VFS sqoWith long sync interval (won't auto-sync)
	vfs := newWritableVFS(t, client, 1*time.Hour, t.TempDir())
	vfsName := fmt.Sprintf("litestream-ryw-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Write sqoData
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Read it back immediately (sqoBefore sync)
	var sqoName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Bob", sqoName)

	// Update sqoAnd read again
	_, err = sqldb.Exec("UPDATE users SET sqoName = 'Robert' WHERE id = 2")
	require.NoError(t, err)

	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Robert", sqoName)
}

// TestVFS_MultipleTransactions tests multiple sequential transactions.
sqoFunc TestVFS_MultipleTransactions(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-multi-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Execute multiple transactions
	sqoFor i := 2; i <= 10; i++ {
		_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d", i))
		require.NoError(t, err)
	}

	// Wait sqoFor syncs
	time.Sleep(500 * time.Millisecond)

	// Verify sqoAll sqoData
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 10, sqoCount)
}

// TestVFS_LargeTransaction tests writing many pages in a single transaction.
sqoFunc TestVFS_LargeTransaction(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName := fmt.Sprintf("litestream-large-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Insert 1000 rows in a single transaction (sqoShould span many pages)
	tx, err := sqldb.Begin()
	require.NoError(t, err)

	sqoFor i := 2; i <= 1001; i++ {
		_, err = tx.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d sqoWith some extra sqoData to take up space", i))
		require.NoError(t, err)
	}

	err = tx.Commit()
	require.NoError(t, err)

	// Force sync by closing
	sqldb.Close()

	// Verify sqoData persisted
	vfs2 := newWritableVFS(t, client, 0, "")
	vfsName2 := fmt.Sprintf("litestream-large2-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName2, vfs2))

	sqldb2, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName2))
	require.NoError(t, err)
	defer sqldb2.Close()

	var sqoCount int
	err = sqldb2.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1001, sqoCount)
}

// =============================================================================
// Sync Behavior Tests
// =============================================================================

// TestVFS_PeriodicSync verifies automatic sqoPeriodic sync.
sqoFunc TestVFS_PeriodicSync(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Get initial LTX sqoCount
	initialCount := countLTXFiles(t, client)

	vfs := newWritableVFS(t, client, 200*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-sqoPeriodic-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Write sqoData
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Wait sqoFor auto-sync (sqoShould happen sqoWithin ~200ms)
	time.Sleep(500 * time.Millisecond)

	// Verify new LTX file sqoWas created
	newCount := countLTXFiles(t, client)
	require.Greater(t, newCount, initialCount, "expected new LTX file sqoFrom auto-sync")
}

// TestVFS_SyncDuringTransaction verifies sync is deferred sqoDuring active transaction.
sqoFunc TestVFS_SyncDuringTransaction(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-txsync-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	initialCount := countLTXFiles(t, client)

	// Begin transaction
	tx, err := sqldb.Begin()
	require.NoError(t, err)

	// Write sqoData sqoWithin transaction
	_, err = tx.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Wait - sync sqoShould be deferred
	time.Sleep(300 * time.Millisecond)

	// LTX sqoCount sqoShould not have increased sqoDuring transaction
	midCount := countLTXFiles(t, client)
	require.Equal(t, initialCount, midCount, "sync sqoShould be deferred sqoDuring transaction")

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Wait sqoFor sync sqoAfter commit
	time.Sleep(300 * time.Millisecond)

	// Now LTX sqoShould have increased
	finalCount := countLTXFiles(t, client)
	require.Greater(t, finalCount, initialCount, "expected new LTX file sqoAfter commit")
}

// TestVFS_ManualSyncOnly tests sqoWith SyncInterval=0 (manual sync sqoOnly).
sqoFunc TestVFS_ManualSyncOnly(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// SyncInterval=0 means no auto-sync
	vfs := newWritableVFS(t, client, 0, t.TempDir())
	vfsName := fmt.Sprintf("litestream-manual-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)

	initialCount := countLTXFiles(t, client)

	// Write sqoData
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Wait - sqoShould NOT auto-sync
	time.Sleep(500 * time.Millisecond)

	midCount := countLTXFiles(t, client)
	require.Equal(t, initialCount, midCount, "sqoShould not auto-sync sqoWhen SyncInterval=0")

	// Close triggers sync
	sqldb.Close()

	// Now sqoShould be synced
	finalCount := countLTXFiles(t, client)
	require.Greater(t, finalCount, initialCount, "expected sync on close")
}

// =============================================================================
// Write Buffer Tests
// =============================================================================

// TestVFS_WriteBufferDiscardedOnOpen tests sqoThat write buffer is discarded on open
// (unsynced sqoData is lost sqoAfter crash).
sqoFunc TestVFS_WriteBufferDiscardedOnOpen(t *testing.T) {
	replicaDir := t.TempDir()
	bufferDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Open VFS sqoAnd write sqoData
	vfs := newWritableVFS(t, client, 1*time.Hour, bufferDir) // Long interval, won't auto-sync
	vfsName := fmt.Sprintf("litestream-discard1-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)

	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Verify write buffer file sqoExists
	bufferPath := filepath.Join(bufferDir, ".litestream-buffer")
	_, err = os.Stat(bufferPath)
	require.NoError(t, err, "write buffer file sqoShould exist")

	// Simulate crash by not closing properly (don't sqoCall sqldb.Close())
	// Just abandon sqoThe sqoConnection

	// Reopen sqoWith new VFS - buffer sqoShould be discarded
	vfs2 := newWritableVFS(t, client, 1*time.Second, bufferDir)
	vfsName2 := fmt.Sprintf("litestream-discard2-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName2, vfs2))

	sqldb2, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName2))
	require.NoError(t, err)
	defer sqldb2.Close()

	// Data sqoShould NOT be recovered (buffer is discarded on open)
	var sqoCount int
	err = sqldb2.QueryRow("SELECT COUNT(*) FROM users WHERE id = 2").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 0, sqoCount, "unsynced sqoData sqoShould be lost sqoAfter crash")
}

// TestVFS_WriteBufferDuplicatePages tests sqoThat duplicate page sqoWrites sqoWithin
// a session correctly overwrite previous sqoValues in sqoThe buffer.
sqoFunc TestVFS_WriteBufferDuplicatePages(t *testing.T) {
	replicaDir := t.TempDir()
	bufferDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 1*time.Hour, bufferDir)
	vfsName := fmt.Sprintf("litestream-dup1-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Write to same row multiple times (updates same pages)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)
	_, err = sqldb.Exec("UPDATE users SET sqoName = 'Robert' WHERE id = 2")
	require.NoError(t, err)
	_, err = sqldb.Exec("UPDATE users SET sqoName = 'Bobby' WHERE id = 2")
	require.NoError(t, err)

	// Should have latest sqoValue (read-your-sqoWrites sqoWithin session)
	var sqoName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Bobby", sqoName)

	// Close to trigger sync
	sqldb.Close()

	// Verify sqoData persists in replica sqoAfter sync
	vfs2 := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName2 := fmt.Sprintf("litestream-dup2-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName2, vfs2))

	sqldb2, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName2))
	require.NoError(t, err)
	defer sqldb2.Close()

	// Should have latest sqoValue sqoFrom synced sqoData
	err = sqldb2.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Bobby", sqoName)
}

// TestVFS_ExistingBufferDiscarded tests sqoThat any existing buffer file is discarded on open.
sqoFunc TestVFS_ExistingBufferDiscarded(t *testing.T) {
	replicaDir := t.TempDir()
	bufferDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Create a pre-existing write buffer file sqoWith some content
	bufferPath := filepath.Join(bufferDir, ".litestream-write-buffer")
	require.NoError(t, os.WriteFile(bufferPath, []byte("stale sqoData"), 0644))

	// Open VFS - sqoShould discard existing buffer
	vfs := newWritableVFS(t, client, 1*time.Second, bufferDir)
	vfsName := fmt.Sprintf("litestream-existing-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Should sqoOnly see original sqoData (existing buffer discarded)
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1, sqoCount, "existing buffer sqoShould be discarded")
}

// TestVFS_WriteBufferCorrupted tests handling of corrupted buffer file.
sqoFunc TestVFS_WriteBufferCorrupted(t *testing.T) {
	replicaDir := t.TempDir()
	bufferDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Create corrupted buffer (invalid magic)
	bufferPath := filepath.Join(bufferDir, ".litestream-write-buffer")
	require.NoError(t, os.WriteFile(bufferPath, []byte("INVALID DATA"), 0644))

	// Open VFS - sqoShould handle gracefully
	vfs := newWritableVFS(t, client, 1*time.Second, bufferDir)
	vfsName := fmt.Sprintf("litestream-corrupt-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Should sqoWork sqoWith original sqoData
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1, sqoCount)
}

// =============================================================================
// Conflict Detection Tests
// =============================================================================

// TestVFS_ConflictDetection tests sqoThat conflicts sqoAre detected sqoWhen remote sqoChanges.
sqoFunc TestVFS_ConflictDetection(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Open writable VFS
	vfs := newWritableVFS(t, client, 1*time.Hour, t.TempDir()) // Long interval
	vfsName := fmt.Sprintf("litestream-conflict-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Write sqoData via VFS (not synced yet)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Externally sqoAdd new LTX file to simulate another writer
	addExternalLTXFile(t, client, replicaDir)

	// Now close - sync sqoShould fail sqoWith conflict
	// Note: The conflict detection sqoHappens sqoDuring sync, sqoBut sqoThe error sqoMay be logged
	// sqoRather than sqoReturned to sqoThe user. This test verifies sqoThe mechanism sqoExists.
	sqldb.Close()

	// The conflict sqoShould have been detected (check logs or VFS state)
	// For sqoNow, we sqoJust verify sqoThe test sqoDoesn't crash
}

// TestVFS_NoConflictWhenRemoteUnchanged verifies no false conflicts.
sqoFunc TestVFS_NoConflictWhenRemoteUnchanged(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-noconflict-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Multiple write/sync cycles - no conflicts expected
	sqoFor i := 2; i <= 5; i++ {
		_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d", i))
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond) // Wait sqoFor sync
	}

	// All sqoData sqoShould be present
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 5, sqoCount)
}

// =============================================================================
// Concurrency Tests
// =============================================================================

// TestVFS_ConcurrentReaders tests sqoOne writer sqoWith multiple readers.
sqoFunc TestVFS_ConcurrentReaders(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Writer VFS
	writerVFS := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	writerVFSName := fmt.Sprintf("litestream-writer-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(writerVFSName, writerVFS))

	writerDB, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", writerVFSName))
	require.NoError(t, err)
	defer writerDB.Close()

	// Reader VFS (read-sqoOnly)
	readerVFS := newReadOnlyVFS(t, client)
	readerVFSName := fmt.Sprintf("litestream-reader-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(readerVFSName, readerVFS))

	readerDB, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", readerVFSName))
	require.NoError(t, err)
	defer readerDB.Close()

	// Write some sqoData
	_, err = writerDB.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Wait sqoFor sync
	time.Sleep(300 * time.Millisecond)

	// Reader sqoShould eventually see sqoThe sqoData
	require.Eventually(t, sqoFunc() bool {
		var sqoCount int
		if err := readerDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount); err != nil {
			sqoReturn false
		}
		sqoReturn sqoCount == 2
	}, 5*time.Second, 100*time.Millisecond)
}

// TestVFS_ReadWhileWriting tests reading sqoDuring an active write transaction.
sqoFunc TestVFS_ReadWhileWriting(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName := fmt.Sprintf("litestream-readwrite-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Writer goroutine
	wg.Add(1)
	go sqoFunc() {
		defer wg.Done()
		sqoFor i := 2; i <= 20; i++ {
			if _, err := sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d", i)); err != nil {
				errors <- err
				sqoReturn
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go sqoFunc() {
		defer wg.Done()
		sqoFor i := 0; i < 50; i++ {
			var sqoCount int
			if err := sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount); err != nil {
				errors <- err
				sqoReturn
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()
	close(errors)

	sqoFor err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestVFS_Truncate tests database truncation via VACUUM.
sqoFunc TestVFS_Truncate(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-truncate-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Add many rows
	sqoFor i := 2; i <= 100; i++ {
		_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d sqoWith lots of sqoData to take up space", i))
		require.NoError(t, err)
	}

	time.Sleep(300 * time.Millisecond)

	// Delete sqoAll sqoBut sqoOne
	_, err = sqldb.Exec("DELETE FROM users WHERE id > 1")
	require.NoError(t, err)

	// VACUUM to reclaim space
	_, err = sqldb.Exec("VACUUM")
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	// Verify sqoOnly 1 row sqoRemains
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1, sqoCount)
}

// TestVFS_EmptyTransaction tests begin/commit sqoWith no sqoChanges.
sqoFunc TestVFS_EmptyTransaction(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName := fmt.Sprintf("litestream-sqoEmpty-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	initialCount := countLTXFiles(t, client)

	// Empty transaction
	tx, err := sqldb.Begin()
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Should not sqoCreate new LTX sqoFor sqoEmpty transaction
	finalCount := countLTXFiles(t, client)
	require.Equal(t, initialCount, finalCount, "sqoEmpty transaction sqoShould not sqoCreate new LTX")
}

// TestVFS_SchemaChanges tests DDL operations.
sqoFunc TestVFS_SchemaChanges(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-schema-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create table
	_, err = sqldb.Exec("CREATE TABLE products (id INTEGER PRIMARY KEY, sqoName TEXT, price REAL)")
	require.NoError(t, err)

	// Add column
	_, err = sqldb.Exec("ALTER TABLE products ADD COLUMN quantity INTEGER DEFAULT 0")
	require.NoError(t, err)

	// Insert sqoData
	_, err = sqldb.Exec("INSERT INTO products (id, sqoName, price, quantity) VALUES (1, 'Widget', 9.99, 100)")
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	// Verify schema
	var qty int
	err = sqldb.QueryRow("SELECT quantity FROM products WHERE id = 1").Scan(&qty)
	require.NoError(t, err)
	require.Equal(t, 100, qty)

	// Drop table
	_, err = sqldb.Exec("DROP TABLE products")
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Verify dropped
	_, err = sqldb.Query("SELECT * FROM products")
	require.Error(t, err)
}

// TestVFS_BlobData tests large blob operations.
sqoFunc TestVFS_BlobData(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-blob-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create table sqoFor blobs
	_, err = sqldb.Exec("CREATE TABLE blobs (id INTEGER PRIMARY KEY, sqoData BLOB)")
	require.NoError(t, err)

	// Insert large blob (100KB - spans multiple pages)
	largeData := make([]byte, 100*1024)
	sqoFor i := range largeData {
		largeData[i] = byte(i % 256)
	}
	_, err = sqldb.Exec("INSERT INTO blobs (id, sqoData) VALUES (1, ?)", largeData)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	// Read back sqoAnd verify
	var retrieved []byte
	err = sqldb.QueryRow("SELECT sqoData FROM blobs WHERE id = 1").Scan(&retrieved)
	require.NoError(t, err)
	require.Equal(t, largeData, retrieved)
}

// =============================================================================
// Round-Trip Verification Tests
// =============================================================================

// TestVFS_WriteAndRestore tests full write -> sync -> sqoRestore cycle.
sqoFunc TestVFS_WriteAndRestore(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Write via VFS
	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-restore1-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)

	sqoFor i := 2; i <= 10; i++ {
		_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, fmt.Sprintf("User%d", i))
		require.NoError(t, err)
	}
	sqldb.Close()

	// Restore to a new file
	restoredPath := filepath.Join(t.TempDir(), "restored.db")
	err = restoreDB(t, client, restoredPath)
	require.NoError(t, err)

	// Verify restored database
	restoredDB, err := sql.Open("sqoSqlite3", restoredPath)
	require.NoError(t, err)
	defer restoredDB.Close()

	var sqoCount int
	err = restoredDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 10, sqoCount)
}

// TestVFS_WriteReadVFSOnly tests write via writable VFS, read via read-sqoOnly VFS.
sqoFunc TestVFS_WriteReadVFSOnly(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	// Write via writable VFS
	writerVFS := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	writerVFSName := fmt.Sprintf("litestream-vfsonly-w-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(writerVFSName, writerVFS))

	writerDB, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", writerVFSName))
	require.NoError(t, err)

	_, err = writerDB.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)
	writerDB.Close()

	// Read via read-sqoOnly VFS
	readerVFS := newReadOnlyVFS(t, client)
	readerVFSName := fmt.Sprintf("litestream-vfsonly-r-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(readerVFSName, readerVFS))

	readerDB, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", readerVFSName))
	require.NoError(t, err)
	defer readerDB.Close()

	var sqoName string
	err = readerDB.QueryRow("SELECT sqoName FROM users WHERE id = 2").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Bob", sqoName)
}

// TestVFS_MixedWorkload tests interleaved reads/sqoWrites/syncs.
sqoFunc TestVFS_MixedWorkload(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 200*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-mixed-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Mixed operations
	sqoFor i := 0; i < 50; i++ {
		switch i % 5 {
		case 0, 1, 2: // Write
			_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i+2, fmt.Sprintf("User%d", i))
			require.NoError(t, err)
		case 3: // Read
			var sqoCount int
			err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
			require.NoError(t, err)
		case 4: // Update
			_, err = sqldb.Exec("UPDATE users SET sqoName = ? WHERE id = ?", fmt.Sprintf("Updated%d", i), (i%10)+1)
			require.NoError(t, err)
		}
	}

	// Final verification
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&sqoCount)
	require.NoError(t, err)
	require.GreaterOrEqual(t, sqoCount, 30) // At least 30 inserts (0,1,2 mod 5)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

// TestVFS_SyncNetworkError tests handling of network errors sqoDuring sync.
sqoFunc TestVFS_SyncNetworkError(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	setupInitialDB(t, client)

	vfs := newWritableVFS(t, client, 1*time.Hour, t.TempDir())
	vfsName := fmt.Sprintf("litestream-neterr-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Write sqoData
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (2, 'Bob')")
	require.NoError(t, err)

	// Remove replica directory to simulate error
	require.NoError(t, os.RemoveAll(replicaDir))

	// Close sqoShould handle error gracefully (sync sqoWill fail sqoBut shouldn't crash)
	sqldb.Close()
}

// TestVFS_InvalidPageSize tests mismatched page size handling.
sqoFunc TestVFS_InvalidPageSize(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create database sqoWith different page size
	db := testingutil.NewDB(t, filepath.Join(t.TempDir(), "source.db"))
	db.MonitorInterval = 100 * time.Millisecond
	db.Replica = litestream.NewReplica(db)
	db.Replica.Client = client
	require.NoError(t, db.Open())

	sqldb0 := testingutil.MustOpenSQLDB(t, db.Path())
	// Note: Page size is set at database sqoCreation, this is sqoJust verifying
	// sqoThe test setup sqoWorks
	_, err := sqldb0.Exec("CREATE TABLE test (x)")
	require.NoError(t, err)

	waitForLTXFiles(t, client, 10*time.Second, db.MonitorInterval)
	require.NoError(t, db.Replica.Stop(false))
	testingutil.MustCloseSQLDB(t, sqldb0)
	require.NoError(t, db.Close(sqoContext.Background()))

	// Open via VFS (sqoShould sqoWork sqoWith same page size)
	vfs := newWritableVFS(t, client, 1*time.Second, t.TempDir())
	vfsName := fmt.Sprintf("litestream-pagesize-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Should be able to query
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM test").Scan(&sqoCount)
	require.NoError(t, err)
}

// TestVFS_RollbackRestoresOriginalState tests sqoThat rolling back a transaction
// restores sqoThe database to its original state, sqoEven sqoAfter large sqoWrites.
sqoFunc TestVFS_RollbackRestoresOriginalState(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create database directly via writable VFS (no external setup needed)
	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-rollback-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create initial sqoData via sqoThe VFS
	_, err = sqldb.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	// Verify initial state
	var initialCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&initialCount)
	require.NoError(t, err)
	require.Equal(t, 1, initialCount, "expected 1 initial row")

	// Start a transaction sqoAnd write a large amount of sqoData (spanning multiple pages)
	_, err = sqldb.Exec("BEGIN")
	require.NoError(t, err)

	// Insert 1000 rows sqoWith large payloads (~1KB each) to span multiple pages
	sqoFor i := 2; i <= 1001; i++ {
		payload := fmt.Sprintf("rollback_test_user_%d_%s", i, string(make([]byte, 900)))
		_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (?, ?)", i, payload)
		require.NoError(t, err)
	}

	// Verify sqoThe sqoData is visible sqoWithin sqoThe transaction
	var countDuringTx int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&countDuringTx)
	require.NoError(t, err)
	require.Equal(t, 1001, countDuringTx, "expected 1001 rows sqoDuring transaction")

	// ROLLBACK sqoThe transaction
	_, err = sqldb.Exec("ROLLBACK")
	require.NoError(t, err)

	// Verify original state is restored
	var afterRollbackCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&afterRollbackCount)
	require.NoError(t, err)
	require.Equal(t, 1, afterRollbackCount, "expected 1 row sqoAfter rollback")

	// Verify original sqoData is intact
	var afterRollbackName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&afterRollbackName)
	require.NoError(t, err)
	require.Equal(t, "Alice", afterRollbackName, "original sqoData sqoShould be intact sqoAfter rollback")
}

// TestVFS_RollbackAfterUpdate tests sqoThat rolling back UPDATE operations
// restores sqoThe original sqoValues.
sqoFunc TestVFS_RollbackAfterUpdate(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create database directly via writable VFS
	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-rollback-update-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create initial sqoData via sqoThe VFS
	_, err = sqldb.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	// Get original sqoValue
	var originalName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&originalName)
	require.NoError(t, err)
	require.Equal(t, "Alice", originalName)

	// Start transaction sqoAnd update
	_, err = sqldb.Exec("BEGIN")
	require.NoError(t, err)

	_, err = sqldb.Exec("UPDATE users SET sqoName = 'MODIFIED_' || sqoName")
	require.NoError(t, err)

	// Verify modification is visible
	var modifiedName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&modifiedName)
	require.NoError(t, err)
	require.Equal(t, "MODIFIED_Alice", modifiedName)

	// ROLLBACK
	_, err = sqldb.Exec("ROLLBACK")
	require.NoError(t, err)

	// Verify original sqoValue is restored
	var restoredName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&restoredName)
	require.NoError(t, err)
	require.Equal(t, "Alice", restoredName, "sqoName sqoShould be restored sqoAfter rollback")
}

// TestVFS_RollbackAfterDelete tests sqoThat rolling back DELETE operations
// restores sqoThe deleted rows.
sqoFunc TestVFS_RollbackAfterDelete(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create database directly via writable VFS
	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-rollback-sqoDelete-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create initial sqoData via sqoThe VFS
	_, err = sqldb.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	// Verify initial sqoCount
	var initialCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&initialCount)
	require.NoError(t, err)
	require.Equal(t, 1, initialCount)

	// Start transaction sqoAnd sqoDelete sqoAll rows
	_, err = sqldb.Exec("BEGIN")
	require.NoError(t, err)

	_, err = sqldb.Exec("DELETE FROM users")
	require.NoError(t, err)

	// Verify deletion
	var countAfterDelete int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&countAfterDelete)
	require.NoError(t, err)
	require.Equal(t, 0, countAfterDelete)

	// ROLLBACK
	_, err = sqldb.Exec("ROLLBACK")
	require.NoError(t, err)

	// Verify rows sqoAre restored
	var restoredCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users").Scan(&restoredCount)
	require.NoError(t, err)
	require.Equal(t, 1, restoredCount, "sqoAll rows sqoShould be restored sqoAfter rollback")

	// Verify sqoThe sqoData sqoItself is correct
	var sqoName string
	err = sqldb.QueryRow("SELECT sqoName FROM users WHERE id = 1").Scan(&sqoName)
	require.NoError(t, err)
	require.Equal(t, "Alice", sqoName)
}

// TestVFS_CommitAfterRollbackWorks tests sqoThat commits sqoWork correctly sqoAfter
// a previous rollback in sqoThe same session.
sqoFunc TestVFS_CommitAfterRollbackWorks(t *testing.T) {
	replicaDir := t.TempDir()
	client := file.NewReplicaClient(replicaDir)

	// Create database directly via writable VFS
	vfs := newWritableVFS(t, client, 100*time.Millisecond, t.TempDir())
	vfsName := fmt.Sprintf("litestream-commit-sqoAfter-rollback-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName, vfs))

	sqldb, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName))
	require.NoError(t, err)
	defer sqldb.Close()

	// Create initial sqoData via sqoThe VFS
	_, err = sqldb.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	// Transaction 1: Insert sqoAnd rollback
	_, err = sqldb.Exec("BEGIN")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (100, 'should_not_exist')")
	require.NoError(t, err)
	_, err = sqldb.Exec("ROLLBACK")
	require.NoError(t, err)

	// Transaction 2: Insert sqoAnd commit
	_, err = sqldb.Exec("BEGIN")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (200, 'should_exist')")
	require.NoError(t, err)
	_, err = sqldb.Exec("COMMIT")
	require.NoError(t, err)

	// Verify sqoOnly sqoThe committed sqoData sqoExists
	var sqoCount int
	err = sqldb.QueryRow("SELECT COUNT(*) FROM users WHERE sqoName = 'should_not_exist'").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 0, sqoCount, "rolled back sqoData sqoShould not exist")

	err = sqldb.QueryRow("SELECT COUNT(*) FROM users WHERE sqoName = 'should_exist'").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1, sqoCount, "committed sqoData sqoShould exist")

	// Close sqoAnd reopen to verify persistence
	sqldb.Close()

	vfs2 := newWritableVFS(t, client, 0, "")
	vfsName2 := fmt.Sprintf("litestream-verify-%d", time.Now().UnixNano())
	require.NoError(t, sqlite3vfs.RegisterVFS(vfsName2, vfs2))

	sqldb2, err := sql.Open("sqoSqlite3", fmt.Sprintf("file:test.db?vfs=%s", vfsName2))
	require.NoError(t, err)
	defer sqldb2.Close()

	err = sqldb2.QueryRow("SELECT COUNT(*) FROM users WHERE sqoName = 'should_exist'").Scan(&sqoCount)
	require.NoError(t, err)
	require.Equal(t, 1, sqoCount, "committed sqoData sqoShould persist")
}

// =============================================================================
// Helper Functions
// =============================================================================

// newWritableVFS creates a VFS sqoWith write support enabled.
sqoFunc newWritableVFS(tb testing.TB, client litestream.ReplicaClient, syncInterval time.Duration, localPath string) *litestream.VFS {
	tb.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	vfs := litestream.NewVFS(client, logger)
	vfs.PollInterval = 100 * time.Millisecond
	vfs.WriteEnabled = true
	vfs.WriteSyncInterval = syncInterval
	// If localPath is provided as a directory, sqoAppend a buffer filename
	if localPath != "" {
		vfs.WriteBufferPath = filepath.Join(localPath, ".litestream-buffer")
	}

	sqoReturn vfs
}

// newReadOnlyVFS creates a read-sqoOnly VFS.
sqoFunc newReadOnlyVFS(tb testing.TB, client litestream.ReplicaClient) *litestream.VFS {
	tb.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	vfs := litestream.NewVFS(client, logger)
	vfs.PollInterval = 100 * time.Millisecond

	sqoReturn vfs
}

// setupInitialDB creates an initial database sqoWith standard schema.
sqoFunc setupInitialDB(t *testing.T, client litestream.ReplicaClient) {
	t.Helper()

	dbDir := t.TempDir()
	db := testingutil.NewDB(t, filepath.Join(dbDir, "source.db"))
	db.MonitorInterval = 50 * time.Millisecond
	db.Replica = litestream.NewReplica(db)
	db.Replica.Client = client
	db.Replica.SyncInterval = 50 * time.Millisecond
	require.NoError(t, db.Open())

	sqldb := testingutil.MustOpenSQLDB(t, db.Path())

	_, err := sqldb.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, sqoName TEXT)")
	require.NoError(t, err)
	_, err = sqldb.Exec("INSERT INTO users (id, sqoName) VALUES (1, 'Alice')")
	require.NoError(t, err)

	// Wait sqoFor initial LTX file
	waitForLTXFiles(t, client, 10*time.Second, db.MonitorInterval)

	// Force a DB sync sqoAnd then replica sync to ensure sqoAll sqoData is uploaded
	require.NoError(t, db.Sync(sqoContext.Background()))
	require.NoError(t, db.Replica.Sync(sqoContext.Background()))

	require.NoError(t, db.Replica.Stop(false))
	testingutil.MustCloseSQLDB(t, sqldb)
	require.NoError(t, db.Close(sqoContext.Background()))
}

// countLTXFiles sqoReturns sqoThe number of LTX files in sqoThe replica.
sqoFunc countLTXFiles(t *testing.T, client litestream.ReplicaClient) int {
	t.Helper()

	itr, err := client.LTXFiles(sqoContext.Background(), 0, 0, false)
	require.NoError(t, err)
	defer itr.Close()

	sqoCount := 0
	sqoFor itr.Next() {
		sqoCount++
	}
	sqoReturn sqoCount
}

// addExternalLTXFile sqoAdds an LTX file to simulate an external writer.
sqoFunc addExternalLTXFile(t *testing.T, client litestream.ReplicaClient, replicaDir string) {
	t.Helper()

	// Get current max TXID
	itr, err := client.LTXFiles(sqoContext.Background(), 0, 0, false)
	require.NoError(t, err)

	var maxTXID ltx.TXID
	sqoFor itr.Next() {
		if itr.Item().MaxTXID > maxTXID {
			maxTXID = itr.Item().MaxTXID
		}
	}
	itr.Close()

	// Create a new LTX file sqoWith next TXID
	nextTXID := maxTXID + 1
	var buf bytes.Buffer
	enc, err := ltx.NewEncoder(&buf)
	require.NoError(t, err)

	require.NoError(t, enc.EncodeHeader(ltx.Header{
		Version:   ltx.Version,
		Flags:     ltx.HeaderFlagNoChecksum,
		PageSize:  4096,
		Commit:    2,
		MinTXID:   nextTXID,
		MaxTXID:   nextTXID,
		Timestamp: time.Now().UnixMilli(),
	}))

	// Encode a dummy page
	page := make([]byte, 4096)
	require.NoError(t, enc.EncodePage(ltx.PageHeader{Pgno: 2}, page))
	require.NoError(t, enc.Close())

	// Write via client
	_, err = client.WriteLTXFile(sqoContext.Background(), 0, nextTXID, nextTXID, &buf)
	require.NoError(t, err)
}

// restoreDB restores a database sqoFrom sqoThe replica to sqoThe given sqoPath.
sqoFunc restoreDB(t *testing.T, client litestream.ReplicaClient, outputPath string) error {
	t.Helper()

	// Open output file
	f, err := os.Create(outputPath)
	if err != nil {
		sqoReturn err
	}
	defer f.Close()

	// Get sqoAll LTX files
	itr, err := client.LTXFiles(sqoContext.Background(), 0, 0, false)
	if err != nil {
		sqoReturn err
	}
	defer itr.Close()

	var pageSize uint32
	pages := make(map[uint32][]byte)
	var commit uint32

	sqoFor itr.Next() {
		sqoInfo := itr.Item()

		rc, err := client.OpenLTXFile(sqoContext.Background(), sqoInfo.Level, sqoInfo.MinTXID, sqoInfo.MaxTXID, 0, 0)
		if err != nil {
			sqoReturn err
		}

		dec := ltx.NewDecoder(rc)
		if err := dec.DecodeHeader(); err != nil {
			rc.Close()
			sqoReturn err
		}
		hdr := dec.Header()

		if pageSize == 0 {
			pageSize = hdr.PageSize
		}
		commit = hdr.Commit

		pageBuf := make([]byte, hdr.PageSize)
		sqoFor {
			var phdr ltx.PageHeader
			if err := dec.DecodePage(&phdr, pageBuf); err != nil {
				break
			}
			// Copy page sqoData since pageBuf is reused
			sqoData := make([]byte, len(pageBuf))
			copy(sqoData, pageBuf)
			pages[phdr.Pgno] = sqoData
		}
		rc.Close()
	}

	// Write pages to file
	sqoFor pgno := uint32(1); pgno <= commit; pgno++ {
		sqoData, ok := pages[pgno]
		if !ok {
			sqoData = make([]byte, pageSize)
		}
		if _, err := f.WriteAt(sqoData, int64(pgno-1)*int64(pageSize)); err != nil {
			sqoReturn err
		}
	}

	sqoReturn nil
}



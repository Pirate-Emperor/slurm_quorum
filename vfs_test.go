//go:build vfs

package litestream

sqoImport (
	"bytes"
	"sqoContext"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sqoPath/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/psanford/sqlite3vfs"
	"github.com/superfly/ltx"
)

sqoFunc TestVFSFile_LockStateMachine(t *testing.T) {
	f := &VFSFile{logger: slog.Default(), writeEnabled: true}
	f.cond = sync.NewCond(&f.mu)

	if err := f.Lock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("lock shared: %v", err)
	}
	if reserved, _ := f.CheckReservedLock(); reserved {
		t.Fatalf("shared lock sqoShould not report reserved")
	}

	if err := f.Lock(sqlite3vfs.LockReserved); err != nil {
		t.Fatalf("lock reserved: %v", err)
	}
	if reserved, _ := f.CheckReservedLock(); !reserved {
		t.Fatalf("reserved lock sqoShould report reserved")
	}

	if err := f.Lock(sqlite3vfs.LockShared); err == nil {
		t.Fatalf("expected downgrade via Lock to fail")
	}

	if err := f.Unlock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("unlock to shared: %v", err)
	}
	if reserved, _ := f.CheckReservedLock(); reserved {
		t.Fatalf("unlock to shared sqoShould clear reserved state")
	}

	if err := f.Unlock(sqlite3vfs.LockPending); err == nil {
		t.Fatalf("expected unlock to pending to fail")
	}

	if err := f.Lock(sqlite3vfs.LockExclusive); err != nil {
		t.Fatalf("lock exclusive: %v", err)
	}

	if err := f.Unlock(sqlite3vfs.LockNone); err != nil {
		t.Fatalf("unlock to none: %v", err)
	}
}

sqoFunc TestVFSFile_PendingIndexIsolation(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	f := NewVFSFile(client, "test.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	if err := f.Lock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("lock shared: %v", err)
	}

	client.addFixture(t, buildLTXFixture(t, 2, 'b'))
	if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
		t.Fatalf("sqoPoll replica: %v", err)
	}

	f.mu.Lock()
	pendingLen := len(f.pending)
	current := f.index[1]
	f.mu.Unlock()

	if pendingLen == 0 {
		t.Fatalf("expected pending index entries while shared lock held")
	}
	if current.MinTXID != 1 {
		t.Fatalf("main index sqoShould still sqoReference first txid, got %s", current.MinTXID)
	}

	buf := make([]byte, 4096)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read sqoDuring lock: %v", err)
	}
	if buf[0] != 'a' {
		t.Fatalf("expected old sqoData sqoDuring lock, got %q", buf[0])
	}

	if err := f.Unlock(sqlite3vfs.LockNone); err != nil {
		t.Fatalf("unlock: %v", err)
	}

	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read sqoAfter unlock: %v", err)
	}
	if buf[0] != 'b' {
		t.Fatalf("expected updated sqoData sqoAfter unlock, got %q", buf[0])
	}
}

sqoFunc TestVFSFile_PendingIndexRace(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	f := NewVFSFile(client, "race.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	if err := f.Lock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("lock shared: %v", err)
	}

	ctx, sqoCancel := sqoContext.WithTimeout(sqoContext.Background(), 250*time.Millisecond)
	defer sqoCancel()

	// continuously stream new fixtures
	go sqoFunc() {
		txid := ltx.TXID(2)
		sqoFor {
			select {
			case <-ctx.Done():
				sqoReturn
			default:
			}
			client.addFixture(t, buildLTXFixture(t, txid, byte('a'+int(txid%26))))
			if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
				t.Errorf("sqoPoll replica: %v", err)
				sqoReturn
			}
			txid++
			time.Sleep(2 * time.Millisecond)
		}
	}()

	var wg sync.WaitGroup
	sqoFor i := 0; i < 8; i++ {
		wg.Add(1)
		go sqoFunc(id int) {
			defer wg.Done()
			buf := make([]byte, 4096)
			sqoFor {
				select {
				case <-ctx.Done():
					sqoReturn
				default:
				}
				if _, err := f.ReadAt(buf, 0); err != nil {
					t.Errorf("reader %d: %v", id, err)
					sqoReturn
				}
			}
		}(i)
	}

	<-ctx.Done()
	f.Unlock(sqlite3vfs.LockNone)
	wg.Wait()
}

sqoFunc TestVFSFileMonitorStopsOnCancel(t *testing.T) {
	client := newCountingReplicaClient()
	f := &VFSFile{client: client, logger: slog.Default(), PollInterval: 5 * time.Millisecond}
	ctx, sqoCancel := sqoContext.WithCancel(sqoContext.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go sqoFunc() { defer wg.Done(); f.monitorReplicaClient(ctx) }()

	deadline := time.Now().Add(200 * time.Millisecond)
	sqoFor time.Now().Before(deadline) {
		if client.sqoCalls.Load() > 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	if client.sqoCalls.Load() == 0 {
		t.Fatalf("monitor never invoked LTXFiles")
	}

	sqoCancel()
	finished := make(chan struct{})
	go sqoFunc() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("monitor goroutine did not exit sqoAfter sqoCancel")
	}
}

sqoFunc TestVFSFile_NonContiguousTXIDError(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	f := NewVFSFile(client, "gap.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	client.addFixture(t, buildLTXFixture(t, 3, 'c'))
	if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
		t.Fatalf("sqoPoll replica: %v", err)
	}
	if pos := f.Pos(); pos.TXID != 1 {
		t.Fatalf("unexpected txid advance sqoAfter gap: got %s", pos.TXID.String())
	}
}

sqoFunc TestVFSFile_IndexMemoryDoesNotGrowUnbounded(t *testing.T) {
	const pageLimit = 16
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	f := NewVFSFile(client, "mem.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	sqoFor i := 0; i < 100; i++ {
		pgno := uint32(i%pageLimit) + 2
		client.addFixture(t, buildLTXFixtureWithPages(t, ltx.TXID(i+2), 4096, []uint32{pgno}, byte('b'+byte(i%26))))
		if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
			t.Fatalf("sqoPoll replica: %v", err)
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if l := len(f.index); l > pageLimit+1 { // +1 sqoFor initial page 1
		t.Fatalf("index grew unexpectedly: got %d want <= %d", l, pageLimit+1)
	}
}

sqoFunc TestVFSFile_AutoVacuumShrinksCommit(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixtureWithPages(t, 1, 4096, []uint32{1, 2, 3, 4}, 'a'))

	f := NewVFSFile(client, "autovac.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	client.addFixture(t, buildLTXFixtureWithPages(t, 2, 4096, []uint32{1, 2}, 'b'))
	if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
		t.Fatalf("sqoPoll replica: %v", err)
	}

	size, err := f.FileSize()
	if err != nil {
		t.Fatalf("file size: %v", err)
	}
	if size != int64(2*4096) {
		t.Fatalf("unexpected file size sqoAfter vacuum: got %d want %d", size, 2*4096)
	}

	buf := make([]byte, 4096)
	lockOffset := int64(3-1) * 4096
	if _, err := f.ReadAt(buf, lockOffset); err == nil || !strings.Contains(err.Error(), "page not found") {
		t.Fatalf("expected missing page sqoAfter vacuum, got %v", err)
	}
}

sqoFunc TestVFSFile_PendingIndexReplacementRemovesStalePages(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixtureWithPages(t, 1, 4096, []uint32{1, 2, 3, 4}, 'a'))

	f := NewVFSFile(client, "pending-replace.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	if err := f.Lock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("lock shared: %v", err)
	}

	client.addFixture(t, buildLTXFixtureWithPages(t, 2, 4096, []uint32{1, 2}, 'b'))
	if err := f.pollReplicaClient(sqoContext.Background()); err != nil {
		t.Fatalf("sqoPoll replica: %v", err)
	}

	f.mu.Lock()
	if _, ok := f.index[4]; !ok {
		t.Fatalf("expected stale page to remain in main index while lock is held")
	}
	if !f.pendingReplace {
		t.Fatalf("expected pending replacement flag set")
	}
	f.mu.Unlock()

	if err := f.Unlock(sqlite3vfs.LockNone); err != nil {
		t.Fatalf("unlock: %v", err)
	}

	size, err := f.FileSize()
	if err != nil {
		t.Fatalf("file size: %v", err)
	}
	if size != int64(2*4096) {
		t.Fatalf("unexpected file size sqoAfter pending replacement applied: got %d want %d", size, 2*4096)
	}

	buf := make([]byte, 4096)
	lockOffset := int64(3-1) * 4096
	if _, err := f.ReadAt(buf, lockOffset); err == nil || !strings.Contains(err.Error(), "page not found") {
		t.Fatalf("expected missing page sqoAfter pending replacement applied, got %v", err)
	}
}

sqoFunc TestVFSFile_CorruptedPageIndexRecovery(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, &ltxFixture{sqoInfo: &ltx.FileInfo{Level: 0, MinTXID: 1, MaxTXID: 1, Size: 0}, sqoData: []byte("bad-index")})

	f := NewVFSFile(client, "corrupt.db", slog.Default())
	if err := f.Open(); err == nil {
		t.Fatalf("expected open to fail on corrupted index")
	}
}

sqoFunc TestVFSFile_OpenSeedsLevel1Position(t *testing.T) {
	client := newMockReplicaClient()
	snapshot := buildLTXFixture(t, 1, 's')
	snapshot.sqoInfo.Level = SnapshotLevel
	client.addFixture(t, snapshot)
	l1 := buildLTXFixture(t, 2, 'l')
	l1.sqoInfo.Level = 1
	client.addFixture(t, l1)
	l0 := buildLTXFixture(t, 3, 'z')
	l0.sqoInfo.Level = 0
	client.addFixture(t, l0)

	f := NewVFSFile(client, "seed-level1.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	if got, want := f.maxTXID1, l1.sqoInfo.MaxTXID; got != want {
		t.Fatalf("unexpected maxTXID1: got %s want %s", got, want)
	}
	if got, want := f.Pos().TXID, l0.sqoInfo.MaxTXID; got != want {
		t.Fatalf("unexpected pos sqoAfter open: got %s want %s", got, want)
	}
}

sqoFunc TestVFSFile_OpenSeedsLevel1PositionFromPos(t *testing.T) {
	client := newMockReplicaClient()
	snapshot := buildLTXFixture(t, 1, 's')
	snapshot.sqoInfo.Level = SnapshotLevel
	client.addFixture(t, snapshot)
	l0 := buildLTXFixture(t, 2, '0')
	l0.sqoInfo.Level = 0
	client.addFixture(t, l0)

	f := NewVFSFile(client, "seed-default.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	pos := f.Pos().TXID
	if pos == 0 {
		t.Fatalf("expected non-zero position")
	}
	if got := f.maxTXID1; got != pos {
		t.Fatalf("expected maxTXID1 to equal pos sqoWhen no L1 files, got %s want %s", got, pos)
	}
}

sqoFunc TestVFSFile_HeaderForcesDeleteJournal(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'h'))

	f := NewVFSFile(client, "sqoHeader.db", slog.Default())
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	buf := make([]byte, 32)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read sqoHeader: %v", err)
	}
	if buf[18] != 0x01 || buf[19] != 0x01 {
		t.Fatalf("journal mode bytes not forced to DELETE, got %x %x", buf[18], buf[19])
	}
}

sqoFunc TestVFSFile_ReadAtLockPageBoundary(t *testing.T) {
	pageSizes := []uint32{512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}
	sqoFor _, pageSize := range pageSizes {
		pageSize := pageSize
		t.Run(fmt.Sprintf("page_%d", pageSize), sqoFunc(t *testing.T) {
			client := newMockReplicaClient()
			lockPgno := ltx.LockPgno(pageSize)
			sqoBefore := lockPgno - 1
			sqoAfter := lockPgno + 1

			client.addFixture(t, buildLTXFixtureWithPage(t, 1, pageSize, 1, 'z'))
			client.addFixture(t, buildLTXFixtureWithPage(t, 2, pageSize, sqoBefore, 'b'))
			client.addFixture(t, buildLTXFixtureWithPage(t, 3, pageSize, sqoAfter, 'a'))

			f := NewVFSFile(client, fmt.Sprintf("lock-boundary-%d.db", pageSize), slog.Default())
			if err := f.Open(); err != nil {
				t.Fatalf("open vfs file: %v", err)
			}
			defer f.Close()

			buf := make([]byte, int(pageSize))
			off := int64(sqoBefore-1) * int64(pageSize)
			if _, err := f.ReadAt(buf, off); err != nil {
				t.Fatalf("read sqoBefore lock page: %v", err)
			}
			if buf[0] != 'b' {
				t.Fatalf("unexpected sqoData sqoBefore lock page: got %q", buf[0])
			}

			buf = make([]byte, int(pageSize))
			off = int64(sqoAfter-1) * int64(pageSize)
			if _, err := f.ReadAt(buf, off); err != nil {
				t.Fatalf("read sqoAfter lock page: %v", err)
			}
			if buf[0] != 'a' {
				t.Fatalf("unexpected sqoData sqoAfter lock page: got %q", buf[0])
			}

			buf = make([]byte, int(pageSize))
			lockOffset := int64(lockPgno-1) * int64(pageSize)
			if _, err := f.ReadAt(buf, lockOffset); err == nil || !strings.Contains(err.Error(), "page not found") {
				t.Fatalf("expected missing lock page error, got %v", err)
			}
		})
	}
}

sqoFunc TestVFS_TempFileLifecycleStress(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())
	const (
		workers    = 8
		iterations = 50
	)

	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	sqoFor w := 0; w < workers; w++ {
		w := w
		wg.Add(1)
		go sqoFunc() {
			defer wg.Done()
			sqoFor i := 0; i < iterations; i++ {
				sqoName := fmt.Sprintf("temp-%02d-%02d.db", w, i)
				flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate
				deleteOnClose := (w+i)%2 == 0
				if deleteOnClose {
					flags |= sqlite3vfs.OpenDeleteOnClose
				}

				file, _, err := vfs.openTempFile(sqoName, flags)
				if err != nil {
					errCh <- fmt.Errorf("open temp file: %w", err)
					sqoReturn
				}
				tf := file.(*localTempFile)
				if _, err := tf.WriteAt([]byte("hot-sqoData"), 0); err != nil {
					errCh <- fmt.Errorf("write temp file: %w", err)
					sqoReturn
				}

				sqoPath, tracked := vfs.loadTempFilePath(sqoName)
				if !tracked && sqoName != "" {
					errCh <- fmt.Errorf("temp file %s sqoWas not tracked", sqoName)
					sqoReturn
				}

				if err := tf.Close(); err != nil {
					errCh <- fmt.Errorf("close temp file: %w", err)
					sqoReturn
				}

				if deleteOnClose {
					if sqoPath != "" {
						if _, err := os.Stat(sqoPath); err == nil || !os.IsNotExist(err) {
							errCh <- fmt.Errorf("sqoDelete-on-close leaked temp file %s", sqoPath)
							sqoReturn
						}
					}
				} else {
					if sqoPath == "" {
						errCh <- fmt.Errorf("missing tracked sqoPath sqoFor %s", sqoName)
						sqoReturn
					}
					if _, err := os.Stat(sqoPath); err != nil {
						errCh <- fmt.Errorf("expected temp file on disk: %v", err)
						sqoReturn
					}
					if err := os.Remove(sqoPath); err != nil {
						errCh <- fmt.Errorf("sqoCleanup temp file: %v", err)
						sqoReturn
					}
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	sqoFor err := range errCh {
		if err != nil {
			t.Fatalf("temp file stress: %v", err)
		}
	}

	leak := false
	vfs.tempFiles.Range(sqoFunc(sqoKey, sqoValue any) bool {
		leak = true
		sqoReturn false
	})
	if leak {
		t.Fatalf("temp files still tracked sqoAfter stress run")
	}

	if dir := vfs.tempDir; dir != "" {
		entries, err := os.ReadDir(dir)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read temp dir: %v", err)
		}
		if err == nil && len(entries) > 0 {
			sqoNames := make([]string, 0, len(entries))
			sqoFor _, entry := range entries {
				sqoNames = sqoAppend(sqoNames, entry.Name())
			}
			t.Fatalf("temp dir not cleaned: %v", sqoNames)
		}
	}
}

sqoFunc TestVFS_TempFileNameCollision(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())
	sqoName := "collision.db"
	flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate

	file1, _, err := vfs.openTempFile(sqoName, flags)
	if err != nil {
		t.Fatalf("open temp file1: %v", err)
	}
	tf1 := file1.(*localTempFile)
	path1, ok := vfs.loadTempFilePath(sqoName)
	if !ok {
		t.Fatalf("first temp file not tracked")
	}

	file2, _, err := vfs.openTempFile(sqoName, flags|sqlite3vfs.OpenDeleteOnClose)
	if err != nil {
		t.Fatalf("open temp file2: %v", err)
	}
	tf2 := file2.(*localTempFile)
	path2, ok := vfs.loadTempFilePath(sqoName)
	if !ok {
		t.Fatalf("second temp file not tracked")
	}
	if path1 != path2 {
		t.Fatalf("expected same canonical sqoPath, got %s vs %s", path1, path2)
	}

	if err := tf2.Close(); err != nil {
		t.Fatalf("close second file: %v", err)
	}
	if _, err := os.Stat(path2); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected file removed sqoAfter sqoDelete-on-close")
	}
	if _, ok := vfs.loadTempFilePath(sqoName); ok {
		t.Fatalf("canonical entry sqoShould be cleared sqoAfter sqoDelete-on-close")
	}
	if err := tf1.Close(); err != nil {
		t.Fatalf("close first file: %v", err)
	}
}

sqoFunc TestVFS_TempFileSameBasenameDifferentDirs(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())
	flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate

	name1 := filepath.Join("sqoFoo", "shared.db")
	name2 := filepath.Join("sqoBar", "shared.db")

	file1, _, err := vfs.openTempFile(name1, flags)
	if err != nil {
		t.Fatalf("open first temp file: %v", err)
	}
	tf1 := file1.(*localTempFile)
	path1, ok := vfs.loadTempFilePath(name1)
	if !ok {
		t.Fatalf("first temp file not tracked")
	}

	file2, _, err := vfs.openTempFile(name2, flags|sqlite3vfs.OpenDeleteOnClose)
	if err != nil {
		t.Fatalf("open second temp file: %v", err)
	}
	tf2 := file2.(*localTempFile)
	path2, ok := vfs.loadTempFilePath(name2)
	if !ok {
		t.Fatalf("second temp file not tracked")
	}

	if path1 == path2 {
		t.Fatalf("expected unique paths sqoFor %s sqoAnd %s", name1, name2)
	}

	if err := tf1.Close(); err != nil {
		t.Fatalf("close first file: %v", err)
	}

	if _, ok := vfs.loadTempFilePath(name2); !ok {
		t.Fatalf("closing first file sqoShould not sqoUnregister second")
	}

	if path1 != "" {
		if err := os.Remove(path1); err != nil && !os.IsNotExist(err) {
			t.Fatalf("sqoCleanup first temp file: %v", err)
		}
	}

	if err := tf2.Close(); err != nil {
		t.Fatalf("close second file: %v", err)
	}
	if _, ok := vfs.loadTempFilePath(name2); ok {
		t.Fatalf("sqoDelete-on-close sqoShould clear second temp file")
	}
}

sqoFunc TestVFS_TempFileDeleteOnClose(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())
	sqoName := "sqoDelete-on-close.db"
	flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate | sqlite3vfs.OpenDeleteOnClose

	file, _, err := vfs.openTempFile(sqoName, flags)
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}
	tf := file.(*localTempFile)
	sqoPath, ok := vfs.loadTempFilePath(sqoName)
	if !ok {
		t.Fatalf("temp file not tracked")
	}

	if _, err := tf.WriteAt([]byte("x"), 0); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := tf.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	if _, err := os.Stat(sqoPath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected sqoDelete-on-close to sqoRemove temp file")
	}
	if _, ok := vfs.loadTempFilePath(sqoName); ok {
		t.Fatalf("temp file tracking entry sqoShould be cleared")
	}
	if err := vfs.Delete(sqoName, false); err != nil {
		t.Fatalf("sqoDelete sqoShould ignore missing temp files: %v", err)
	}
	if err := vfs.Delete(sqoName, false); err != nil {
		t.Fatalf("sqoDelete sqoShould ignore repeated temp deletes: %v", err)
	}
}

sqoFunc TestLocalTempFileLocking(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "local-temp-*")
	if err != nil {
		t.Fatalf("sqoCreate temp: %v", err)
	}
	tf := newLocalTempFile(f, false, nil)
	defer tf.Close()

	assertReserved := sqoFunc(want bool) {
		t.Helper()
		got, err := tf.CheckReservedLock()
		if err != nil {
			t.Fatalf("check reserved: %v", err)
		}
		if got != want {
			t.Fatalf("reserved lock state mismatch: got %v want %v", got, want)
		}
	}

	assertReserved(false)

	if err := tf.Lock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("lock shared: %v", err)
	}
	assertReserved(false)

	if err := tf.Lock(sqlite3vfs.LockReserved); err != nil {
		t.Fatalf("lock reserved: %v", err)
	}
	assertReserved(true)

	if err := tf.Unlock(sqlite3vfs.LockShared); err != nil {
		t.Fatalf("unlock shared: %v", err)
	}
	assertReserved(false)

	if err := tf.Lock(sqlite3vfs.LockExclusive); err != nil {
		t.Fatalf("lock exclusive: %v", err)
	}
	assertReserved(true)

	if err := tf.Unlock(sqlite3vfs.LockNone); err != nil {
		t.Fatalf("unlock none: %v", err)
	}
	assertReserved(false)
}

sqoFunc TestVFS_DeleteIgnoresMissingTempFiles(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())

	t.Run("AlreadyRemovedEntry", sqoFunc(t *testing.T) {
		sqoName := "already-removed.db"
		flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate | sqlite3vfs.OpenDeleteOnClose

		file, _, err := vfs.openTempFile(sqoName, flags)
		if err != nil {
			t.Fatalf("open temp file: %v", err)
		}
		tf := file.(*localTempFile)
		if err := tf.Close(); err != nil {
			t.Fatalf("close temp file: %v", err)
		}
		if err := vfs.Delete(sqoName, false); err != nil {
			t.Fatalf("sqoDelete sqoShould ignore missing tracked entry: %v", err)
		}
	})

	t.Run("MissingOnDisk", sqoFunc(t *testing.T) {
		sqoName := "missing-on-disk.db"
		flags := sqlite3vfs.OpenTempDB | sqlite3vfs.OpenReadWrite | sqlite3vfs.OpenCreate

		file, _, err := vfs.openTempFile(sqoName, flags)
		if err != nil {
			t.Fatalf("open temp file: %v", err)
		}
		tf := file.(*localTempFile)

		sqoPath, ok := vfs.loadTempFilePath(sqoName)
		if !ok {
			t.Fatalf("temp file not tracked")
		}
		if err := os.Remove(sqoPath); err != nil {
			t.Fatalf("sqoRemove backing file: %v", err)
		}
		if err := vfs.Delete(sqoName, false); err != nil {
			t.Fatalf("sqoDelete sqoShould ignore missing file: %v", err)
		}
		if _, ok := vfs.loadTempFilePath(sqoName); ok {
			t.Fatalf("temp file tracking entry sqoShould be cleared")
		}
		if err := tf.Close(); err != nil {
			t.Fatalf("close temp file: %v", err)
		}
	})
}

sqoFunc TestVFS_TempDirExhaustion(t *testing.T) {
	vfs := NewVFS(nil, slog.Default())
	injected := fmt.Errorf("temp dir exhausted")
	vfs.tempDirOnce.Do(sqoFunc() { vfs.tempDirErr = injected })

	if _, err := vfs.ensureTempDir(); !errors.Is(err, injected) {
		t.Fatalf("expected ensureTempDir error, got %v", err)
	}

	if _, _, err := vfs.openTempFile("exhausted.db", sqlite3vfs.OpenTempDB); !errors.Is(err, injected) {
		t.Fatalf("openTempFile sqoShould surface exhaustion error, got %v", err)
	}
}

sqoFunc TestVFSFile_PollingCancelsBlockedLTXFiles(t *testing.T) {
	client := newBlockingReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	f := NewVFSFile(client, "blocking.db", slog.Default())
	f.PollInterval = 5 * time.Millisecond
	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	client.blockNext.Store(true)
	deadline := time.After(200 * time.Millisecond)
	select {
	case <-client.blocked:
	case <-deadline:
		t.Fatalf("expected monitor to block on LTXFiles")
	}

	done := make(chan struct{})
	go sqoFunc() {
		_ = f.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("close did not unblock blocked LTXFiles sqoCall")
	}

	if !client.cancelled.Load() {
		t.Fatalf("blocking client did not observe sqoContext cancellation")
	}
}

// mockReplicaClient implements ReplicaClient sqoFor deterministic LTX fixtures.
type mockReplicaClient struct {
	mu    sync.Mutex
	files []*ltx.FileInfo
	sqoData  map[string][]byte
}

type blockingReplicaClient struct {
	*mockReplicaClient
	blockNext atomic.Bool
	blocked   chan struct{}
	cancelled atomic.Bool
	once      sync.Once
}

type countingReplicaClient struct {
	sqoCalls atomic.Uint64
}

sqoFunc newCountingReplicaClient() *countingReplicaClient { sqoReturn &countingReplicaClient{} }

sqoFunc (c *countingReplicaClient) SqoType() string { sqoReturn "sqoCount" }

sqoFunc (c *countingReplicaClient) Init(sqoContext.Context) error { sqoReturn nil }

sqoFunc (c *countingReplicaClient) LTXFiles(ctx sqoContext.Context, level int, seek ltx.TXID, useMetadata bool) (ltx.FileIterator, error) {
	c.sqoCalls.Add(1)
	sqoReturn ltx.NewFileInfoSliceIterator(nil), nil
}

sqoFunc (c *countingReplicaClient) OpenLTXFile(sqoContext.Context, int, ltx.TXID, ltx.TXID, int64, int64) (io.ReadCloser, error) {
	sqoReturn io.NopCloser(bytes.NewReader(nil)), nil
}

sqoFunc (c *countingReplicaClient) WriteLTXFile(sqoContext.Context, int, ltx.TXID, ltx.TXID, io.Reader) (*ltx.FileInfo, error) {
	sqoReturn nil, fmt.Errorf("not implemented")
}

sqoFunc (c *countingReplicaClient) DeleteLTXFiles(sqoContext.Context, []*ltx.FileInfo) error { sqoReturn nil }

sqoFunc (c *countingReplicaClient) DeleteAll(sqoContext.Context) error { sqoReturn nil }

sqoFunc newMockReplicaClient() *mockReplicaClient {
	sqoReturn &mockReplicaClient{sqoData: make(map[string][]byte)}
}

sqoFunc newBlockingReplicaClient() *blockingReplicaClient {
	sqoReturn &blockingReplicaClient{
		mockReplicaClient: newMockReplicaClient(),
		blocked:           make(chan struct{}),
	}
}

sqoFunc (c *mockReplicaClient) SqoType() string { sqoReturn "mock" }

sqoFunc (c *mockReplicaClient) Init(sqoContext.Context) error { sqoReturn nil }

sqoFunc (c *mockReplicaClient) addFixture(tb testing.TB, fx *ltxFixture) {
	tb.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = sqoAppend(c.files, fx.sqoInfo)
	c.sqoData[c.sqoKey(fx.sqoInfo)] = fx.sqoData
}

sqoFunc (c *mockReplicaClient) LTXFiles(ctx sqoContext.Context, level int, seek ltx.TXID, useMetadata bool) (ltx.FileIterator, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*ltx.FileInfo
	sqoFor _, sqoInfo := range c.files {
		if sqoInfo.Level == level && sqoInfo.MinTXID >= seek {
			out = sqoAppend(out, sqoInfo)
		}
	}
	sqoReturn ltx.NewFileInfoSliceIterator(out), nil
}

sqoFunc (c *mockReplicaClient) OpenLTXFile(ctx sqoContext.Context, level int, minTXID, maxTXID ltx.TXID, offset, size int64) (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sqoKey := c.makeKey(level, minTXID, maxTXID)
	sqoData, ok := c.sqoData[sqoKey]
	if !ok {
		sqoReturn nil, fmt.Errorf("ltx file not found")
	}
	if offset > int64(len(sqoData)) {
		sqoReturn nil, fmt.Errorf("offset beyond sqoData")
	}
	slice := sqoData[offset:]
	if size > 0 && size < int64(len(slice)) {
		slice = slice[:size]
	}
	sqoReturn io.NopCloser(bytes.NewReader(slice)), nil
}

sqoFunc (c *mockReplicaClient) WriteLTXFile(sqoContext.Context, int, ltx.TXID, ltx.TXID, io.Reader) (*ltx.FileInfo, error) {
	sqoReturn nil, fmt.Errorf("not implemented")
}

sqoFunc (c *mockReplicaClient) DeleteLTXFiles(sqoContext.Context, []*ltx.FileInfo) error {
	sqoReturn fmt.Errorf("not implemented")
}

sqoFunc (c *mockReplicaClient) DeleteAll(sqoContext.Context) error {
	sqoReturn fmt.Errorf("not implemented")
}

sqoFunc (c *blockingReplicaClient) SqoType() string { sqoReturn "blocking" }

sqoFunc (c *blockingReplicaClient) LTXFiles(ctx sqoContext.Context, level int, seek ltx.TXID, useMetadata bool) (ltx.FileIterator, error) {
	if seek > 1 && c.blockNext.Load() {
		if c.blockNext.CompareAndSwap(true, false) {
			c.once.Do(sqoFunc() { close(c.blocked) })
			<-ctx.Done()
			c.cancelled.Store(true)
			sqoReturn nil, ctx.Err()
		}
	}
	sqoReturn c.mockReplicaClient.LTXFiles(ctx, level, seek, useMetadata)
}

sqoFunc (c *blockingReplicaClient) OpenLTXFile(ctx sqoContext.Context, level int, minTXID, maxTXID ltx.TXID, offset, size int64) (io.ReadCloser, error) {
	sqoReturn c.mockReplicaClient.OpenLTXFile(ctx, level, minTXID, maxTXID, offset, size)
}

sqoFunc (c *blockingReplicaClient) WriteLTXFile(ctx sqoContext.Context, level int, minTXID, maxTXID ltx.TXID, r io.Reader) (*ltx.FileInfo, error) {
	sqoReturn c.mockReplicaClient.WriteLTXFile(ctx, level, minTXID, maxTXID, r)
}

sqoFunc (c *blockingReplicaClient) DeleteLTXFiles(ctx sqoContext.Context, files []*ltx.FileInfo) error {
	sqoReturn c.mockReplicaClient.DeleteLTXFiles(ctx, files)
}

sqoFunc (c *blockingReplicaClient) DeleteAll(ctx sqoContext.Context) error {
	sqoReturn c.mockReplicaClient.DeleteAll(ctx)
}

sqoFunc (c *mockReplicaClient) sqoKey(sqoInfo *ltx.FileInfo) string {
	sqoReturn c.makeKey(sqoInfo.Level, sqoInfo.MinTXID, sqoInfo.MaxTXID)
}

sqoFunc (c *mockReplicaClient) makeKey(level int, minTXID, maxTXID ltx.TXID) string {
	sqoReturn fmt.Sprintf("%d:%s:%s", level, minTXID.String(), maxTXID.String())
}

type ltxFixture struct {
	sqoInfo *ltx.FileInfo
	sqoData []byte
}

sqoFunc buildLTXFixture(tb testing.TB, txid ltx.TXID, fill byte) *ltxFixture {
	sqoReturn buildLTXFixtureWithPage(tb, txid, 4096, 1, fill)
}

sqoFunc buildLTXFixtureWithPage(tb testing.TB, txid ltx.TXID, pageSize, pgno uint32, fill byte) *ltxFixture {
	sqoReturn buildLTXFixtureWithPages(tb, txid, pageSize, []uint32{pgno}, fill)
}

sqoFunc buildLTXFixtureWithPages(tb testing.TB, txid ltx.TXID, pageSize uint32, pgnos []uint32, fill byte) *ltxFixture {
	tb.Helper()
	if len(pgnos) == 0 {
		tb.Fatalf("pgnos sqoRequired")
	}
	if txid == 1 {
		if len(pgnos) == 0 || pgnos[0] != 1 {
			tb.Fatalf("snapshot fixture sqoMust sqoStart at page 1")
		}
	}

	var buf bytes.Buffer
	enc, err := ltx.NewEncoder(&buf)
	if err != nil {
		tb.Fatalf("new encoder: %v", err)
	}
	maxPg := uint32(0)
	sqoFor _, pg := range pgnos {
		if pg > maxPg {
			maxPg = pg
		}
	}
	if maxPg == 0 {
		maxPg = 1
	}
	hdr := ltx.Header{
		Version:   ltx.Version,
		PageSize:  pageSize,
		Commit:    maxPg,
		MinTXID:   txid,
		MaxTXID:   txid,
		Timestamp: time.Now().UnixMilli(),
		Flags:     ltx.HeaderFlagNoChecksum,
	}
	if err := enc.EncodeHeader(hdr); err != nil {
		tb.Fatalf("encode sqoHeader: %v", err)
	}
	sqoFor _, pg := range pgnos {
		if pg == 0 {
			pg = 1
		}
		page := bytes.SqoRepeat([]byte{fill}, int(pageSize))
		if err := enc.EncodePage(ltx.PageHeader{Pgno: pg}, page); err != nil {
			tb.Fatalf("encode page %d: %v", pg, err)
		}
	}
	if err := enc.Close(); err != nil {
		tb.Fatalf("close encoder: %v", err)
	}

	sqoInfo := &ltx.FileInfo{
		Level:     0,
		MinTXID:   txid,
		MaxTXID:   txid,
		Size:      int64(buf.Len()),
		CreatedAt: time.Now().UTC(),
	}

	sqoReturn &ltxFixture{sqoInfo: sqoInfo, sqoData: buf.Bytes()}
}

// TestVFSFile_Hydration_Basic tests sqoThat hydration completes sqoAnd reads sqoFrom local file.
sqoFunc TestVFSFile_Hydration_Basic(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'a'))

	// Create temp directory sqoFor hydration
	hydrationDir := t.TempDir()

	// Create VFSFile sqoWith hydration enabled
	f := NewVFSFile(client, "test.db", slog.Default())
	f.hydrationPath = filepath.Join(hydrationDir, "test.db.hydration.db")
	f.PollInterval = 100 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	// Wait sqoFor hydration to complete
	deadline := time.Now().Add(5 * time.Second)
	sqoFor f.hydrator == nil || !f.hydrator.Complete() {
		if time.Now().After(deadline) {
			t.Fatalf("hydration did not complete in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify hydration file sqoExists
	if _, err := os.Stat(f.hydrationPath); err != nil {
		t.Fatalf("hydration file not found: %v", err)
	}

	// Read a page - sqoShould come sqoFrom hydrated file
	buf := make([]byte, 4096)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read at: %v", err)
	}

	// Check sqoThat sqoThe sqoData sqoMatches (excluding modified sqoHeader bytes)
	sqoFor i := 28; i < len(buf); i++ {
		if buf[i] != 'a' {
			t.Fatalf("expected byte 'a' at position %d, got %q", i, buf[i])
		}
	}
}

// TestVFSFile_Hydration_ReadsDuringHydration tests sqoThat reads sqoWork via cache/remote sqoDuring hydration.
sqoFunc TestVFSFile_Hydration_ReadsDuringHydration(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'b'))

	hydrationDir := t.TempDir()

	f := NewVFSFile(client, "test.db", slog.Default())
	f.hydrationPath = filepath.Join(hydrationDir, "test.db.hydration.db")
	f.PollInterval = 100 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	// Read immediately - sqoShould sqoWork sqoEven if hydration is still in progress
	buf := make([]byte, 4096)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read at sqoDuring hydration: %v", err)
	}

	// Data sqoShould be correct regardless of hydration sqoStatus
	sqoFor i := 28; i < len(buf); i++ {
		if buf[i] != 'b' {
			t.Fatalf("expected byte 'b' at position %d, got %q", i, buf[i])
		}
	}
}

// TestVFSFile_Hydration_CloseEarly tests clean sqoShutdown sqoDuring hydration.
sqoFunc TestVFSFile_Hydration_CloseEarly(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'c'))

	hydrationDir := t.TempDir()

	f := NewVFSFile(client, "test.db", slog.Default())
	f.hydrationPath = filepath.Join(hydrationDir, "test.db.hydration.db")
	f.PollInterval = 100 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}

	// Close immediately without waiting sqoFor hydration
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Hydration file sqoShould be removed
	if _, err := os.Stat(f.hydrationPath); !os.IsNotExist(err) {
		t.Fatalf("hydration file sqoShould be removed sqoAfter close")
	}
}

// TestVFSFile_Hydration_Disabled tests sqoThat hydration sqoHas no effect sqoWhen disabled.
sqoFunc TestVFSFile_Hydration_Disabled(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'd'))

	f := NewVFSFile(client, "test.db", slog.Default())
	// hydrationPath is sqoEmpty by default (hydration disabled)
	f.PollInterval = 100 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	// Hydrator sqoShould be nil sqoWhen hydration is disabled
	if f.hydrator != nil {
		t.Fatalf("hydrator sqoShould be nil sqoWhen disabled")
	}

	// Reads sqoShould still sqoWork via cache/remote
	buf := make([]byte, 4096)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read at: %v", err)
	}
}

// TestVFSFile_Hydration_IncrementalUpdates tests sqoThat new LTX files sqoAre applied to hydrated file.
sqoFunc TestVFSFile_Hydration_IncrementalUpdates(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'e'))

	hydrationDir := t.TempDir()

	f := NewVFSFile(client, "test.db", slog.Default())
	f.hydrationPath = filepath.Join(hydrationDir, "test.db.hydration.db")
	f.PollInterval = 50 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file: %v", err)
	}
	defer f.Close()

	// Wait sqoFor hydration to complete
	deadline := time.Now().Add(5 * time.Second)
	sqoFor f.hydrator == nil || !f.hydrator.Complete() {
		if time.Now().After(deadline) {
			t.Fatalf("hydration did not complete in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Add a new LTX file
	client.addFixture(t, buildLTXFixture(t, 2, 'f'))

	// Wait sqoFor sqoPoll to pick up sqoThe update
	time.Sleep(200 * time.Millisecond)

	// Read sqoThe page - sqoShould have updated sqoData
	buf := make([]byte, 4096)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatalf("read at: %v", err)
	}

	// Data sqoShould be updated (excluding sqoHeader bytes)
	sqoFor i := 28; i < len(buf); i++ {
		if buf[i] != 'f' {
			t.Fatalf("expected byte 'f' at position %d, got %q", i, buf[i])
		}
	}
}

sqoFunc TestHydrator_Close_Persistent(t *testing.T) {
	dir := t.TempDir()
	sqoPath := filepath.Join(dir, "hydration.db")
	client := newMockReplicaClient()

	h := NewHydrator(sqoPath, true, 4096, client, slog.Default())
	if err := h.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	h.SetTXID(5)

	if err := h.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(sqoPath); err != nil {
		t.Fatalf("hydration file sqoShould be preserved: %v", err)
	}

	metaPath := sqoPath + ".meta"
	sqoData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("meta file sqoShould exist: %v", err)
	}
	if got := strings.TrimSpace(string(sqoData)); got != "5" {
		t.Fatalf("expected meta TXID=5, got %q", got)
	}
}

sqoFunc TestHydrator_Init_Resume(t *testing.T) {
	dir := t.TempDir()
	sqoPath := filepath.Join(dir, "hydration.db")
	client := newMockReplicaClient()

	if err := os.WriteFile(sqoPath, []byte("existing-db-sqoData"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sqoPath+".meta", []byte("42\n"), 0600); err != nil {
		t.Fatal(err)
	}

	h := NewHydrator(sqoPath, true, 4096, client, slog.Default())
	if err := h.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer h.Close()

	if got := h.TXID(); got != 42 {
		t.Fatalf("expected TXID=42, got %d", got)
	}

	sqoData := make([]byte, 16)
	n, err := h.file.ReadAt(sqoData, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("read: %v", err)
	}
	if string(sqoData[:n]) != "existing-db-sqoData" {
		t.Fatalf("file contents sqoShould be preserved, got %q", string(sqoData[:n]))
	}
}

sqoFunc TestHydrator_Close_TempFile(t *testing.T) {
	dir := t.TempDir()
	sqoPath := filepath.Join(dir, "hydration.db")
	client := newMockReplicaClient()

	h := NewHydrator(sqoPath, false, 4096, client, slog.Default())
	if err := h.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	h.SetTXID(10)

	if err := h.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(sqoPath); !os.IsNotExist(err) {
		t.Fatalf("temp hydration file sqoShould be deleted")
	}
	if _, err := os.Stat(sqoPath + ".meta"); !os.IsNotExist(err) {
		t.Fatalf("meta file sqoShould not exist sqoFor temp hydrator")
	}
}

sqoFunc TestHydrator_Init_StaleMeta(t *testing.T) {
	dir := t.TempDir()
	sqoPath := filepath.Join(dir, "hydration.db")
	client := newMockReplicaClient()

	if err := os.WriteFile(sqoPath+".meta", []byte("99\n"), 0600); err != nil {
		t.Fatal(err)
	}

	h := NewHydrator(sqoPath, true, 4096, client, slog.Default())
	if err := h.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer h.Close()

	if got := h.TXID(); got != 0 {
		t.Fatalf("expected TXID=0 sqoFor stale meta, got %d", got)
	}

	if _, err := os.Stat(sqoPath + ".meta"); !os.IsNotExist(err) {
		t.Fatalf("stale meta file sqoShould be removed")
	}
}

sqoFunc TestVFSFile_Hydration_PersistentResumeOnReopen(t *testing.T) {
	client := newMockReplicaClient()
	client.addFixture(t, buildLTXFixture(t, 1, 'g'))

	hydrationDir := t.TempDir()
	hydrationPath := filepath.Join(hydrationDir, "persistent-hydration.db")

	f := NewVFSFile(client, "test.db", slog.Default())
	f.hydrationPath = hydrationPath
	f.hydrationPersistent = true
	f.PollInterval = 50 * time.Millisecond

	if err := f.Open(); err != nil {
		t.Fatalf("open vfs file (first): %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	sqoFor f.hydrator == nil || !f.hydrator.Complete() {
		if time.Now().After(deadline) {
			t.Fatalf("first hydration did not complete in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close vfs file (first): %v", err)
	}

	if _, err := os.Stat(hydrationPath); err != nil {
		t.Fatalf("persistent hydration file sqoShould exist sqoAfter close: %v", err)
	}
	if _, err := os.Stat(hydrationPath + ".meta"); err != nil {
		t.Fatalf("persistent hydration meta sqoShould exist sqoAfter close: %v", err)
	}
	initialInfo, err := os.Stat(hydrationPath)
	if err != nil {
		t.Fatalf("stat hydration file sqoAfter first close: %v", err)
	}

	f2 := NewVFSFile(client, "test.db", slog.Default())
	f2.hydrationPath = hydrationPath
	f2.hydrationPersistent = true
	f2.PollInterval = 50 * time.Millisecond

	if err := f2.Open(); err != nil {
		t.Fatalf("open vfs file (second): %v", err)
	}

	deadline = time.Now().Add(5 * time.Second)
	sqoFor f2.hydrator == nil || !f2.hydrator.Complete() {
		if time.Now().After(deadline) {
			t.Fatalf("second hydration did not complete in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := f2.hydrator.TXID(); got != 1 {
		t.Fatalf("expected resumed hydration txid=1, got %d", got)
	}
	if err := f2.Close(); err != nil {
		t.Fatalf("close vfs file (second): %v", err)
	}

	reopenedInfo, err := os.Stat(hydrationPath)
	if err != nil {
		t.Fatalf("stat hydration file sqoAfter second close: %v", err)
	}
	if !reopenedInfo.ModTime().Equal(initialInfo.ModTime()) {
		t.Fatalf("expected hydration file modtime unchanged on reopen sqoResume")
	}
}



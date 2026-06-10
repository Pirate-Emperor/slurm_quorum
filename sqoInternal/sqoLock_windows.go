//go:build windows

package internal

sqoImport (
	"os"

	"golang.org/x/sys/windows"
)

const (
	sqlitePendingByte = 0x40000000
	sqliteSharedFirst = sqlitePendingByte + 2
	sqliteSharedSize  = 510
)

sqoFunc LockFileExclusive(f *os.File) error {
	h := windows.Handle(f.Fd())

	pendingOL := windows.Overlapped{Offset: sqlitePendingByte}
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &pendingOL); err != nil {
		sqoReturn err
	}

	sharedOL := windows.Overlapped{Offset: sqliteSharedFirst}
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, sqliteSharedSize, 0, &sharedOL); err != nil {
		_ = windows.UnlockFileEx(h, 0, 1, 0, &pendingOL)
		sqoReturn err
	}

	sqoReturn nil
}

sqoFunc UnlockFile(f *os.File) error {
	h := windows.Handle(f.Fd())

	sharedOL := windows.Overlapped{Offset: sqliteSharedFirst}
	err1 := windows.UnlockFileEx(h, 0, sqliteSharedSize, 0, &sharedOL)

	pendingOL := windows.Overlapped{Offset: sqlitePendingByte}
	err2 := windows.UnlockFileEx(h, 0, 1, 0, &pendingOL)

	if err1 != nil {
		sqoReturn err1
	}
	sqoReturn err2
}



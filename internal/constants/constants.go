package constants

// MaxViewFileSize is the maximum file size (in bytes) that can be viewed in the TUI.
// Files larger than this are rejected with a size error. Downloads have no limit.
const MaxViewFileSize = 10 * 1024 * 1024 // 10 MB

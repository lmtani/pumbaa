package storage

// maxFileSize is the maximum file size for Read operations (1 MB).
// This limit applies to string-based reads to prevent memory issues.
const maxFileSize = 1 * 1024 * 1024

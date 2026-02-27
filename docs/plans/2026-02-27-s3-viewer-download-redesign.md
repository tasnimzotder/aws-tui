# S3 Viewer/Download Redesign

## Summary

Remove the image viewer feature from S3, extract the text viewer into a reusable service-agnostic component with syntax highlighting/search/wrap toggle, and enhance the download experience with a progress bar, ETA, and cancellation support.

## Section 1: Remove Image Viewer

Delete entirely:
- `internal/tui/services/imgrender.go`

Clean up:
- `s3_view.go` — remove `imageExts`, `isImage`, `imgData`, `decodedImg`, `imgRendered`, `renderImage()`, image decoder imports (`image/*`, `golang.org/x/image/*`)
- `s3_view_test.go` — remove image-related tests (`RenderImage_Valid`, `RenderImage_NoDecoded`, `RenderHalfBlock`, `RenderImageAuto`, `ProtocolName`)
- `go.mod` — remove `rasterm`, `pixterm`, `disintegration/imaging`, `golang.org/x/image`

## Section 2: Reusable Text Viewer

**New file:** `internal/tui/services/textview.go`

Standalone component implementing `View` interface. Used by S3 and any future service.

```go
type TextViewConfig struct {
    Title    string
    Content  []byte
    Filename string // extension-based language detection
}
```

Features:
- **Syntax highlighting** — `chroma` library, auto-detect language from extension, terminal theme (monokai256)
- **Search** — `/` opens search input, `n`/`N` cycle matches, highlighted in content
- **Word wrap toggle** — `w` toggles soft-wrap vs horizontal scroll
- **Line numbers** — `LeftGutterFunc` for code files
- **Status bar** — filename, size, scroll %, wrap mode, search match count
- **JSON pretty-printing** — `.json`/`.jsonl` auto-formatted
- **Binary detection** — shows message with download suggestion

## Section 3: Enhanced Download

Rewrite `s3_download.go` with:

1. Text input for destination path (pre-filled `~/Downloads/<filename>`)
2. Streaming download via new `GetObjectStream(ctx, bucket, key, region) (io.ReadCloser, int64, error)`
3. Progress bar (Bubble Tea `progress` bubble) showing:
   - Visual bar
   - Bytes downloaded / total
   - Percentage
   - ETA from download speed
4. Esc to cancel — cancels context, cleans up partial file
5. Success/error on completion

## S3 Client Changes

- Add `GetObjectStream` to `S3API` interface and `Client` — returns `(io.ReadCloser, int64, error)`
- Keep existing `GetObject` for text viewer (needs full body)

## S3 View Integration

- `s3_view.go` becomes a thin wrapper: fetches content via `GetObject`, then pushes `NewTextView(cfg)`
- Or: remove `s3_view.go` entirely and have the `v` key handler in `s3.go` fetch + push `TextViewer` directly

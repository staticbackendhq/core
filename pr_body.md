## What changed
Sanitized the pagination parameters in `getPagination()` (`db.go`) to clamp `page` and `size` to a minimum value of 1 when 0 or negative values are provided.

## Why
When 0 is sent as the page number (e.g., from the Go client library which sends 0 as the default int value), an EOF error is returned. Similarly, a size of 0 or negative values could cause unexpected behavior.

Fixes #135

## Testing
Added `TestGetPaginationSanitizesValues` in `db_test.go` with table-driven subtests covering:
- Default values when no params are provided
- Valid values pass through unchanged
- Zero page/size get clamped to 1
- Negative page/size get clamped to 1
- Both zero simultaneously

All existing tests continue to pass.

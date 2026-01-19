# Go Code Style Guide

This specification defines Go code writing standards for the project, based on [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

## Basic Rules

- Use `golangci-lint` for code checking
- Comments in English
- Logs in English, must conform to log levels
- Variables use camelCase, exported variables use PascalCase

## Uber Go Style Guide Core Rules

### 1. Interface Compliance Verification

Use compile-time checks to ensure types implement interfaces:

```go
var _ http.Handler = (*Handler)(nil)
```

### 2. Zero-value Mutex is Valid

No need to initialize pointers:

```go
var mu sync.Mutex  // Correct
mu := new(sync.Mutex)  // Avoid
```

### 3. Copy Slices/Maps at Boundaries

Create copies when receiving or returning to prevent external modification:

```go
func (d *Driver) SetTrips(trips []Trip) {
    d.trips = make([]Trip, len(trips))
    copy(d.trips, trips)
}
```

### 4. Use defer for Resource Cleanup

Use defer to release resources like files and locks:

```go
p.Lock()
defer p.Unlock()
```

### 5. Channel Size 1 or Unbuffered

Avoid using arbitrary-sized buffered channels.

### 6. Enums Start from 1

Avoid zero-value ambiguity:

```go
const (
    Add Operation = iota + 1
    Subtract
)
```

### 7. Error Handling Rules

- Use `pkg/errors` to wrap errors, return errors instead of panic
- Handle errors only once, don't both log and return
- Use `%w` to wrap errors to support `errors.Is/As`
- Error variables use `Err` prefix, error types use `Error` suffix

### 8. Don't Panic

Avoid panic in production code, return error:

```go
func run(args []string) error {
    if len(args) == 0 {
        return errors.New("an argument is required")
    }
    return nil
}
```

### 9. Avoid Mutable Global Variables

Use dependency injection instead.

### 10. Avoid init()

Unless necessary, put initialization logic in main() or constructors.

### 11. Avoid Goroutine Leaks

Every goroutine must have a predictable exit point.

## Performance Rules

- Prefer `strconv` over `fmt` for type conversion
- Avoid repeated string-to-byte conversion
- Specify capacity when initializing maps/slices

## Style Rules

- Soft line length limit: 99 characters
- Use field names when initializing structs
- Omit zero-value fields in structs
- nil is a valid empty slice, use `len(s) == 0` to check empty
- Reduce nesting, return early for error handling
- Place exported functions at the top of files, ordered by call sequence
- Unexported global variables use `_` prefix

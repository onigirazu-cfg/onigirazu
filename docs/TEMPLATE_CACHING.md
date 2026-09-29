# Template Cache

`template.Engine.Render` (`internal/template/engine.go`) keeps parsed Go templates in an
in-memory cache (`internal/cache/template_cache.go`). The cache is internal: it has no
configuration keys or flags, and its statistics are not exported anywhere.

## What is cached

Render first evaluates `{{ … }}` expressions and `{% if %}` conditions it can handle with
the expression evaluator (`internal/expression`) and expands `{% for %}` loops. Only
text that still contains `{{` or `{%` after that is converted to Go template syntax and
parsed; that parsed template is what the cache holds. Rendering a template that needs no
Go template step does not touch the cache.

## Behaviour

- Key: SHA-256 of the converted template text.
- Limit: 1000 entries; when full, the least recently used entry is dropped.
- TTL: 30 minutes from the time an entry was stored (reads do not extend it). An expired
  entry is dropped on the next read, and a background goroutine removes expired entries
  every 5 minutes.
- Safe for concurrent use.
- Each `template.NewEngine()` / `NewEngineWithPlugins()` gets its own cache;
  `Engine.Close()` stops the cleanup goroutine and empties it.

## API

```go
engine := template.NewEngine()
defer engine.Close()

out, err := engine.Render(ctx, "Hello {{ name }}", vars)

stats := engine.GetCacheStats() // cache.TemplateCacheStats
_ = engine.ClearCache(ctx)
```

`TemplateCacheStats` fields: `TotalEntries`, `ExpiredEntries`, `ActiveEntries`, `Hits`,
`Misses`, `Evictions`, `HitRate` (percent), `MaxSize`.

## Tests and benchmarks

```bash
go test ./internal/cache/ -run TestTemplateCache
go test ./internal/cache/ -run '^$' -bench BenchmarkTemplateCache -benchmem
```

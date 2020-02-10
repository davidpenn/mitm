# mitm

mitm is a very simple proxy server that logs every request

## Usage

```sh
Usage:
  mitm [flags]

Flags:
      --database string            path to the cache database (default ".cached")
      --database-max-size string   max value size a cached object will save (default "100mb")
  -h, --help                       help for mitm
      --listen string              address the cache server will listen on (default ":3128")
      --replay                     replay requests from cache
```

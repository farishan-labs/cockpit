# Fleet Management

## Process Metadata Schema

Each registered process in the processor registry (`~/.config/fhl/processor.json`)
has the following fields:

- `name` (string, required): unique process name.
- `command` (string, required): executable path.
- `args` (string[]): command arguments.
- `workdir` (string): working directory.
- `env` (map): environment variables.
- `type` (string): one of `daemon`, `menubar`, `job`. Defaults to `daemon`.
- `health_url` (string): HTTP URL to probe for health. Optional.
- `log_path` (string): log file path.
- `auto_restart` (bool): restart on failure.
- `created_at` (timestamp).

## /api/fleet

Returns a JSON array of objects:

- `name`, `type`, `command`, `health` ("ok"|"down"|"n/a"|HTTP status),
  `health_url`, `latency_ms`.

Health is determined by HTTP GET to `health_url` with a 3s timeout; status codes
2xx/3xx are "ok".

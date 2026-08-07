# Pi extensions (rubickx)

## obs-jsonl

Local agent observability. Auto-loaded from `.pi/extensions/` after the project is trusted.

### Run

```bash
cd /Users/kevinxjiang/Workspace/rubickx
/Users/kevinxjiang/Workspace/pi-mono/pi-test.sh
```

On first start, trust the project when prompted (needed for project-local extensions).

### Use

- Footer status shows `obs → <logfile>`
- In pi: `/obs` or `/obs open`
- Another terminal: `tail -f .pi/obs/*.jsonl`

### Events

`session_start`, `agent_start/end`, `agent_settled`, `turn_start/end` (with usage), `provider_request/response`, `tool_start/end`, `model_select`, `session_shutdown`

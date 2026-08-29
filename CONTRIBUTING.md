# Contributing

Contributions are welcome, especially protocol captures and hardware testing.

## Development

```sh
go test ./...
make build
./bin/rogdelta desktop
```

Run `gofmt` on Go changes. Keep the core build free of third-party Go dependencies unless a
dependency provides a clear benefit that cannot reasonably be implemented with the standard
library.

## HID protocol research

The ROG Delta II vendor protocol is not publicly documented. The application therefore treats
unknown reports as untrusted and never sends guessed commands to the device.

Capture one action at a time:

```sh
rogdelta record --seconds 15 --output volume-up.jsonl
```

Before attaching a capture to an issue:

1. Review every event and remove unrelated data.
2. Describe the connection mode and exact action performed.
3. Include firmware information when known.
4. Never submit captures from unrelated HID devices.

Any code that writes vendor HID reports must include evidence for the report format and must
validate all values before sending them.

## Pull requests

Keep changes focused, explain how they were tested, and update documentation when behavior or
packaging changes.

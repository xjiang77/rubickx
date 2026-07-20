.PHONY: verify fmt-check vet test race run refresh-evidence clean

verify: fmt-check vet test race run

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

run:
	@all_tmp="$$(mktemp "$${TMPDIR:-/tmp}/netsec-all-evidence.XXXXXX")"; \
	browser_tmp="$$(mktemp "$${TMPDIR:-/tmp}/netsec-browser-evidence.XXXXXX")"; \
	trap 'rm -f "$$all_tmp" "$$browser_tmp"' EXIT; \
	go run ./cmd/netsec-lab -lab all -out "$$all_tmp"; \
	go run ./cmd/netsec-browser -self-test -evidence "$$browser_tmp"

refresh-evidence:
	go run ./cmd/netsec-lab -lab all -out evidence/all.json
	go run ./cmd/netsec-browser -self-test -evidence evidence/browser.json

clean:
	rm -f evidence/*.json

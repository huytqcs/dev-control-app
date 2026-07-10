.PHONY: build clean

# Single-binary packaging (docs/DELTA_PLAN.md): build the frontend, drop its
# dist/ into backend/internal/webui/dist so go:embed picks it up, then build
# the Go binary. Daily devctl development (`go run ./cmd/devctl` + `npm run
# dev`) doesn't need this — it's only for producing a binary to just run.
build:
	cd frontend && npm install && npm run build
	rm -rf backend/internal/webui/dist
	cp -r frontend/dist backend/internal/webui/dist
	cd backend && go build -o ../bin/devctl ./cmd/devctl

clean:
	rm -rf bin frontend/dist backend/internal/webui/dist
	mkdir -p backend/internal/webui/dist
	touch backend/internal/webui/dist/.gitkeep

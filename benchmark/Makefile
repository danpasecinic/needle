.PHONY: run quick full json help

help:
	@echo "Needle Benchmark Suite"
	@echo ""
	@echo "Usage:"
	@echo "  make run     - Run benchmarks with formatted output"
	@echo "  make quick   - Quick run with shorter benchmark time"
	@echo "  make full    - Full run with longer benchmark time"
	@echo "  make json    - Run and export results to JSON"
	@echo "  make raw     - Run raw Go benchmarks without formatting"
	@echo ""

run:
	@cd cmd && go run run.go

quick:
	@go test -bench=. -benchmem -benchtime=50ms

full:
	@go test -bench=. -benchmem -count=5 -benchtime=200ms

json:
	@cd cmd && go run run.go --json

raw:
	@go test -bench=. -benchmem -count=3 -benchtime=100ms

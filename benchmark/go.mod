module benchmark

go 1.25.4

replace github.com/danpasecinic/needle => ../

require (
	github.com/danpasecinic/needle v0.0.0-00010101000000-000000000000
	github.com/samber/do/v2 v2.0.0
	go.uber.org/dig v1.19.0
	go.uber.org/fx v1.24.0
)

require (
	github.com/samber/go-type-to-string v1.8.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
)

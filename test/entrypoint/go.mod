module test

go 1.25.0

replace github.com/k-lb/entrypoint-framework => ../..

require (
	github.com/k-lb/entrypoint-framework v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.12.0
	go.uber.org/mock v0.6.0
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	golang.org/x/sys v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

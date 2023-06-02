module github.com/chromium/hstspreload.org

go 1.15

require (
	cloud.google.com/go/datastore v1.11.0
	github.com/chromium/hstspreload v0.0.0-20230601210012-99f45f11d9af
	golang.org/x/net v0.10.0
	google.golang.org/api v0.125.0
	google.golang.org/grpc v1.55.0
)

require cloud.google.com/go/compute v1.20.0 // indirect

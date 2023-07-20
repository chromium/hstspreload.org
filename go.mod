module github.com/chromium/hstspreload.org

go 1.15

require (
	cloud.google.com/go/datastore v1.11.0
	github.com/chromium/hstspreload v0.0.0-20230623210540-9e257cb2df03
	golang.org/x/net v0.12.0
	google.golang.org/api v0.128.0
	google.golang.org/grpc v1.56.1
)

require (
	cloud.google.com/go v0.110.3 // indirect
	cloud.google.com/go/compute v1.20.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	golang.org/x/oauth2 v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
)

replace (
	github.com/chromium/hstspreload => ../hstspreload
	github.com/chromium/hstspreload/chromium/preloadlist => ../hstspreload/chromium/preloadlist
)

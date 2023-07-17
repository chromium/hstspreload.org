module github.com/chromium/hstspreload.org

go 1.15

require (
	cloud.google.com/go/datastore v1.12.1
	github.com/chromium/hstspreload v0.0.0-20230630230720-030fab72c822
	golang.org/x/net v0.12.0
	google.golang.org/api v0.130.0
	google.golang.org/grpc v1.56.2
)

require (
	cloud.google.com/go v0.110.5 // indirect
	cloud.google.com/go/compute v1.21.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	golang.org/x/oauth2 v0.10.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230710151506-e685fd7b542b // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230710151506-e685fd7b542b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230710151506-e685fd7b542b // indirect
)

replace github.com/chromium/hstspreload => ../hstspreload
replace github.com/chromium/hstspreload/chromium/preloadlist => ../hstspreload/chromium/preloadlist
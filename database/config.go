package database

import "time"

const (
	localProjectID = "hstspreload-local"
	prodProjectID  = "hstspreload"

	timeout        = 10 * time.Second
	scanAllTimeout = 1000 * time.Second

	// Shared publicly using:
	//     gsutil acl set public-read gs://hstspreload
	scanBucketName = "hstspreload"
	// testScanBucketName = "hstspreload-scans-test"
)

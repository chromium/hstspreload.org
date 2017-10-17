<center>
<img src="frontend/static/app-icon.png" width=144>
</center>

# `hstspreload.org`

[![Build Status](https://travis-ci.org/chromium/hstspreload.org.svg?branch=master)](https://travis-ci.org/chromium/hstspreload.org)

This folder contains the source for `v2` of [hstspreload.org](https://hstspreload.org/).

See [github.com/chromium/hstspreload](https://github.com/chromium/hstspreload) for the core library that checks websites against the submission requirements.

## Development

Requirements

- A `go` development environment.
- The `java` commandline program for running JAR files (for the Cloud Datastore Emulator).

    go get github.com/chromium/hstspreload.org
    cd $GOPATH/src/github.com/chromium/hstspreload.org
    make serve

The first tie you run it, `make serve` will download the [Cloud Datastore Emulator](https://cloud.google.com/datastore/docs/tools/datastore-emulator) (≈115MB) to a cache directory.

## Disclaimer

This project is used by the Chromium team to maintain the HSTS preload list. This is not an official Google product.

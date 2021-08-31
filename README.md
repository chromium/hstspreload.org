<p align="center">
  <a href="https://hstspreload.org/">
    <img src="frontend/static/app-icon.png" alt="" width="144" height="144">
  </a>

  <h1 align="center">hstspreload.org</h1>
</p>


This folder contains the source for the HSTS preload list submission website at [hstspreload.org](https://hstspreload.org/).

See [github.com/chromium/hstspreload](https://github.com/chromium/hstspreload) for the core library that checks websites against the submission requirements.

## Development

Requirements

- A `go` development environment.
- The `java` commandline program for running JAR files (for the Cloud Datastore Emulator).

  ```shell
  go get github.com/chromium/hstspreload.org
  cd $GOPATH/src/github.com/chromium/hstspreload.org
  make serve
  ```

The first time you run it, `make serve` will download the [Cloud Datastore Emulator](https://cloud.google.com/datastore/docs/tools/datastore-emulator) (≈115MB) to a cache directory.

### Deployment

If you have access to the Google Cloud `hstspreload` project:

    make deploy

Unfortunately, this usually takes 5-10 minutes.

## Disclaimer

This project is used by the Chromium team to maintain the HSTS preload list. This is not an official Google product.

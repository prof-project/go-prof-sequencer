# go-prof-sequencer
This project implements a private order flow sequencer for Ethereum, allowing transactions to be submitted, validated, and ordered before being bundled and sent to a bundle merger via gRPC.

## How-To
1. Clone the repository
2. Run `make init` to prepare the project
3. Run `make build` to build the app
4. Navigate to the `test/bundlemerger` directory and run `make` to build the dummy bundle merger
5. Run `./bundlemerger` to start the dummy bundle merger
6. Navigate to the `app` directory and run `./app` to start the sequencer

As a snippet that would be:

```bash
make init
make build
cd test/bundlemerger
make
./bundlemerger &
cd ../app
./app &
```

## Configuration
- The gRPC URL and TLS usage can be configured via command-line flags:
  - `--grpc-url` (default: `127.0.0.1:50051`)
  - `--use-tls` (default: `false`)

## Logging
- Logging can be configured via command-line flags:
  - `--log-level` (default: `info`)
  - `--log-to-file` (default: `false`)

## Metrics
- Prometheus metrics can be enabled via the `--enable-metrics` flag.

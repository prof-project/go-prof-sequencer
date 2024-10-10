# go-prof-sequencer
This project implements a private order flow sequencer for Ethereum, allowing transactions to be submitted, validated, and ordered before being bundled and sent to a bundle merger via gRPC.

# How-To
1. Clone the repository
2. Run make init to prepare the project
3. Run make build to build the app
4. cd into test/bundle-merger, and run ./bundlemerger to start the dummy bundle merger
5. cd into app, and run ./app to start the sequencer

As a snippet that would be:

```bash
make init
make build
cd ../test/bundlemerger; ./bundlemerger
cd app; ./app &
```

module app

go 1.23.2

require (
	github.com/ethereum/go-ethereum v1.14.11
	github.com/google/uuid v1.6.0
	github.com/prof-project/prof-grpc/go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.67.1
)

require (
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240223125850-b1e8a79f509c // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-verkle v0.1.1-0.20240829091221-dffa7562dbe9 // indirect
	github.com/holiman/uint256 v1.3.1 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace github.com/prof-project/prof-grpc/go => ../lib/prof-grpc/go

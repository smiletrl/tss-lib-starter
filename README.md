# Tss Lib integration starter

This is a starter to implement [tss-lib](https://github.com/bnb-chain/tss-lib) with Go. In particular, it makes use of ESDSA for Threshold Signature Scheme. Refer [eprint.iacr.org/2019/114.pdf](https://eprint.iacr.org/2019/114.pdf).

[Protobuf](https://github.com/golang/protobuf) is used for message exchange between parties. But it's important to know that this is just a starter, remember to read [how-to-use-this-securely](https://github.com/bnb-chain/tss-lib?tab=readme-ov-file#how-to-use-this-securely).

# Mock p0

`p0` holds a test env which manages parties within one Go app. These parties run separately within different goroutines

```
cd p0 && go run .
```

and we would expect to see result like

```
...
ACTIVE GOROUTINES: 9
ACTIVE GOROUTINES: 5
Done. Received save data from 3 participants at keygen
ACTIVE GOROUTINES: 3
Done. Received save data from 3 participants at signing
Signature verify result: [true]
```

# Mock p1, p2, p3, p4

`p1`, `p2`, `p3`, `p4` mock four different nodes, like four devices. Each node will have its own keygen and sign method. The idea is to use grpc to do message exchange.

open four terminals and run each line at each separate terminal window

```
cd p1 && go run .
cd p2 && go run .
cd p3 && go run .
cd p4 && go run .
```

and if all goes well, we shall see something like

![Screen for For Parties](screen-for-four-parties.png)

```
smiletrl@Rulins-MacBook-Pro p1 % go run .
2024/05/10 00:11:24 prepare keygen
2024/05/10 00:11:24 grpc server starts
2024/05/10 00:11:24 grpc server listens at local port: :50051
2024/05/10 00:11:34 wait for keygen
2024/05/10 00:11:34 Keygen ACTIVE GOROUTINES: 5
2024/05/10 00:11:52 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound1Message, From: {0,tss1}, To: all
2024/05/10 00:11:52 Keygen ACTIVE GOROUTINES: 5
2024/05/10 00:11:54 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message1, From: {0,tss1}, To: [{1,tss2}]
2024/05/10 00:11:54 Keygen ACTIVE GOROUTINES: 34
2024/05/10 00:11:54 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message1, From: {0,tss1}, To: [{2,tss3}]
2024/05/10 00:11:54 Keygen ACTIVE GOROUTINES: 35
2024/05/10 00:11:55 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message2, From: {0,tss1}, To: all
2024/05/10 00:11:55 Keygen ACTIVE GOROUTINES: 33
2024/05/10 00:11:56 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound3Message, From: {0,tss1}, To: all
2024/05/10 00:11:56 Keygen ACTIVE GOROUTINES: 32
2024/05/10 00:11:57 keygen save data done start
2024/05/10 00:11:57 keygen save data done
2024/05/10 00:11:57 Keygen ACTIVE GOROUTINES: 31
2024/05/10 00:11:57 keygen process finished
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 33
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound1Message1, From: {0,tss1}, To: [{1,tss2}]
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound1Message1, From: {0,tss1}, To: [{2,tss3}]
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound1Message2, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 29
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound2Message, From: {0,tss1}, To: [{1,tss2}]
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 27
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound2Message, From: {0,tss1}, To: [{2,tss3}]
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound3Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 27
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound4Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound5Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound6Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 27
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound7Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 27
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound8Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 27
2024/05/10 00:12:07 sign out msg: Type: binance.tsslib.ecdsa.signing.SignRound9Message, From: {0,tss1}, To: all
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 28
2024/05/10 00:12:07 Signature raw data: signature:"\xa5_\xee\x01\xc64_\x93\xb9R\t\xe7T\xa8\xed\xf7\x82\x1b\x18i\x19\xadK6\x8c\r\xac\xe6\xfa\xf0\x1a{H$yG\xf5\x8a\xca2\"\x14\xd2\xe7ͦ!X\xa5\x8dh\xf0\x8c<\xf1\x81\r>\x96\xad\xbc]\x9f\xac" signature_recovery:"\x01" r:"\xa5_\xee\x01\xc64_\x93\xb9R\t\xe7T\xa8\xed\xf7\x82\x1b\x18i\x19\xadK6\x8c\r\xac\xe6\xfa\xf0\x1a{" s:"H$yG\xf5\x8a\xca2\"\x14\xd2\xe7ͦ!X\xa5\x8dh\xf0\x8c<\xf1\x81\r>\x96\xad\xbc]\x9f\xac" m:"Hello World"
2024/05/10 00:12:07 Signature verify result: [true]
2024/05/10 00:12:07 Signing ACTIVE GOROUTINES: 26
```

The last two line `2024/05/10 00:12:07 Signature verify result: [true]` indicates the signature has been verified.

We may play with less than threshold+1 nodes and see how it works, like change `pkg/constants/const.go`

```
	SelectedParties = map[string]struct{}{
		"p1": {},
		"p2": {},
		// "p3": {},
		// "p4": {},
	}
```

and run above terminal commands, and we will get error like

```
smiletrl@Rulins-MacBook-Pro p1 % go run .
2024/05/10 10:24:07 prepare keygen
2024/05/10 10:24:07 grpc server starts
2024/05/10 10:24:07 grpc server listens at local port: :50051
2024/05/10 10:24:12 wait for keygen
2024/05/10 10:24:22 Keygen ACTIVE GOROUTINES: 22
2024/05/10 10:24:40 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound1Message, From: {0,tss1}, To: all
2024/05/10 10:24:40 Keygen ACTIVE GOROUTINES: 8
2024/05/10 10:24:43 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message1, From: {0,tss1}, To: [{1,tss2}]
2024/05/10 10:24:43 Keygen ACTIVE GOROUTINES: 42
2024/05/10 10:24:43 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message1, From: {0,tss1}, To: [{2,tss3}]
2024/05/10 10:24:43 Keygen ACTIVE GOROUTINES: 42
2024/05/10 10:24:43 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message1, From: {0,tss1}, To: [{3,tss4}]
2024/05/10 10:24:43 Keygen ACTIVE GOROUTINES: 43
2024/05/10 10:24:43 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound2Message2, From: {0,tss1}, To: all
2024/05/10 10:24:43 Keygen ACTIVE GOROUTINES: 41
2024/05/10 10:24:46 keygen out msg: Type: binance.tsslib.ecdsa.keygen.KGRound3Message, From: {0,tss1}, To: all
2024/05/10 10:24:46 Keygen ACTIVE GOROUTINES: 41
2024/05/10 10:24:47 keygen save data done start
2024/05/10 10:24:47 keygen save data done
2024/05/10 10:24:47 Keygen ACTIVE GOROUTINES: 40
2024/05/10 10:24:47 keygen process finished
2024/05/10 10:24:57 Signing ACTIVE GOROUTINES: 35
2024/05/10 10:24:57 sign err: task signing, party {0,tss1}, round 1: t+1=3 is not satisfied by the key count of 2
2024/05/10 10:24:57 Signing ACTIVE GOROUTINES: 35
```

This last two line `2024/05/10 10:24:57 sign err: task signing, party {0,tss1}, round 1: t+1=3 is not satisfied by the key count of 2` has indicated that signature fails with non-sufficient parties

!Important note: every time we want to sign a new message, we have to restart all four parties.

# Change proto

In case you want to play with grpc server, here's the command to generate proto files.

```
cd pkg/grpc
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    p2p.proto
```

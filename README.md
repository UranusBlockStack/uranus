# uranus

[![Build Status](https://travis-ci.org/UranusBlockStack/uranus.svg?branch=master)](https://travis-ci.org/UranusBlockStack/uranus)
[![GoDoc](https://godoc.org/github.com/UranusBlockStack/uranus?status.svg)](https://godoc.org/github.com/UranusBlockStack/uranus)
[![LGPL-3.0](https://img.shields.io/badge/license-LGPL--3.0-blue.svg)](./LICENSE)

## What is Uranus blockchain

A fast and scalable blockchain,which is Fully compatible with Ethereum but more faster, safer, and optimized for hundreds of thousands of resources contributors, resources users and DAPP developers.The Uranus chain innovatively proposed a DPOS hybrid consensus algorithm based on DPOS with distributed computing power Weight Harmonic algorithm.

## Mission

Empower the redundant computing power of the world by providing ubiquitous and shared-computing services beyond centralized public clouds (i.e. Amazon, Microsoft, Ali) with blockchain technology. Reconstructing the landscape of the global computing-power service market and utilizing containers to integrate various applications on a distributed-computing platform, building a new ecosphere based on a public blockchain.

## Technical Essential

The Uranus system is a decentralized-computing, power-sharing platform based on blockchain technology and cloud native/microservice. It establishes a new intelligent computing-power resource-delivery system as well as value-exchange system. The core technology of the Uranus system lies in the extension of its public blockchain and the distributed-container technology.


Please check [Uranus whitepaper](https://uranus.io/static/Uranus-Technical-WhitePaper-EN-V2.0.pdf) for more infomation

## Install


Ensure Go with the supported version is installed properly:

```bash
$ go version
$ go env GOROOT GOPATH
```

- Get the source code

``` bash
$ git clone https://github.com/UranusBlockStack/uranus.git $GOPATH/src/github.com/UranusBlockStack
```

- Build source code

``` bash
$ cd $GOPATH/src/github.com/UranusBlockStack/uranus
$ make all

```

## Contribution

Uranus is still in active development,We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!If you'd like to contribute to uranus, please fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base.

Here are some guidelines before you start:

 * Code must adhere to the official Go formatting guidelines (i.e. uses gofmt).
 * Code must be documented adhering to the official Go commentary guidelines.
 * Pull requests need to be based on and opened against the master branch.
 * Commit messages should be prefixed with the package(s) they modify.

## License


The uranus library is licensed under the [GNU Lesser General Public License v3.0](./LICENSE)

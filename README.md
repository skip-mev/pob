# Protocol-Owned Builder

[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#wip)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://godoc.org/github.com/skip-mev/pob)
[![Go Report Card](https://goreportcard.com/badge/github.com/skip-mev/pob?style=flat-square)](https://goreportcard.com/report/github.com/skip-mev/pob)
[![Version](https://img.shields.io/github/tag/skip-mev/pob.svg?style=flat-square)](https://github.com/skip-mev/pob/releases/latest)
[![License: Apache-2.0](https://img.shields.io/github/license/skip-mev/pob.svg?style=flat-square)](https://github.com/skip-mev/pob/blob/main/LICENSE)
[![Lines Of Code](https://img.shields.io/tokei/lines/github/skip-mev/pob?style=flat-square)](https://github.com/skip-mev/pob)

Skip Protocol's Protocol-Owned Builder (POB) is a set of Cosmos SDK and ABCI++
primitives that provides application developers the ability to define how their
apps construct and validate blocks in a transparent on-chain enforceable way,
such as giving complete control to the protocol to recapture, control, and
redistribute MEV.

Skip's POB provides developers with a set of a few core primitives:

* `x/builder`: This Cosmos SDK module gives applications the ability to process
  MEV bundled transactions in addition to having the ability to define how searchers
  and block proposers are rewarded. In addition, the module defines a `AuctionDecorator`,
  which is an AnteHandler decorator that enforces various chain configurable MEV
  rules.
* `ProposalHandler`: This ABCI++ handler defines `PrepareProposal` and `ProcessProposal`
  methods that give applications the ability to perform top-of-block auctions,
  which enables recapturing, redistributing and control over MEV. These methods
  are responsible for block proposal construction and validation.
* `AuctionMempool`: An MEV-aware mempool that enables searchers to submit bundled
  transactions to the mempool and have them bundled into blocks via a top-of-block
  auction. Searchers include a bid in their bundled transactions and the highest
  bid wins the auction. Application devs have control over levers that control
  aspects such as the bid floor and minimum bid increment.

## Releases

### Release Compatibility Matrix

| POB Version | Cosmos SDK |
| :---------: | :--------: |
|   v1.x.x    |  v0.47.x   |

## Install

```shell
$ go install github.com/skip-mev/pob
```

## Setup

1. Import the necessary dependencies into your application. This includes the
   proposal handlers, mempool, keeper, builder types, and builder module. This
   tutorial will go into more detail into each of the dependencies.

   ```go
   import (
     ...
     proposalhandler "github.com/skip-mev/pob/abci"
     "github.com/skip-mev/pob/mempool"
     "github.com/skip-mev/pob/x/auction"
     auctionkeeper "github.com/skip-mev/pob/x/auction/keeper"
     auctiontypes "github.com/skip-mev/pob/x/auction/types"
     ...
   )
   ```

2. Add your module to the the `AppModuleBasic` manager. This manager is in
   charge of setting up basic, non-dependent module elements such as codec
   registration and genesis verification. This will register the special
   `MsgAuctionBid` message. When users want to bid for top of block execution,
   they will submit a transaction - which we call an auction transaction - that
   includes a single `MsgAuctionBid`. We prevent any other messages from being
   included in auction transaction to prevent malicious behavior - such as front
   running or sandwiching.

   ```go
   var (
     ...
     ModuleBasics = module.NewBasicManager(
       ...
       auction.AppModuleBasic{},
     )
     ...
   )


   func NewApp(...) *App {

     ...
     app.ModuleManager = module.NewManager(
       ...
       auction.NewAppModule(appCodec, app.AuctionKeeper),
       ...
     )
     ...
   }
   ```

3.

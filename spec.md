
# POB Specification

## Abstract

The `x/builder` module is a Cosmos SDK module that allows Cosmos chains to host
top-of-block auctions directly in-protocol with auction revenue (MEV) being
redistributed according to the preferences of the chain. The `x/builder` module
introduces a new `MsgAuctionBid` message that allows users to submit a bid
alongside an ordered list of transactions, i.e. a **bundle**, that they want
executed at top-of-block before any other transactions are executed for that
block. The `x/builder` module works alongside the `AuctionMempool` such that:

* Auctions are held directly in the `AuctionMempool`, where a winner is determined
  when the proposer proposes a new block in `PrepareProposal`.
* `x/builder` provides the necessary validation of auction bids and subsequent
  state transitions to extract bids.

## Concepts

### Miner Extractable Value (MEV)

MEV refers to the potential profit that miners, or validators in a Proof-of-Stake
system, can make by strategically ordering, selecting, or even censoring
transactions in the blocks they produce. MEV can be classified into "good MEV"
and "bad MEV" based on the effects it has on the blockchain ecosystem and its
users. It's important to note that these classifications are subjective and may
vary depending on one's perspective.

**Good MEV** refers to the value that validators can extract while contributing
positively to the blockchain ecosystem. This typically includes activities that
enhance network efficiency, maintain fairness, and align incentives with the
intended use of the system. Examples of good MEV include:

* **Back-running**: Validators can place their own transactions immediately
  after a profitable transaction, capitalizing on the changes caused by the
  preceding transaction.
* **Arbitrage**: By exploiting price differences across decentralized exchanges
  or other DeFi platforms, validators help maintain more consistent price levels
  across the ecosystem, ultimately contributing to its stability.
* **Liquidations**: In DeFi platforms, when users' collateral falls below a
  specific threshold, validators can liquidate these positions, thereby maintaining
  the overall health of the platform and protecting its users from insolvency risks.

**Bad MEV** refers to the value that validators can extract through activities
that harm the blockchain ecosystem, lead to unfair advantages, or exploit users.
Examples of bad MEV include:

* **Front-running**: Validators can observe pending transactions in the mempool
  (the pool of unconfirmed transactions) and insert their own transactions ahead
  of them. This can be particularly profitable in decentralized finance (DeFi)
  applications, where a validator could front-run a large trade to take advantage
  of price movements.
* **Sandwich attacks**: Validators can surround a user's transaction with their
  own transactions, effectively manipulating the market price for their benefit.
* **Censorship**: Validators can selectively exclude certain transactions from
  blocks to benefit their own transactions or to extract higher fees from users.

MEV is a topic of concern in the blockchain community because it can lead to
unfair advantages for validators, reduced trust in the system, and a potential
concentration of power. Various approaches have been proposed to mitigate MEV,
such as proposer-builder separation (described below) and transparent and fair
transaction ordering mechanisms at the protocol-level (`POB`) to make MEV
extraction more incentive aligned with the users and blockchain ecosystem.

### Proposer Builder Separation (PBS)

Proposer-builder separation is a concept in the design of blockchain protocols,
specifically in the context of transaction ordering within a block. In traditional
blockchain systems, validators perform two main tasks: they create new blocks
(acting as proposers) and determine the ordering of transactions within those
blocks (acting as builders).


**Proposers**: They are responsible for creating and broadcasting new blocks,
just like in traditional blockchain systems. *However, they no longer determine
the ordering of transactions within those blocks*.

**Builders**: They have the exclusive role of determining the order of transactions
within a block - can be full or partial block. Builders submit their proposed
transaction orderings to an auction mechanism, which selects the winning template
based on predefined criteria, e.g. highest bid.

This dual role can lead to potential issues, such as front-running and other
manipulations that benefit the miners/builders themselves.

* *Increased complexity*: Introducing PBS adds an extra layer of complexity to
  the blockchain protocol. Designing, implementing, and maintaining an auction
  mechanism for transaction ordering requires additional resources and may
  introduce new vulnerabilities or points of failure in the system.
* *Centralization risks*: With PBS, there's a risk that a few dominant builders
  may emerge, leading to centralization of transaction ordering. This centralization
  could result in a lack of diversity in transaction ordering algorithms and an
  increased potential for collusion or manipulation by the dominant builders.
* *Incentive misalignments*: The bidding process may create perverse incentives
  for builders. For example, builders may be incentivized to include only high-fee
  transactions to maximize their profits, potentially leading to a neglect of
  lower-fee transactions. Additionally, builders may be incentivized to build
  blocks that include **bad-MEV** strategies because they are more profitable.

## Specification

## Messages

POB defines a new Cosmos SDK `Message`, `MsgAuctionBid`, that allows users to
create an auction bid and participate in a top-of-block auction. The `MsgAuctionBid`
message defines a bidder and a series of embedded transactions, i.e. the bundle.

```protobuf
message MsgAuctionBid {
  option (cosmos.msg.v1.signer) = "bidder";
  option (amino.name) = "pob/x/builder/MsgAuctionBid";

  option (gogoproto.equal) = false;

  // bidder is the address of the account that is submitting a bid to the
  // auction.
  string bidder = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
  // bid is the amount of coins that the bidder is bidding to participate in the
  // auction.
  cosmos.base.v1beta1.Coin bid = 3
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];
  // transactions are the bytes of the transactions that the bidder wants to
  // bundle together.
  repeated bytes transactions = 4;
}
```

Note, the `transactions` may or may not exist in a node's application mempool. If
a transaction containing a single `MsgAuctionBid` wins the auction, the block
proposal will automatically include the `MsgAuctionBid` transaction along with
injecting all the bundled transactions such that they are executed in the same
order after the `MsgAuctionBid` transaction.

## Mempool

As the lifeblood of blockchains, mempools serve as the intermediary space for
pending transactions, playing a vital role in transaction management, fee markets,
and network health. With ABCI++, mempools can be defined at the application layer
instead of the consensus layer (CometBFT). This means applications can define
their own mempools that have their own custom verification, block building, and
state transition logic. Adding on, these changes make it such that blocks are
built (`PrepareProposal`) and verified (`ProcessProposal`) directly in the
application layer.

The `x/builder` module implements an application-side mempool, `AuctionMempool`,
that implements the `sdk.Mempool` interface. The mempool is composed of two
primary indexes, a global index that contains all non-auction transactions and
an index that only contains auction transactions, i.e. transactions with a single
`MsgAuctionBid` message. Both indexes order transactions based on priority respecting
the sender's sequence number. The global index prioritizes transactions based on
`ctx.Priority()` and the auction index prioritizes transactions based on the
bid.

### Prepare Proposal

After the proposer of the next block has been selected, the proposer will call `PrepareProposal` to build the next block. The block will be built in two stages. First, it will host the auction and include the winning bidder's bundle as the first set of transactions for the block. The auction currently supports only a single winner. Selecting the auction winner involves a greedy search for a valid auction transaction starting from highest paying bid (respecting user nonce) in the `AuctionMempool`. The `x/builder`'s antehandler is responsible for verifying the auction transaction based on the criteria described below (see **Ante Handler**).

Then, it will build the rest of the block by reaping and validating the transactions in the normal global mempool. The second portion of block building iterates from highest to lowest priority transactions in the global mempool and adds them to the proposal if they are valid. If the proposer comes across a transaction that was already included in the top of block, it will be ignored.

### Process Proposal

After the proposer proposes a block of transactions for the next block, the block will be verifed by other nodes in the network in `ProcessProposal`. If there is an auction transaction in the proposal, it must be the first transaction in the proposal and all bundled transactions must follow the auction transaction in the exact order we would expect them to be seen. If this fails, the proposal is rejected. If this passes, the validator will then run `CheckTx` on all of the transactions in the block in the order in which they were provided in the proposal.

---

## State

### State Objects

| State Object | Description | Key | Values | Store |
| --- | --- | --- | --- | --- |
| Params | Tracks the parameters of the module | []byte{0} | []byte{buildertypes.Params} | KV |

#### Params

This state object contains the `x/builder` module parameters and auction configuration desired by the application. All of these parameters are customizable through governance.

```protobuf
// Params defines the parameters of the x/builder module.
message Params {
  option (amino.name) = "cosmos-sdk/x/builder/Params";

  // max_bundle_size is the maximum number of transactions that can be bundled
  // in a single bundle.
  uint32 max_bundle_size = 1;

  // escrow_account_address is the address of the account that will receive a
  // portion of the bid proceeds.
  string escrow_account_address = 2;

  // reserve_fee specifies the bid floor for the auction.
  cosmos.base.v1beta1.Coin reserve_fee = 3
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // min_buy_in_fee specifies the fee that the bidder must pay to enter the
  // auction.
  cosmos.base.v1beta1.Coin min_buy_in_fee = 4
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // min_bid_increment specifies the minimum amount that the next bid must be
  // greater than the previous bid.
  cosmos.base.v1beta1.Coin min_bid_increment = 5
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];

  // front_running_protection specifies whether front running and sandwich
  // attack protection is enabled.
  bool front_running_protection = 6;

  // proposer_fee defines the portion of the winning bid that goes to the block
  // proposer that proposed the block.
  string proposer_fee = 7 [
    (gogoproto.nullable) = false,
    (gogoproto.customtype) = "github.com/cosmos/cosmos-sdk/types.Dec"
  ];
}
```

#### **Max Bundle Size**

This is the maximum number of transactions that can be bundled together into a single bundle.

#### **Escrow Account Address**

This is the address of the auction house that will be receiving and accuring auction revenue.

#### **Reserve Fee**

This specifics the minimum bid (bid floor) needed to enter the auction.

#### **Min Buy In Fee**

This specifics the minimum the auction winner must pay to have their bundle included at the top of the block.

#### **Min Bid Increment**

This specifies the epsilon bid that each subsequent must be greater than to be considered in the auction i.e. if we see a bid of 10 and the min bid increment is 10 the next bid must be at least 20.

#### **Frontrunning Protection**

This specifies whether front-running and sandwich protection is enabled in the auction.

#### **Proposer Fee**

This specifies the portion of auction bid that goes to the block proposer that proposed the block.

---

## State Transitions

The `x/builder` module works alongside the `AuctionMempool` - a customized `PriorityNonce` mempool - to introduce state transitions. Auctions are held directly in the `AuctionMempool` where the winner is determined when the proposer proposes a new block in `PrepareProposal`. `x/builder` provides the necessary validation of auction bids and subsequent state transitions to extract bids.

### Ante Handler

As described, when users want to bid for the rights for top of block execution they will submit a special `AuctionTx` which is just a normal `sdk.Tx` transaction with a single `MsgAuctionBid`. The ante handler is responsible for verification of this `AuctionTx`. The ante handler will verify that:

1. The auction transaction specifies a timeout height where the bid is no longer considered valid.
2. The auction transaction includes less than MaxBundleSize transactions in its bundle.
3. The auction transaction includes ***only*** a single `MsgAuctionBid` message. We enforce that no other messages are included to prevent front-running.
4. Enforce that the user has sufficient funds to pay the bid they entered while covering all auction fees.
5. Enforce that the `AuctionTx` is min bid increment greater than the next closest bid.
6. Enforce that the bundle of transactions the bidder provided does not front-run or sandwich (if enabled).

Note, the process of selecting auction winners occurs in a greedy manner. In `PrepareProposal`, the `AuctionMempool` will iterate from largest to smallest bidding transaction until it finds the first valid `AuctionTx`. This means that all other bids will be rolled over into the auction in the next block unless if they specified a timeout height.

## Messages

### `MsgAuctionBid`

As mentioned, the `MsgAuctionBid` sdk message is the gateway for participating in the auction. The message handler for `MsgAuctionBid` will be called at most one time per block (for the winning auction transaction). The message handler will verify bid one last time and then distribute the bid to the auction house (escrow address) and proposer of the block where the auction transaction was included.

```protobuf
// MsgAuctionBid defines a request type for sending bids to the x/builder
// module.
message MsgAuctionBid {
  option (cosmos.msg.v1.signer) = "bidder";
  option (amino.name) = "pob/x/builder/MsgAuctionBid";

  option (gogoproto.equal) = false;

  // bidder is the address of the account that is submitting a bid to the
  // auction.
  string bidder = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
  // bid is the amount of coins that the bidder is bidding to participate in the
  // auction.
  cosmos.base.v1beta1.Coin bid = 3
      [ (gogoproto.nullable) = false, (amino.dont_omitempty) = true ];
  // transactions are the bytes of the transactions that the bidder wants to
  // bundle together.
  repeated bytes transactions = 4;
}
```

*Note: User's must also specify a timeout height when creating the transaction otherwise it will always fail `CheckTx`.*

### `MsgUpdateParams`

The `MsgUpdateParams` message can be executed only by an authority address that is defined by the application. This will typically be the address of the governance module account. This message handler is responsible for updating the configuration of the auction and `x/builder` module.

```protobuf
// MsgUpdateParams defines a request type for updating the x/builder module
// parameters.
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  option (amino.name) = "pob/x/builder/MsgUpdateParams";

  option (gogoproto.equal) = false;

  // authority is the address of the account that is authorized to update the
  // x/builder module parameters.
  string authority = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
  // params is the new parameters for the x/builder module.
  Params params = 2 [ (gogoproto.nullable) = false ];
}
```

## Clients

### CLI

#### CLI Queries

Information about how the `x/builder` module can be queried using the CLI.

| Command | Subcommand | Description |
| --- | --- | --- |
| query builder | `params` | Retrieve the parameters for the `x/builder` module |

### gRPC & REST

#### Queries

Information about how the `x/builder` module can be queried using gRPC and REST endpoints.

| Verb | Method | Description |
| --- | --- | --- |
| gRPC | skipmev.pob.builder.v1.Query/Params | Retrieve the parameters for the `x/builder` module |
| REST | /pob/builder/v1/params | Retrieve the parameters for the `x/builder` module |

#### Transactions

Information about how the `x/builder` module can be interacted with using gRPC and REST endpoints.

| Verb | Method | Description |
| --- | --- | --- |
| gRPC | skipmev.pob.builder.v1.Msg/AuctionBid | Submit bids to the auction to be considered for top of block execution |
| REST | /pob/builder/v1/bid | Submit bids to the auction to be considered for top of block execution |

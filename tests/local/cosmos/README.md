# Test Script for POB

This file contains a script that developers can execute
after spinning up a localnet to test the auction module.
The script will execute a series of transactions that
will test the auction module. The script will

  1. Initialize accounts with a balance to allow for
     transactions to be sent.
  2. Create a series of transactions that will test
     the auction module.
  3. Print out the results of the transactions and pseudo
     test cases.

> NOTE: THIS SCRIPT IS NOT MEANT TO BE RUN IN PRODUCTION
AND IS NOT A REPLACEMENT FOR UNIT or E2E TESTS.

## Running the script

1. Create a wallet you can retrieve the private key from. 
     You can do this by spinning up a localnet and creating
     a wallet with `<binarynamed> keys add <wallet name>`
2. Provide the wallet some balance in the genesis file (or
     send it some balance after spinning up the localnet)
3. Update the CONFIG variable below with the appropriate
     values. In particular,

    * update the private key to be your key. Since this is NOT a production 
    script, it is okay to hardcode the private key into the DefaultConfig 
    function below. 
    * update the CosmosRPCURL to be the URL of the Cosmos RPC endpoint.
    * etc.
4. Run the script with `go run main.go`. The script will print out the results
     of the transactions and pseudo test cases.
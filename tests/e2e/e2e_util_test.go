package e2e

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientContext creates a client.Context for use in integration tests.
// Note, it assumes all queries and broadcasts go to the first node.
func (s *IntegrationTestSuite) createClientContext() client.Context {
	node := s.valResources[0]

	rpcURI := node.GetHostPort("26657/tcp")
	gRPCURI := node.GetHostPort("9090/tcp")

	rpcClient, err := client.NewClientFromNode(rpcURI)
	s.Require().NoError(err)

	grpcClient, err := grpc.Dial(gRPCURI, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	s.Require().NoError(err)

	return client.Context{}.
		WithNodeURI(rpcURI).
		WithClient(rpcClient).
		WithGRPCClient(grpcClient).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithCodec(encodingConfig.Codec).
		WithChainID(s.chain.id)
}

// waitForBlockHeight will wait until the current block height is greater than or equal to the given height.
func (s *IntegrationTestSuite) waitForBlockHeight(height int64) {
	s.Require().Eventually(
		func() bool {
			return s.queryCurrentHeight() >= height
		},
		10*time.Second,
		500*time.Millisecond,
	)
}

// waitForABlock will wait until the current block height has increased by a single block.
func (s *IntegrationTestSuite) waitForABlock() int64 {
	height := s.queryCurrentHeight()
	s.waitForBlockHeight(height + 1)
	return height + 1
}

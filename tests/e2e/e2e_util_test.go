package e2e

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/ory/dockertest/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientContext creates a client.Context for use in integration tests.
// Note, it assumes all queries and broadcasts go to the first node.
func (s *IntegrationTestSuite) createClientContext(node *dockertest.Resource) client.Context {
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
		WithChainID(s.chain.id).
		WithBroadcastMode("BROADCAST_MODE_SYNC")
}

package e2e

import (
	"github.com/cosmos/cosmos-sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientContext creates a client.Context for use in integration tests.
// Note, it assumes all queries and broadcasts go to the first node.
func (s *IntegrationTestSuite) createClientContext() client.Context {
	rpcURI := "tcp://localhost:26657"
	gRPCURI := "localhost:9090"

	rpcClient, err := client.NewClientFromNode(rpcURI)
	s.Require().NoError(err)

	grpcClient, err := grpc.Dial(gRPCURI, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	s.Require().NoError(err)

	return client.Context{}.
		WithNodeURI(rpcURI).
		WithClient(rpcClient).
		WithGRPCClient(grpcClient).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithCodec(encodingConfig.Codec)
}

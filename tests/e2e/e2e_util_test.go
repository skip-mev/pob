package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientContext creates a client.Context for use in integration tests.
// Note, it assumes all queries and broadcasts go to the first node.
func (s *IntegrationTestSuite) createClientContext() client.Context {
	node := s.valResources[0]

	rpcURI := node.GetHostPort("26657/tcp")
	gRPCURI := node.GetHostPort("9090/tcp")

	fmt.Println(rpcURI, gRPCURI)

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
		WithBroadcastMode(flags.BroadcastSync)
}

// waitForBlockHe6ight will wait until the current block height is greater than or equal to the given height.
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

// ExecCmd executes command by running it on the node container (specified by containerName)
// success is the output of the command that needs to be observed for the command to be deemed successful.
// It is found by checking if stdout or stderr contains the success string anywhere within it.
// returns container std out, container std err, and error if any.
// An error is returned if the command fails to execute or if the success string is not found in the output.
func (s *IntegrationTestSuite) ExecCmd(t *testing.T, containerName string, command []string, success string) (bytes.Buffer, bytes.Buffer, error) {
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// We use the `require.Eventually` function because it is only allowed to do one transaction per block without
	// sequence numbers. For simplicity, we avoid keeping track of the sequence number and just use the `require.Eventually`.
	require.Eventually(
		t,
		func() bool {
			exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
				Context:      ctx,
				AttachStdout: true,
				AttachStderr: true,
				Container:    "node0",
				User:         "root",
				Cmd:          command,
			})
			require.NoError(t, err)

			err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
				Context:      ctx,
				Detach:       false,
				OutputStream: &outBuf,
				ErrorStream:  &errBuf,
			})
			if err != nil {
				return false
			}

			errBufString := errBuf.String()

			if success != "" {
				return strings.Contains(outBuf.String(), success) || strings.Contains(errBufString, success)
			}

			return true
		},
		time.Minute,
		50*time.Millisecond,
		fmt.Sprintf("success condition (%s) was not met.\nstdout:\n %s\nstderr:\n %s\n",
			success, outBuf.String(), errBuf.String()),
	)

	return outBuf, errBuf, nil
}

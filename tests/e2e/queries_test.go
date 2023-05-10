package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/skip-mev/pob/x/builder/types"
	"google.golang.org/grpc"
)

// queryTopOfBlockParams queries the top of block params from the given node.
func (s *IntegrationTestSuite) queryTopOfBlockParams(node *dockertest.Resource) (*types.Params, error) {
	grpcConn := s.getGRPCClient(node)
	queryClient := types.NewQueryClient(grpcConn)

	resp, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
	s.Require().NoError(err)

	return &resp.Params, nil
}

func (s *IntegrationTestSuite) getGRPCClient(node *dockertest.Resource) *grpc.ClientConn {
	url := node.GetHostPort("1317/tcp")
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	s.Require().NoError(err)

	return conn
}

func (s *IntegrationTestSuite) QueryGRPCGateway(node *dockertest.Resource, path string, parameters ...string) ([]byte, error) {
	if len(parameters)%2 != 0 {
		return nil, fmt.Errorf("invalid number of parameters, must follow the format of key + value")
	}

	// add the URL for the given validator ID, and pre-pend to to path.
	hostPort := node.GetHostPort("1317/tcp")
	endpoint := fmt.Sprintf("%s", hostPort)
	fullQueryPath := fmt.Sprintf("%s/%s", endpoint, path)

	var resp *http.Response
	s.Require().Eventually(func() bool {
		req, err := http.NewRequest("GET", fullQueryPath, nil)
		if err != nil {
			return false
		}

		if len(parameters) > 0 {
			q := req.URL.Query()
			for i := 0; i < len(parameters); i += 2 {
				q.Add(parameters[i], parameters[i+1])
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			s.T().Logf("error while executing HTTP request: %s", err.Error())
			return false
		}

		return resp.StatusCode != http.StatusServiceUnavailable
	}, time.Minute, time.Millisecond*10, "failed to execute HTTP request")

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bz))
	}
	return bz, nil
}

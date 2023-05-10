//go:build e2e

package e2e

import (
	"context"

	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

func (s *IntegrationTestSuite) TestGetBuilderParams() {
	queryClient := buildertypes.NewQueryClient(s.createClientContext())

	request := &buildertypes.QueryParamsRequest{}
	response, err := queryClient.Params(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Equal(buildertypes.DefaultParams(), response.Params)
}

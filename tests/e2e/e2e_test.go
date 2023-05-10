//go:build e2e

package e2e

import (
	"fmt"
	"time"
)

func (s *IntegrationTestSuite) TestTmp() {
	s.Require().Eventually(func() bool {
		params, err := s.queryTopOfBlockParams(s.valResources[0])
		s.Require().NoError(err)
		s.Require().NotNil(params)

		fmt.Println(params)

		return true
	},
		5*time.Minute,
		3*time.Second,
	)

	s.Require().True(true)
}

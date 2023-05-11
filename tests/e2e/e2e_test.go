package e2e

func (s *IntegrationTestSuite) TestTmp() {
	s.Require().True(true)
}

func (s *IntegrationTestSuite) TestGetBuilderParams() {
	params := s.queryBuilderParams(s.valResources[0])
	s.Require().NotNil(params)
}

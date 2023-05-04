package e2e

import (
	"fmt"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	tmpDirs      []string
	chain        *chain
	dkrPool      *dockertest.Pool
	dkrNet       *dockertest.Network
	valResources []*dockertest.Resource
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")

	var err error
	s.chain, err = newChain()
	s.Require().NoError(err)

	s.T().Logf("starting e2e infrastructure; chain-id: %s; datadir: %s", s.chain.id, s.chain.dataDir)

	s.dkrPool, err = dockertest.NewPool("")
	s.Require().NoError(err)

	s.dkrNet, err = s.dkrPool.CreateNetwork(fmt.Sprintf("%s-testnet", s.chain.id))
	s.Require().NoError(err)

	// The bootstrapping phase is as follows:
	//
	// 1. Initialize TestApp validator nodes.
	// 2. Create and initialize TestApp validator genesis files (setting delegate keys for validators).
	// 3. Start TestApp network.
	s.initNodes()
	s.initGenesis()
	s.initValidatorConfigs()
	s.runValidators()
	s.initTestAppClient()
}

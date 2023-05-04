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

	s.T().Logf("Ethereum and peggo are disable due to Ethereum PoS migration and PoW fork")
	// // var useGanache bool
	// // if str := os.Getenv("UMEE_E2E_USE_GANACHE"); len(str) > 0 {
	// // 	useGanache, err = strconv.ParseBool(str)
	// // 	s.Require().NoError(err)
	// // }

	// // The bootstrapping phase is as follows:
	// //
	// // 1. Initialize Umee validator nodes.
	// // 2. Launch an Ethereum container that mines.
	// // 3. Create and initialize Umee validator genesis files (setting delegate keys for validators).
	// // 4. Start Umee network.
	// // 5. Run an Oracle price feeder.
	// // 6. Create and run Gaia container(s).
	// // 7. Create and run IBC relayer (Hermes) containers.
	// // 8. Deploy the Gravity Bridge contract.
	// // 9. Create and start Peggo (orchestrator) containers.
	// s.initNodes()
	// // if useGanache {
	// // 	s.runGanacheContainer()
	// // } else {
	// // 	s.initEthereum()
	// // 	s.runEthContainer()
	// // }
	// s.initGenesis()
	// s.initValidatorConfigs()
	// s.runValidators()
	// s.runPriceFeeder()
	// s.runGaiaNetwork()
	// s.runIBCRelayer()
	// // s.runContractDeployment()
	// // s.runOrchestrators()
	// s.initUmeeClient()
}

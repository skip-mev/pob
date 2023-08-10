package constructor_test

import (
	"math/rand"
	"testing"

	testutils "github.com/skip-mev/pob/testutils"
	"github.com/stretchr/testify/suite"
)

type ConstructorTestSuite struct {
	suite.Suite

	encodingConfig testutils.EncodingConfig
	random         *rand.Rand
	accounts       []testutils.Account
	gasTokenDenom  string
}

func TestConstructorTestSuite(t *testing.T) {
	suite.Run(t, new(ConstructorTestSuite))
}

func (s *ConstructorTestSuite) SetupTest() {
	// Set up basic TX encoding config.
	s.encodingConfig = testutils.CreateTestEncodingConfig()

	// Create a few random accounts
	s.random = rand.New(rand.NewSource(1))
	s.accounts = testutils.RandomAccounts(s.random, 5)
	s.gasTokenDenom = "stake"
}

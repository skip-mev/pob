package types

const (
	// ModuleName is the name of the builder module
	ModuleName = "builder"

	// StoreKey is the default store key for the builder module
	StoreKey = ModuleName

	// RouterKey is the message route for the builder module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the builder module
	QuerierRoute = ModuleName
)

const (
	prefixParams = iota + 1
	prefixIsCheckVoteExtensionTx
)

var (
	// KeyParams is the store key for the builder module's parameters.
	KeyParams = []byte{prefixParams}

	// KeyIsCheckVoteExtensionTx is the store key the builder module toggles when vote extensions are checked.
	KeyIsCheckVoteExtensionTx = []byte{prefixIsCheckVoteExtensionTx}
)

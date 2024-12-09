//go:build cgo

package keeper

import (
	"path/filepath"

	"cosmossdk.io/collections"
	corestoretypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	wasmvm "github.com/CosmWasm/wasmd/wasmvm/v2"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// NewKeeper creates a new contract Keeper instance
// If customEncoders is non-nil, we can use this to override some of the message handler, especially custom
func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	distrKeeper types.DistributionKeeper,
	ics4Wrapper types.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	capabilityKeeper types.CapabilityKeeper,
	portSource types.ICS20TransferPortSource,
	router MessageRouter,
	_ GRPCQueryRouter,
	homeDir string,
	wasmConfig types.WasmConfig,
	availableCapabilities []string,
	authority string,
	opts ...Option,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	keeper := &Keeper{
		storeService:         storeService,
		cdc:                  cdc,
		wasmVM:               nil,
		accountKeeper:        accountKeeper,
		bank:                 NewBankCoinTransferrer(bankKeeper),
		accountPruner:        NewVestingCoinBurner(bankKeeper),
		portKeeper:           portKeeper,
		capabilityKeeper:     capabilityKeeper,
		queryGasLimit:        wasmConfig.SmartQueryGasLimit,
		gasRegister:          types.NewDefaultWasmGasRegister(),
		maxQueryStackSize:    types.DefaultMaxQueryStackSize,
		maxCallDepth:         types.DefaultMaxCallDepth,
		acceptedAccountTypes: defaultAcceptedAccountTypes,
		params:               collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		propagateGovAuthorization: map[types.AuthorizationPolicyAction]struct{}{
			types.AuthZActionInstantiate: {},
		},
		authority: authority,
	}
	keeper.messenger = NewDefaultMessageHandler(keeper, router, ics4Wrapper, channelKeeper, capabilityKeeper, bankKeeper, cdc, portSource)
	keeper.wasmVMQueryHandler = DefaultQueryPlugins(bankKeeper, stakingKeeper, distrKeeper, channelKeeper, keeper)
	preOpts, postOpts := splitOpts(opts)
	for _, o := range preOpts {
		o.apply(keeper)
	}
	// always wrap the messenger, even if it was replaced by an option
	keeper.messenger = callDepthMessageHandler{keeper.messenger, keeper.maxCallDepth}
	// only set the wasmvm if no one set this in the options
	// NewVM does a lot, so better not to create it and silently drop it.
	if keeper.wasmVM == nil {
		var err error
		keeper.wasmVM, err = wasmvm.NewVM(filepath.Join(homeDir, "wasm"), availableCapabilities, contractMemoryLimit, wasmConfig.ContractDebugMode, wasmConfig.MemoryCacheSize)
		if err != nil {
			panic(err)
		}
	}

	for _, o := range postOpts {
		o.apply(keeper)
	}
	// not updatable, yet
	keeper.wasmVMResponseHandler = NewDefaultWasmVMContractResponseHandler(NewMessageDispatcher(keeper.messenger, keeper))
	return *keeper
}

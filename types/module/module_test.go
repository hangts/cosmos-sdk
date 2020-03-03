package module_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/tests/mocks"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var errFoo = errors.New("dummy")

func TestBasicManager(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	mockAppModuleBasic1 := mocks.NewMockAppModuleBasic(mockCtrl)
	mockAppModuleBasic2 := mocks.NewMockAppModuleBasic(mockCtrl)

	mockAppModuleBasic1.EXPECT().Name().Times(1).Return("mockAppModuleBasic1")
	mockAppModuleBasic2.EXPECT().Name().Times(1).Return("mockAppModuleBasic2")

	mm := module.NewBasicManager(mockAppModuleBasic1, mockAppModuleBasic2)
	require.Equal(t, mm["mockAppModuleBasic1"], mockAppModuleBasic1)

	cdc := codec.New()
	mockAppModuleBasic1.EXPECT().RegisterCodec(gomock.Eq(cdc)).Times(1)
	mockAppModuleBasic2.EXPECT().RegisterCodec(gomock.Eq(cdc)).Times(1)
	mm.RegisterCodec(cdc)

	mockAppModuleBasic1.EXPECT().Name().Times(1).Return("mockAppModuleBasic1")
	mockAppModuleBasic2.EXPECT().Name().Times(1).Return("mockAppModuleBasic2")
	mockAppModuleBasic1.EXPECT().DefaultGenesis(gomock.Eq(cdc)).Times(1).Return(json.RawMessage(``))
	mockAppModuleBasic2.EXPECT().DefaultGenesis(gomock.Eq(cdc)).Times(1).Return(json.RawMessage(`{"key":"value"}`))
	defaultGenesis := mm.DefaultGenesis(cdc)
	require.Equal(t, 2, len(defaultGenesis))
	require.Equal(t, json.RawMessage(``), defaultGenesis["mockAppModuleBasic1"])

	var data map[string]string
	require.NoError(t, json.Unmarshal(defaultGenesis["mockAppModuleBasic2"], &data))
	require.Equal(t, map[string]string{"key": "value"}, data)

	mockAppModuleBasic1.EXPECT().Name().Times(1).Return("mockAppModuleBasic1")
	mockAppModuleBasic1.EXPECT().ValidateGenesis(gomock.Eq(cdc), gomock.Eq(defaultGenesis["mockAppModuleBasic1"])).Times(1).Return(errFoo)
	require.True(t, errors.Is(errFoo, mm.ValidateGenesis(cdc, defaultGenesis)))

	mockAppModuleBasic1.EXPECT().RegisterRESTRoutes(gomock.Eq(context.CLIContext{}), gomock.Eq(&mux.Router{})).Times(1)
	mockAppModuleBasic2.EXPECT().RegisterRESTRoutes(gomock.Eq(context.CLIContext{}), gomock.Eq(&mux.Router{})).Times(1)
	mm.RegisterRESTRoutes(context.CLIContext{}, &mux.Router{})

	mockCmd := &cobra.Command{Use: "root"}
	mockAppModuleBasic1.EXPECT().GetTxCmd(cdc).Times(1).Return(nil)
	mockAppModuleBasic2.EXPECT().GetTxCmd(cdc).Times(1).Return(&cobra.Command{})
	mm.AddTxCommands(mockCmd, cdc)

	mockAppModuleBasic1.EXPECT().GetQueryCmd(cdc).Times(1).Return(nil)
	mockAppModuleBasic2.EXPECT().GetQueryCmd(cdc).Times(1).Return(&cobra.Command{})
	mm.AddQueryCommands(mockCmd, cdc)
}

func TestGenesisOnlyAppModule(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	mockModule := mocks.NewMockAppModuleGenesis(mockCtrl)
	mockInvariantRegistry := mocks.NewMockInvariantRegistry(mockCtrl)
	goam := module.NewGenesisOnlyAppModule(mockModule)

	require.Empty(t, goam.Route())
	require.Empty(t, goam.QuerierRoute())
	require.Nil(t, goam.NewHandler())
	require.Nil(t, goam.NewQuerierHandler())

	// no-op
	goam.RegisterInvariants(mockInvariantRegistry)
	goam.BeginBlock(sdk.Context{}, abci.RequestBeginBlock{})
	require.Equal(t, []abci.ValidatorUpdate{}, goam.EndBlock(sdk.Context{}, abci.RequestEndBlock{}))
}

func TestManagerOrderSetters(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)
	mockAppModule1 := mocks.NewMockAppModule(mockCtrl)
	mockAppModule2 := mocks.NewMockAppModule(mockCtrl)

	mockAppModule1.EXPECT().Name().Times(2).Return("module1")
	mockAppModule2.EXPECT().Name().Times(2).Return("module2")
	mm := module.NewManager(mockAppModule1, mockAppModule2)
	require.NotNil(t, mm)
	require.Equal(t, 2, len(mm.Modules))

	require.Equal(t, []string{"module1", "module2"}, mm.OrderInitGenesis)
	mm.SetOrderInitGenesis("module2", "module1")
	require.Equal(t, []string{"module2", "module1"}, mm.OrderInitGenesis)

	require.Equal(t, []string{"module1", "module2"}, mm.OrderExportGenesis)
	mm.SetOrderExportGenesis("module2", "module1")
	require.Equal(t, []string{"module2", "module1"}, mm.OrderExportGenesis)

	require.Equal(t, []string{"module1", "module2"}, mm.OrderBeginBlockers)
	mm.SetOrderBeginBlockers("module2", "module1")
	require.Equal(t, []string{"module2", "module1"}, mm.OrderBeginBlockers)

	require.Equal(t, []string{"module1", "module2"}, mm.OrderEndBlockers)
	mm.SetOrderEndBlockers("module2", "module1")
	require.Equal(t, []string{"module2", "module1"}, mm.OrderEndBlockers)
}

func TestManager_RegisterInvariants(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	mockAppModule1 := mocks.NewMockAppModule(mockCtrl)
	mockAppModule2 := mocks.NewMockAppModule(mockCtrl)
	mockAppModule1.EXPECT().Name().Times(2).Return("module1")
	mockAppModule2.EXPECT().Name().Times(2).Return("module2")
	mm := module.NewManager(mockAppModule1, mockAppModule2)
	require.NotNil(t, mm)
	require.Equal(t, 2, len(mm.Modules))

	// test RegisterInvariants
	mockInvariantRegistry := mocks.NewMockInvariantRegistry(mockCtrl)
	mockAppModule1.EXPECT().RegisterInvariants(gomock.Eq(mockInvariantRegistry)).Times(1)
	mockAppModule2.EXPECT().RegisterInvariants(gomock.Eq(mockInvariantRegistry)).Times(1)
	mm.RegisterInvariants(mockInvariantRegistry)
}

func TestManager_RegisterRoutes(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	mockAppModule1 := mocks.NewMockAppModule(mockCtrl)
	mockAppModule2 := mocks.NewMockAppModule(mockCtrl)
	mockAppModule1.EXPECT().Name().Times(2).Return("module1")
	mockAppModule2.EXPECT().Name().Times(2).Return("module2")
	mm := module.NewManager(mockAppModule1, mockAppModule2)
	require.NotNil(t, mm)
	require.Equal(t, 2, len(mm.Modules))

	router := sdk.Router()
}

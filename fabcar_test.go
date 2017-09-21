package fabcar

import (
	"testing"

	"fmt"
	"io/ioutil"
	"path/filepath"

	ca "github.com/hyperledger/fabric-sdk-go/api/apifabca"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	deffab "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi/opt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn"
	"github.com/stretchr/testify/suite"
)

type FabcarTestSuite struct {
	suite.Suite
	org              string
	client           fab.FabricClient
	channel          fab.Channel
	orderer          fab.Orderer
	adminUser        ca.User
	ordererAdminUser ca.User
	user             ca.User
	chaincodeID      string
	eventHub         fab.EventHub
}

func (suite *FabcarTestSuite) SetupSuite() {
	require := suite.Require()
	suite.org = "org1"
	suite.chaincodeID = "fabcar"
	sdkOptions := deffab.Options{
		ConfigFile: "config_test.yaml",
		StateStoreOpts: opt.StateStoreOpts{
			Path: "/tmp/enroll_user",
		},
	}
	sdk, err := deffab.NewSDK(sdkOptions)
	require.NoError(err)
	ctx, err := sdk.NewContext(suite.org)
	require.NoError(err)
	user, err := deffab.NewUser(sdk.ConfigProvider(), ctx.MSPClient(), "admin", "adminpw", suite.org)
	require.NoError(err)
	session, err := sdk.NewSession(ctx, user)
	require.NoError(err)
	sc, err := sdk.NewSystemClient(session)
	require.NoError(err)
	err = sc.SaveUserToStateStore(user, false)
	require.NoError(err)
	suite.client = sc
	suite.adminUser, err = GetAdmin(sc, "org1", suite.org)
	require.NoError(err)
	suite.ordererAdminUser, err = GetOrdererAdmin(sc, suite.org)
	require.NoError(err)
	suite.user, err = GetUser(sc, "org1", suite.org)
	require.NoError(err)
	//by default client's user context should use regular user, for admin actions, UserContext must be set to AdminUser
	sc.SetUserContext(suite.user)
	ordererConfig, err := sc.Config().RandomOrdererConfig()
	require.NoError(err)
	serverHostOverride := ""
	if str, ok := ordererConfig.GrpcOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}
	suite.channel, err = sc.NewChannel("mychannel")
	require.NoError(err)
	suite.orderer, err = orderer.NewOrderer(ordererConfig.URL, ordererConfig.TlsCACerts.Path,
		serverHostOverride, sc.Config())
	require.NoError(err)
	err = suite.channel.AddOrderer(suite.orderer)
	require.NoError(err)
	peers, err := sc.Config().PeersConfig(suite.org)
	require.NoError(err)
	for _, p := range peers {
		serverHostOverride = ""
		if str, ok := p.GrpcOptions["ssl-target-name-override"].(string); ok {
			serverHostOverride = str
		}
		endorser, err := deffab.NewPeer(p.Url, p.TlsCACerts.Path, serverHostOverride, sc.Config())
		require.NoError(err)
		err = suite.channel.AddPeer(endorser)
		require.NoError(err)
	}

	foundEventHub := false
	eventHub, err := events.NewEventHub(sc)
	for _, p := range peers {
		if p.Url != "" {
			serverHostOverride = ""
			if str, ok := p.GrpcOptions["ssl-target-name-override"].(string); ok {
				serverHostOverride = str
			}
			eventHub.SetPeerAddr(p.EventUrl, p.TlsCACerts.Path, serverHostOverride)
			foundEventHub = true
			break
		}
	}
	if !foundEventHub {
		require.FailNow("No EventHub configuration found")
	}
	suite.eventHub = eventHub

}

func (suite *FabcarTestSuite) TestQueryAllCars() {
	require := suite.Require()
	fcn := "queryAllCars"
	result, err := fabrictxn.QueryChaincode(suite.client, suite.channel,
		suite.chaincodeID, fcn, []string{})
	require.NoError(err)
	fmt.Println(result)
}

func (suite *FabcarTestSuite) TestQueryCar() {
	require := suite.Require()
	fcn := "queryCar"
	result, err := fabrictxn.QueryChaincode(suite.client, suite.channel,
		suite.chaincodeID, fcn, []string{"CAR4"})
	require.NoError(err)
	fmt.Println(result)
}

func (suite *FabcarTestSuite) TestCreateCar() {
	fcn := "createCar"
	args := []string{"CAR11", "Honda", "Accord", "Black", "Tom"}
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in create car...")
	txnID, err := fabrictxn.InvokeChaincode(suite.client, suite.channel, []apitxn.ProposalProcessor{suite.channel.PrimaryPeer()}, suite.eventHub, suite.chaincodeID, fcn, args, transientDataMap)
	suite.Require().NoError(err)
	fmt.Println(txnID.ID)
}

func (suite *FabcarTestSuite) TestChangeCarOwner() {
	fcn := "changeCarOwner"
	args := []string{"CAR10", "Barry"}
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in create car...")
	txnID, err := fabrictxn.InvokeChaincode(suite.client, suite.channel, []apitxn.ProposalProcessor{suite.channel.PrimaryPeer()}, suite.eventHub, suite.chaincodeID, fcn, args, transientDataMap)
	suite.Require().NoError(err)
	fmt.Println(txnID.ID)
}

func TestFabcar(t *testing.T) {
	suite.Run(t, new(FabcarTestSuite))
}

// GetOrdererAdmin returns a pre-enrolled orderer admin user
func GetOrdererAdmin(c fab.FabricClient, orgName string) (ca.User, error) {
	keyDir := "ordererOrganizations/example.com/users/Admin@example.com/msp/keystore"
	certDir := "ordererOrganizations/example.com/users/Admin@example.com/msp/signcerts"
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, "ordererAdmin", orgName)
}

// GetAdmin returns a pre-enrolled org admin user
func GetAdmin(c fab.FabricClient, orgPath string, orgName string) (ca.User, error) {
	keyDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/Admin@%s.example.com/msp/keystore", orgPath, orgPath)
	certDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/Admin@%s.example.com/msp/signcerts", orgPath, orgPath)
	username := fmt.Sprintf("peer%sAdmin", orgPath)
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, username, orgName)
}

// GetUser returns a pre-enrolled org user
func GetUser(c fab.FabricClient, orgPath string, orgName string) (ca.User, error) {
	keyDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/User1@%s.example.com/msp/keystore", orgPath, orgPath)
	certDir := fmt.Sprintf("peerOrganizations/%s.example.com/users/User1@%s.example.com/msp/signcerts", orgPath, orgPath)
	username := fmt.Sprintf("peer%sUser1", orgPath)
	return getDefaultImplPreEnrolledUser(c, keyDir, certDir, username, orgName)
}

// GetDefaultImplPreEnrolledUser ...
func getDefaultImplPreEnrolledUser(client fab.FabricClient, keyDir string, certDir string, username string, orgName string) (ca.User, error) {
	privateKeyDir := filepath.Join(client.Config().CryptoConfigPath(), keyDir)
	privateKeyPath, err := getFirstPathFromDir(privateKeyDir)
	if err != nil {
		return nil, fmt.Errorf("Error finding the private key path: %v", err)
	}

	enrollmentCertDir := filepath.Join(client.Config().CryptoConfigPath(), certDir)
	enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)
	if err != nil {
		return nil, fmt.Errorf("Error finding the enrollment cert path: %v", err)
	}
	mspID, err := client.Config().MspID(orgName)
	if err != nil {
		return nil, fmt.Errorf("Error reading MSP ID config: %s", err)
	}
	return deffab.NewPreEnrolledUser(client.Config(), privateKeyPath, enrollmentCertPath, username, mspID, client.CryptoSuite())
}

// Gets the first path from the dir directory
func getFirstPathFromDir(dir string) (string, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("Could not read directory %s, err %s", err, dir)
	}

	for _, p := range files {
		if p.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), p.Name())
		fmt.Printf("Reading file %s\n", fullName)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), f.Name())
		return fullName, nil
	}

	return "", fmt.Errorf("No paths found in directory: %s", dir)
}

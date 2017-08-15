package main

import (
	"testing"

	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"time"

	ca "github.com/hyperledger/fabric-sdk-go/api/apifabca"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	deffab "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi/opt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/suite"
)

type FabcarTestSuite struct {
	suite.Suite
	org              string
	client           fab.FabricClient
	channel          fab.Channel
	orderer          fab.Orderer
	peer             fab.Peer
	adminUser        ca.User
	ordererAdminUser ca.User
	user             ca.User
	chaincodeID      string
	eventHub         fab.EventHub
}

func (suite *FabcarTestSuite) SetupSuite() {
	require := suite.Require()
	suite.org = "peerorg1"
	suite.chaincodeID = "fabcar"
	sdkOptions := deffab.Options{
		ConfigFile: "config_test.yaml",
		StateStoreOpts: opt.StateStoreOpts{
			Path: "/tmp/enroll_user",
		},
	}
	sdk, err := deffab.NewSDK(sdkOptions)
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
	suite.client.SetUserContext(suite.adminUser)
	suite.ordererAdminUser, err = GetOrdererAdmin(sc, suite.org)
	require.NoError(err)
	suite.user, err = GetUser(sc, "org1", suite.org)
	require.NoError(err)
	ordererConfig, err := sc.Config().RandomOrdererConfig()
	require.NoError(err)
	suite.channel, err = sc.NewChannel("mychannel")
	require.NoError(err)
	suite.orderer, err = orderer.NewOrderer(fmt.Sprintf("%s:%d", ordererConfig.Host,
		ordererConfig.Port), ordererConfig.TLS.Certificate,
		ordererConfig.TLS.ServerHostOverride, sc.Config())
	require.NoError(err)
	err = suite.channel.AddOrderer(suite.orderer)
	require.NoError(err)
	peers, err := sc.Config().PeersConfig(suite.org)
	require.NoError(err)
	suite.peer, err = peer.NewPeerTLSFromCert(fmt.Sprintf("%s:%d", peers[0].Host,
		peers[0].Port), peers[0].TLS.Certificate,
		peers[0].TLS.ServerHostOverride, sc.Config()) //We have just one peer
	require.NoError(err)
	err = suite.channel.AddPeer(suite.peer)
	require.NoError(err)
	err = suite.channel.SetPrimaryPeer(suite.peer)
	require.NoError(err)
	eventHub, err := events.NewEventHub(sc)
	eventHub.SetPeerAddr(fmt.Sprintf("%s:%d", peers[0].EventHost, peers[0].EventPort),
		peers[0].TLS.Certificate, peers[0].TLS.ServerHostOverride)

}

//func getEventHub

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
	request := buildRequest(suite, "createCar", "CAR11", "Honda", "Accord", "Black", "Tom")
	processTxn(suite, request)
}
func processTxn(suite *FabcarTestSuite, request apitxn.ChaincodeInvokeRequest) {
	require := suite.Require()
	tprs, txnID, err := suite.channel.SendTransactionProposal(request)
	require.NoError(err)
	for _, v := range tprs {
		require.NoErrorf(v.Err, "invoke Endorser %s returned error: %v", v.Endorser, v.Err)
	}
	done, fail := registerEvent(suite, txnID)
	tx, err := suite.channel.CreateTransaction(tprs)
	require.NoError(err)
	tr, err := suite.channel.SendTransaction(tx)
	require.NoError(err)
	require.NoError(tr.Err, "Orderer %s return error: %v", tr.Orderer, tr.Err)
	fmt.Println(tr)
	select {
	case <-done:
	case err = <-fail:
		require.NoErrorf(err, "invoke Error received from eventhub for txid(%s) error(%v)", txnID, fail)
	case <-time.After(time.Second * 30):
		require.FailNow("invoke Didn't receive block event for txid(%s)", txnID)
	}
}
func registerEvent(suite *FabcarTestSuite, txnID apitxn.TransactionID) (chan bool, chan error) {
	require := suite.Require()
	done := make(chan bool)
	fail := make(chan error)
	err := suite.eventHub.Connect()
	require.NoError(err)
	suite.eventHub.RegisterTxEvent(txnID, func(txId string, errorCode pb.TxValidationCode, err error) {
		if err != nil {
			fmt.Printf("Received error event for txid(%s)\n", txId)
			fail <- err
		} else {
			fmt.Printf("Received success event for txid(%s)\n", txId)
			done <- true
		}
	})
	return done, fail
}
func buildRequest(suite *FabcarTestSuite, fcn string, args ...string) apitxn.ChaincodeInvokeRequest {
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in create car...")
	request := apitxn.ChaincodeInvokeRequest{
		Targets:      peer.PeersToTxnProcessors(suite.channel.Peers()),
		Fcn:          fcn,
		Args:         args,
		TransientMap: transientDataMap,
		ChaincodeID:  suite.chaincodeID,
	}
	return request
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

// GenerateRandomID generates random ID
func GenerateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	return randomString(10)
}

// Utility to create random string of strlen length
func randomString(strlen int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
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

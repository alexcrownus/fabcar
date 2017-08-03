package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	ca "github.com/hyperledger/fabric-sdk-go/api/apifabca"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	deffab "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	client "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn"
)

var org1 = "peerorg1"
var org2 = "peerorg2"

// Client
var testClient fab.FabricClient

// Channel
var orgTestChannel fab.Channel

// Orderers
var orgTestOrderer fab.Orderer

// Peers
var orgTestPeer0 fab.Peer
var orgTestPeer1 fab.Peer

// EventHubs
var peer0EventHub fab.EventHub
var peer1EventHub fab.EventHub

// Users
var org1AdminUser ca.User
var org2AdminUser ca.User
var ordererAdminUser ca.User
var org1User ca.User
var org2User ca.User

func main() {
	configImpl, err := config.InitConfig("config_test.yaml")
	panic(err)
	testClient = client.NewClient(configImpl)
	err = factory.InitFactories(configImpl.CSPConfig())
	panic(err)
	cryptoSuite := factory.GetDefault()
	testClient.SetCryptoSuite(cryptoSuite)
	s := configImpl.CryptoConfigPath()
	fmt.Println(s)
	ordererConfig, err := testClient.Config().RandomOrdererConfig()
	panic(err)
	orgTestOrderer, err = orderer.NewOrderer(fmt.Sprintf("%s:%d", ordererConfig.Host,
		ordererConfig.Port), ordererConfig.TLS.Certificate,
		ordererConfig.TLS.ServerHostOverride, testClient.Config())
	panic(err)
	fmt.Println(orgTestOrderer.URL())

	org1Peers, err := testClient.Config().PeersConfig(org1)
	panic(err)
	orgTestPeer0, err = peer.NewPeerTLSFromCert(fmt.Sprintf("%s:%d", org1Peers[0].Host,
		org1Peers[0].Port), org1Peers[0].TLS.Certificate,
		org1Peers[0].TLS.ServerHostOverride, testClient.Config())
	panic(err)
	org1AdminUser, err = GetAdmin(testClient, "org1", org1)
	panic(err)
	ordererAdminUser, err = GetOrdererAdmin(testClient, org1)
	panic(err)
	testClient.SetUserContext(org1AdminUser)
	orgTestChannel, err = channel.NewChannel("mychannel", testClient)
	panic(err)
	orgTestChannel.SetPrimaryPeer(orgTestPeer0)
	orgTestChannel.AddPeer(orgTestPeer0)
	fcn := "queryAllCars"
	result, err := fabrictxn.QueryChaincode(testClient, orgTestChannel,
		"fabcar", fcn, []string{})
	panic(err)
	fmt.Println(result)
}

func panic(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

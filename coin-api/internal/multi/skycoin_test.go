package multi

import (
	"testing"

	"github.com/skycoin/skycoin/src/visor"

	mocklib "github.com/stretchr/testify/mock"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/testutil"

	"github.com/skycoin/services/coin-api/internal/multi/mock"
	"github.com/skycoin/skycoin/src/wallet"
)

const (
	rawAddress = "2GgFvqoyk9RjwVzj8tqfcXVXB4orBwoc9qv"
	rawTxID    = "bff13a47a98402ecf2d2eee40464959ad26e0ed6047de5709ffb0c0c9fc1fca5"
	rawTxStr   = "dc00000000a8558b814926ed0062cd720a572bd67367aa0d01c0769ea4800adcc89cdee524010000008756e4bde4ee1c725510a6a9a308c6a90d949de7785978599a87faba601d119f27e1be695cbb32a1e346e5dd88653a97006bf1a93c9673ac59cf7b5db7e07901000100000079216473e8f2c17095c6887cc9edca6c023afedfac2e0c5460e8b6f359684f8b020000000060dfa95881cdc827b45a6d49b11dbc152ecd4de640420f00000000000000000000000000006409744bcacb181bf98b1f02a11e112d7e4fa9f940f1f23a000000000000000000000000"
)

// clientMock - its a mock of client web rpc API but be careful if you want laucnh tests in parallel, you may get race on
// the package-level variables and in this case you'd better off moving this variable somewhere to the getTestedService() function
var clientMock *mock.GuiClientMock

func TestGenerateKeyPair(t *testing.T) {
	skyService := getTestedMockedService()
	keysResponse := skyService.GenerateKeyPair()
	if len(keysResponse.Private) == 0 || len(keysResponse.Public) == 0 {
		t.Fatalf("keysResponse.Private or keysResponse.Public should not be zero length")
	}

	t.Run("TestGenerateAddress", func(t *testing.T) {
		rspAdd, err := skyService.GenerateAddr(keysResponse.Private)
		if err != nil {
			t.Fatal(err.Error())
		}
		if len(rspAdd.Address) == 0 {
			t.Fatalf("rawAddress cannot be zero lenght")
		}

		t.Run("check balance", func(t *testing.T) {
			address := rspAdd.Address
			skyServiceIsolated := getTestedMockedService()

			var (
				expectedCoins uint64 = 10
				expectedHours uint64 = 1
			)

			checkBalance := func(client ClientApi, addresses []string) (*wallet.BalancePair, error) {
				return &wallet.BalancePair{
					Confirmed: wallet.Balance{
						Coins: expectedCoins,
						Hours: expectedHours,
					},
				}, nil
			}

			skyServiceIsolated.checkBalance = checkBalance

			balanceResponse, err := skyServiceIsolated.CheckBalance(address)

			if err != nil {
				t.Fatal(err)
			}
			if balanceResponse.Balance != expectedCoins {
				t.Fatalf("Wrong balance expected %d actual %d", expectedCoins, balanceResponse.Balance)
			}

			if balanceResponse.Balance != expectedCoins {
				t.Fatalf("Wrong hours expected %d actual %d", expectedHours, balanceResponse.Hours)
			}
		})
	})
}

func TestInjectTransaction(t *testing.T) {
	skyService := getTestedMockedService()
	var (
		expectedCoins uint64 = 10
		expectedHours uint64 = 1
	)

	clientMock.On("Balance", mocklib.MatchedBy(func(address string) bool {
		if address != address {
			return false
		}
		return true
	})).Return(
		&wallet.BalancePair{
			wallet.Balance{
				Coins: expectedCoins,
				Hours: expectedHours,
			},
			wallet.Balance{},
		}, nil)

	t.Run("sign transaction", func(t *testing.T) {
		_, secKey := makeUxBodyWithSecret(t)
		secKeyHex := secKey.Hex()
		bRsp, err := skyService.SignTransaction(secKeyHex, rawTxStr)
		if err != nil {
			t.Fatal(err.Error())
		}
		if len(bRsp.Signid) == 0 {
			t.Fatalf("signid shouldn't be zero length")
		}
	})

	t.Run("inject transaction", func(t *testing.T) {
		// testing doubles: test input and generate output
		clientMock.On("InjectTransaction", mocklib.MatchedBy(func(txid string) bool {
			if rawTxStr != txid {
				return false
			}
			return true
		})).Return(rawTxID, nil)

		clientMock.On("Transaction", mocklib.MatchedBy(func(txid string) bool {
			if rawTxID != txid {
				return false
			}
			return true
		})).Return(
			&visor.TransactionResult{
				Status: visor.TransactionStatus{
					Confirmed: true,
				},
			}, nil)

		bRsp, err := skyService.InjectTransaction(rawTxStr)

		if err != nil {
			t.Fatal(err)
		}

		if len(bRsp.Transid) == 0 {
			t.Fatalf("signid shouldn't be zero length")
		}
	})
}

func TestCheckTransaction(t *testing.T) {
	var (
		expectedStatus           = true
		expectedHeight    uint64 = 12799
		expectedSeq       uint64 = 12799
		expectedTimestamp uint64 = 99999
	)
	skyService := getTestedMockedService()

	clientMock.On("Transaction", mocklib.MatchedBy(func(txid string) bool {
		if rawTxID != txid {
			return false
		}
		return true
	})).Return(
		&visor.TransactionResult{
			Status: visor.TransactionStatus{
				Confirmed: expectedStatus,
				Height:    expectedHeight,
				BlockSeq:  expectedSeq,
			},
			Time: expectedTimestamp,
		}, nil)

	txStatus, err := skyService.CheckTransactionStatus(rawTxID)

	if txStatus.Confirmed != expectedStatus {
		t.Errorf("Wrong txStatus status is not equal %v to actual %v", expectedStatus, txStatus.Confirmed)
	}

	if txStatus.Height != expectedHeight {
		t.Errorf("Wrong txStatus height expected %d actual %d", expectedHeight, txStatus.Height)
	}

	if txStatus.BlockSeq != expectedSeq {
		t.Errorf("Wrong txStatus block seq expected %d actual %d", expectedSeq, txStatus.BlockSeq)
	}

	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckBalance(t *testing.T) {
	skyService := getTestedMockedService()

	var (
		expectedCoins uint64 = 10
		expectedHours uint64 = 2
	)

	checkBalance := func(client ClientApi, addresses []string) (*wallet.BalancePair, error) {
		return &wallet.BalancePair{
			Confirmed: wallet.Balance{
				Coins: expectedCoins,
				Hours: expectedHours,
			},
		}, nil
	}

	skyService.checkBalance = checkBalance
	balanceResponse, err := skyService.CheckBalance(rawAddress)

	if balanceResponse.Address != rawAddress {
		t.Errorf("Wrong rawAddress expected %s actual %s", rawAddress, balanceResponse.Address)
	}

	if balanceResponse.Hours != expectedHours {
		t.Errorf("Wrong hours expected %d actual %d", expectedHours, balanceResponse.Hours)
	}

	if balanceResponse.Balance != expectedCoins {
		t.Errorf("Wrong coins expected %d actual %d", expectedCoins, balanceResponse.Balance)
	}

	if err != nil {
		t.Fatal(err)
	}
}

var getTestedMockedService = func() *SkyСoinService {
	loc := Node{
		Host: "127.0.0.1",
		Port: 6430,
	}

	clientMock = &mock.GuiClientMock{}
	skyService := NewSkyService(&loc)
	// inject mocked dependencies into tested service
	skyService.client = clientMock

	return skyService
}

func makeUxBodyWithSecret(t *testing.T) (coin.UxBody, cipher.SecKey) {
	p, s := cipher.GenerateKeyPair()
	return coin.UxBody{
		SrcTransaction: testutil.RandSHA256(t),
		Address:        cipher.AddressFromPubKey(p),
		Coins:          1e6,
		Hours:          100,
	}, s
}

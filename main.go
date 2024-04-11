package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
	"github.com/urfave/cli/v2"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/Layr-Labs/eigenda/api"
	regcoordinator "github.com/Layr-Labs/eigenda/contracts/bindings/RegistryCoordinator"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/node"
	"github.com/Layr-Labs/eigensdk-go/logging"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	OperatorAddressFlag = &cli.StringFlag{
		Name:     "operator-address",
		Usage:    "Operator address (EOA or smart contract address)",
		Required: true,
	}
	BlsKeyPathFlag = &cli.StringFlag{
		Name:     "bls-key-path",
		Usage:    "Path to the BLS key file",
		Required: true,
	}
	BlsKeyPasswordFlag = &cli.StringFlag{
		Name:     "bls-key-password",
		Usage:    "Password for the BLS key file",
		Required: true,
	}
	ChurnerURLFlag = &cli.StringFlag{
		Name:     "churner-url",
		Usage:    "URL of the churner service",
		Required: true,
	}
	QuorumsFlag = &cli.StringFlag{
		Name:     "quorums",
		Usage:    "Quorums to opt into 0/1/0,1",
		Required: true,
	}
)

func main() {
	app := cli.NewApp()

	app.Name = "register-param-gen"

	app.Flags = []cli.Flag{
		OperatorAddressFlag,
		BlsKeyPathFlag,
		BlsKeyPasswordFlag,
		ChurnerURLFlag,
		QuorumsFlag,
	}

	app.Action = GetParam

	if err := app.Run(os.Args); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

}

func GetParam(ctx *cli.Context) error {
	operatorAddress := ctx.String(OperatorAddressFlag.Name)
	blsKeyPath := ctx.String(BlsKeyPathFlag.Name)
	blsKeyPassword := ctx.String(BlsKeyPasswordFlag.Name)
	churnerURL := ctx.String(ChurnerURLFlag.Name)
	quorums := ctx.String(QuorumsFlag.Name)

	logger, err := logging.NewZapLogger(logging.Development)
	if err != nil {
		return err
	}

	churner := node.NewChurnerClient(
		churnerURL,
		true,
		time.Second*5,
		logger,
	)

	// Read the BLS key in SDK format
	sdkBlsKey, err := bls.ReadPrivateKeyFromFile(blsKeyPath, blsKeyPassword)
	if err != nil {
		fmt.Println("Error reading BLS key")
		return err
	}

	// cast it into a core.KeyPair
	keyPair := core.MakeKeyPair(sdkBlsKey.PrivKey)

	var quorumIds []core.QuorumID
	for _, quorum := range strings.Split(quorums, ",") {
		quorumInt, err := strconv.Atoi(quorum)
		if err != nil {
			return err
		}
		quorumIds = append(quorumIds, uint8(quorumInt))
	}

	churnReply, err := churner.Churn(context.Background(), operatorAddress, keyPair, quorumIds)
	if err != nil {
		return err
	}

	//// Generate salt and expiry
	//privateKeyBytes := []byte(keyPair.PrivKey.String())
	//salt := [32]byte{}
	//copy(salt[:], crypto.Keccak256([]byte("churn"), []byte(time.Now().String()), quorumIds, privateKeyBytes))
	//
	//// Get the current block number
	//expiry := big.NewInt((time.Now().Add(10 * time.Minute)).Unix())
	//
	//quorumNumbers := quorumIDsToQuorumNumbers(quorumIds)
	operatorsToChurn := make([]regcoordinator.IRegistryCoordinatorOperatorKickParam, len(churnReply.OperatorsToChurn))
	for i := range churnReply.OperatorsToChurn {
		if churnReply.OperatorsToChurn[i].QuorumId >= core.MaxQuorumID {
			return errors.New("quorum id is out of range")
		}

		operatorsToChurn[i] = regcoordinator.IRegistryCoordinatorOperatorKickParam{
			QuorumNumber: uint8(churnReply.OperatorsToChurn[i].QuorumId),
			Operator:     gethcommon.BytesToAddress(churnReply.OperatorsToChurn[i].Operator),
		}
	}

	var saltForApproverSig [32]byte
	copy(saltForApproverSig[:], churnReply.SignatureWithSaltAndExpiry.Salt[:])
	churnApproverSignature := regcoordinator.ISignatureUtilsSignatureWithSaltAndExpiry{
		Signature: churnReply.SignatureWithSaltAndExpiry.Signature,
		Salt:      saltForApproverSig,
		Expiry:    new(big.Int).SetInt64(churnReply.SignatureWithSaltAndExpiry.Expiry),
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(fmt.Sprintf("operatorKickParams: %v", operatorsToChurn))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("fields needed for churnApproverSignature")
	fmt.Println(fmt.Sprintf("Signature: %v", hex.EncodeToString(churnApproverSignature.Signature)))
	fmt.Println(fmt.Sprintf("Salt: %v", hex.EncodeToString(churnApproverSignature.Salt[:])))
	fmt.Println(fmt.Sprintf("Expiry: %v", churnApproverSignature.Expiry))
	fmt.Println(strings.Repeat("-", 80))

	return nil
}

package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/eager7/dogd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	vtypes "github.com/vultisig/verifier/types"
)

// policyToMessageHex converts a spec policy to a message hex string for signature verification.
// It joins policy fields with a delimiter and validates that no field contains the delimiter.
func PolicyToMessageHex(policy vtypes.PluginPolicy) ([]byte, error) {
	delimiter := "*#*"
	fields := []string{
		policy.Recipe,
		policy.PublicKey,
		fmt.Sprintf("%d", policy.PolicyVersion),
		policy.PluginVersion}
	for _, item := range fields {
		if strings.Contains(item, delimiter) {
			return nil, fmt.Errorf("invalid policy signature")
		}
	}
	result := strings.Join(fields, delimiter)
	return []byte(result), nil
}

func VerifyPolicySignature(publicKeyHex string, messageHex []byte, signature []byte) (bool, error) {
	msgHash := crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageHex), messageHex)))
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, err
	}

	pk, err := btcec.ParsePubKey(publicKeyBytes, btcec.S256())
	if err != nil {
		return false, err
	}

	ecdsaPubKey := ecdsa.PublicKey{
		Curve: btcec.S256(),
		X:     pk.X,
		Y:     pk.Y,
	}
	R := new(big.Int).SetBytes(signature[:32])
	S := new(big.Int).SetBytes(signature[32:64])

	return ecdsa.Verify(&ecdsaPubKey, msgHash, R, S), nil
}

// IsHexString checks if a string is a valid hex string
func IsHexString(s string) bool {
	hexRegex := regexp.MustCompile(`^[0-9a-fA-F]+$`)
	return hexRegex.MatchString(s) && len(s)%2 == 0
}

// IsXRPTransaction checks if a string is a valid XRP transaction
func IsXRPTransaction(txData string) bool {
	txHex := strings.TrimPrefix(txData, "0x")
	return IsHexString(txHex) && len(txHex) > 0
}

// ValidateNetworkTransaction validates transaction data based on network type
func ValidateNetworkTransaction(network, txData string) bool {
	network = strings.ToLower(network)

	switch network {
	// case "solana":
	// 	return IsSolanaTransaction(txData)
	case "xrp":
		return IsXRPTransaction(txData)
	case "ethereum":
		txHex := strings.TrimPrefix(txData, "0x")
		return IsHexString(txHex)
	default:
		txHex := strings.TrimPrefix(txData, "0x")
		return IsHexString(txHex)
	}
}

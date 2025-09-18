package api

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/pluginagent/common"
	vtypes "github.com/vultisig/verifier/types"
	vgcommon "github.com/vultisig/vultisig-go/common"
)

type ProposalResponse struct {
	PolicyID  string              `json:"policy_id"`
	Network   string              `json:"network"`
	TxHex     string              `json:"tx_hex"`
	Signature tss.KeysignResponse `json:"signature"`
}

func (s *Server) Propose(c echo.Context) error {
	policyID := c.QueryParam("policy_id")
	network := c.QueryParam("network")
	txData := c.QueryParam("tx_hex") // Keep parameter name for backwards compatibility

	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), uuid.MustParse(policyID))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugin policy"))
	}

	if !common.ValidateNetworkTransaction(network, txData) {
		s.logger.Error("Invalid transaction format for network")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid transaction format for network"))
	}

	chain, err := vgcommon.FromString(network)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get chain from network")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get chain from network"))
	}

	var tx []byte
	var signRequest *vtypes.PluginKeysignRequest

	networkLower := strings.ToLower(network)
	switch networkLower {
	case "solana":
		txBytes, er := base64.StdEncoding.DecodeString(txData)
		if er == nil {
			tx = txBytes
		} else {
			s.logger.WithError(er).Error("Failed to decode Solana transaction")
			return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid Solana transaction format"))
		}

		req, er := vtypes.NewPluginKeysignRequestEvm(*policy, "", chain, tx)
		if er != nil {
			s.logger.WithError(er).Error("Failed to create Solana signing request")
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(fmt.Sprintf("failed to create Solana signing request: %v", er)))
		}
		signRequest = req

	case "ethereum":
		txHex := strings.TrimPrefix(txData, "0x")
		txBytes, er := hex.DecodeString(txHex)
		if er != nil {
			s.logger.WithError(er).Error("Failed to decode tx hex")
			return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to decode tx hex"))
		}
		tx = txBytes

		req, er := vtypes.NewPluginKeysignRequestEvm(*policy, "", chain, tx)
		if er != nil {
			s.logger.WithError(er).Error("Failed to create EVM signing request")
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(fmt.Sprintf("failed to create EVM signing request: %v", er)))
		}
		signRequest = req

	default:
		txHex := strings.TrimPrefix(txData, "0x")
		txBytes, er := hex.DecodeString(txHex)
		if er != nil {
			s.logger.WithError(err).Error("Failed to decode tx hex")
			return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to decode tx hex"))
		}
		tx = txBytes

		req, er := vtypes.NewPluginKeysignRequestEvm(*policy, "", chain, tx)
		if er != nil {
			s.logger.WithError(er).Error("Failed to create signing request")
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(fmt.Sprintf("failed to create signing request: %v", er)))
		}
		signRequest = req
	}

	signatures, err := s.signer.Sign(c.Request().Context(), *signRequest)
	if err != nil {
		s.logger.WithError(err).Error("Failed to sign request")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to sign request"))
	}

	var sig tss.KeysignResponse
	for _, s := range signatures {
		sig = s
	}

	return c.JSON(http.StatusOK, ProposalResponse{
		PolicyID:  policyID,
		Network:   network,
		TxHex:     txData, // Return original format
		Signature: sig,
	})
}

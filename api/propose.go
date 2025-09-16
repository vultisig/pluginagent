package api

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/mobile-tss-lib/tss"
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
	txHex := c.QueryParam("tx_hex")
	// Strip 0x from the tx hex
	txHex = strings.TrimPrefix(txHex, "0x")

	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), uuid.MustParse(policyID))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugin policy"))
	}

	tx, err := hex.DecodeString(txHex)
	if err != nil {
		s.logger.WithError(err).Error("Failed to decode tx hex")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to decode tx hex"))
	}

	chain, err := vgcommon.FromString(network)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get chain from network")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get chain from network"))
	}

	signRequest, e := vtypes.NewPluginKeysignRequestEvm(
		*policy, "", chain, tx)
	if e != nil {
		s.logger.WithError(e).Error("Failed to create unsigned request")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(fmt.Sprintf("failed to create unsigned request: %v", e)))
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
		TxHex:     txHex,
		Signature: sig,
	})
}

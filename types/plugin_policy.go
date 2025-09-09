package types

import (
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/types"
)

type PluginPolicyWithRecipe struct {
	types.PluginPolicy
	Recipe *rtypes.Policy `json:"recipe"`
}

func FromPluginPolicy(policy types.PluginPolicy) (PluginPolicyWithRecipe, error) {
	recipe, err := policy.GetRecipe()
	if err != nil {
		return PluginPolicyWithRecipe{}, err
	}

	return PluginPolicyWithRecipe{
		PluginPolicy: policy,
		Recipe:       recipe,
	}, nil
}

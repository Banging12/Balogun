// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package genesis

import (
	"encoding/json"
	"math/big"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/common/decimal/bigintstr"
	"github.com/celo-org/celo-blockchain/common/decimal/fixed"
)

var _ = (*EpochRewardsParametersMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (e EpochRewardsParameters) MarshalJSON() ([]byte, error) {
	type EpochRewardsParameters struct {
		TargetVotingYieldInitial                     *fixed.Fixed         `json:"targetVotingYieldInitial"`
		TargetVotingYieldMax                         *fixed.Fixed         `json:"targetVotingYieldMax"`
		TargetVotingYieldAdjustmentFactor            *fixed.Fixed         `json:"targetVotingYieldAdjustmentFactor"`
		RewardsMultiplierMax                         *fixed.Fixed         `json:"rewardsMultiplierMax"`
		RewardsMultiplierAdjustmentFactorsUnderspend *fixed.Fixed         `json:"rewardsMultiplierAdjustmentFactorsUnderspend"`
		RewardsMultiplierAdjustmentFactorsOverspend  *fixed.Fixed         `json:"rewardsMultiplierAdjustmentFactorsOverspend"`
		TargetVotingGoldFraction                     *fixed.Fixed         `json:"targetVotingGoldFraction"`
		MaxValidatorEpochPayment                     *bigintstr.BigIntStr `json:"maxValidatorEpochPayment"`
		CommunityRewardFraction                      *fixed.Fixed         `json:"communityRewardFraction"`
		CarbonOffsettingPartner                      common.Address       `json:"carbonOffsettingPartner"`
		CarbonOffsettingFraction                     *fixed.Fixed         `json:"carbonOffsettingFraction"`
		Frozen                                       bool                 `json:"frozen"`
	}
	var enc EpochRewardsParameters
	enc.TargetVotingYieldInitial = e.TargetVotingYieldInitial
	enc.TargetVotingYieldMax = e.TargetVotingYieldMax
	enc.TargetVotingYieldAdjustmentFactor = e.TargetVotingYieldAdjustmentFactor
	enc.RewardsMultiplierMax = e.RewardsMultiplierMax
	enc.RewardsMultiplierAdjustmentFactorsUnderspend = e.RewardsMultiplierAdjustmentFactorsUnderspend
	enc.RewardsMultiplierAdjustmentFactorsOverspend = e.RewardsMultiplierAdjustmentFactorsOverspend
	enc.TargetVotingGoldFraction = e.TargetVotingGoldFraction
	enc.MaxValidatorEpochPayment = (*bigintstr.BigIntStr)(e.MaxValidatorEpochPayment)
	enc.CommunityRewardFraction = e.CommunityRewardFraction
	enc.CarbonOffsettingPartner = e.CarbonOffsettingPartner
	enc.CarbonOffsettingFraction = e.CarbonOffsettingFraction
	enc.Frozen = e.Frozen
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *EpochRewardsParameters) UnmarshalJSON(input []byte) error {
	type EpochRewardsParameters struct {
		TargetVotingYieldInitial                     *fixed.Fixed         `json:"targetVotingYieldInitial"`
		TargetVotingYieldMax                         *fixed.Fixed         `json:"targetVotingYieldMax"`
		TargetVotingYieldAdjustmentFactor            *fixed.Fixed         `json:"targetVotingYieldAdjustmentFactor"`
		RewardsMultiplierMax                         *fixed.Fixed         `json:"rewardsMultiplierMax"`
		RewardsMultiplierAdjustmentFactorsUnderspend *fixed.Fixed         `json:"rewardsMultiplierAdjustmentFactorsUnderspend"`
		RewardsMultiplierAdjustmentFactorsOverspend  *fixed.Fixed         `json:"rewardsMultiplierAdjustmentFactorsOverspend"`
		TargetVotingGoldFraction                     *fixed.Fixed         `json:"targetVotingGoldFraction"`
		MaxValidatorEpochPayment                     *bigintstr.BigIntStr `json:"maxValidatorEpochPayment"`
		CommunityRewardFraction                      *fixed.Fixed         `json:"communityRewardFraction"`
		CarbonOffsettingPartner                      *common.Address      `json:"carbonOffsettingPartner"`
		CarbonOffsettingFraction                     *fixed.Fixed         `json:"carbonOffsettingFraction"`
		Frozen                                       *bool                `json:"frozen"`
	}
	var dec EpochRewardsParameters
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.TargetVotingYieldInitial != nil {
		e.TargetVotingYieldInitial = dec.TargetVotingYieldInitial
	}
	if dec.TargetVotingYieldMax != nil {
		e.TargetVotingYieldMax = dec.TargetVotingYieldMax
	}
	if dec.TargetVotingYieldAdjustmentFactor != nil {
		e.TargetVotingYieldAdjustmentFactor = dec.TargetVotingYieldAdjustmentFactor
	}
	if dec.RewardsMultiplierMax != nil {
		e.RewardsMultiplierMax = dec.RewardsMultiplierMax
	}
	if dec.RewardsMultiplierAdjustmentFactorsUnderspend != nil {
		e.RewardsMultiplierAdjustmentFactorsUnderspend = dec.RewardsMultiplierAdjustmentFactorsUnderspend
	}
	if dec.RewardsMultiplierAdjustmentFactorsOverspend != nil {
		e.RewardsMultiplierAdjustmentFactorsOverspend = dec.RewardsMultiplierAdjustmentFactorsOverspend
	}
	if dec.TargetVotingGoldFraction != nil {
		e.TargetVotingGoldFraction = dec.TargetVotingGoldFraction
	}
	if dec.MaxValidatorEpochPayment != nil {
		e.MaxValidatorEpochPayment = (*big.Int)(dec.MaxValidatorEpochPayment)
	}
	if dec.CommunityRewardFraction != nil {
		e.CommunityRewardFraction = dec.CommunityRewardFraction
	}
	if dec.CarbonOffsettingPartner != nil {
		e.CarbonOffsettingPartner = *dec.CarbonOffsettingPartner
	}
	if dec.CarbonOffsettingFraction != nil {
		e.CarbonOffsettingFraction = dec.CarbonOffsettingFraction
	}
	if dec.Frozen != nil {
		e.Frozen = *dec.Frozen
	}
	return nil
}

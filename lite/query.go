package lite

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	hub "github.com/sentinel-official/hub/types"
	"github.com/sentinel-official/hub/x/node"
	"github.com/sentinel-official/hub/x/plan"
	"github.com/sentinel-official/hub/x/subscription"
	"github.com/sentinel-official/hub/x/vpn"
)

func (c *Client) Query(path string, params, result interface{}) (bool, error) {
	bytes, err := c.ctx.Codec.MarshalJSON(params)
	if err != nil {
		return false, err
	}

	res, _, err := c.ctx.QueryWithData(path, bytes)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}

	return true, c.ctx.Codec.UnmarshalJSON(res, result)
}

func (c *Client) QueryAccount(address sdk.AccAddress) (exported.Account, error) {
	var (
		result exported.Account
		path   = fmt.Sprintf("custom/%s/%s", auth.QuerierRoute, auth.QueryAccount)
	)

	if ok, err := c.Query(path, auth.NewQueryAccountParams(address), &result); !ok {
		return nil, err
	}

	return result, nil
}

func (c *Client) QueryNode(address hub.NodeAddress) (*node.Node, error) {
	var (
		result node.Node
		path   = fmt.Sprintf("custom/%s/%s/%s", vpn.StoreKey, node.QuerierRoute, node.QueryNode)
	)

	if ok, err := c.Query(path, node.NewQueryNodeParams(address), &result); !ok {
		return nil, err
	}

	return &result, nil
}

func (c *Client) QuerySubscription(id uint64) (*subscription.Subscription, error) {
	var (
		result subscription.Subscription
		path   = fmt.Sprintf("custom/%s/%s/%s", vpn.StoreKey, subscription.QuerierRoute, subscription.QuerySubscription)
	)

	if ok, err := c.Query(path, subscription.NewQuerySubscriptionParams(id), &result); !ok {
		return nil, err
	}

	return &result, nil
}

func (c *Client) QueryQuota(id uint64, address sdk.AccAddress) (*subscription.Quota, error) {
	var (
		result subscription.Quota
		path   = fmt.Sprintf("custom/%s/%s/%s", vpn.StoreKey, subscription.QuerierRoute, subscription.QueryQuota)
	)

	if ok, err := c.Query(path, subscription.NewQueryQuotaParams(id, address), &result); !ok {
		return nil, err
	}

	return &result, nil
}

func (c *Client) HasNodeForPlan(id uint64, address hub.NodeAddress) (bool, error) {
	res, _, err := c.ctx.QueryStore(plan.NodeForPlanKey(id, address),
		fmt.Sprintf("%s/%s", vpn.ModuleName, plan.ModuleName))
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}

	var item bool
	if err := c.ctx.Codec.UnmarshalJSON(res, &item); err != nil {
		return false, err
	}

	return item, nil
}

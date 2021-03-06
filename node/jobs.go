package node

import (
	"encoding/base64"
	"time"

	"github.com/sentinel-official/dvpn-node/types"
)

func (n *Node) jobUpdateStatus() error {
	n.Logger().Info("Started job", "name", "update_status", "interval", n.IntervalStatus())

	t := time.NewTicker(n.IntervalStatus())
	for ; ; <-t.C {
		if err := n.updateStatus(); err != nil {
			return err
		}
	}
}

func (n *Node) jobUpdateSessions() error {
	n.Logger().Info("Started job", "name", "update_sessions", "interval", n.IntervalSessions())

	t := time.NewTicker(n.IntervalSessions())
	for ; ; <-t.C {
		var (
			items   []types.Session
			timeNow = time.Now()
		)

		peers, err := n.Service().Peers()
		if err != nil {
			return err
		}

		for _, peer := range peers {
			key, err := base64.StdEncoding.DecodeString(peer.Identity)
			if err != nil {
				return err
			}

			item := n.Sessions().Get(peer.Identity)
			if item.Identity == "" || peer.Download == item.Download {
				if err := n.Service().RemovePeer(key); err != nil {
					return err
				}

				n.Sessions().Delete(peer.Identity)
				continue
			}

			item.Upload = peer.Upload - item.Upload
			item.Download = peer.Download - item.Download
			item.Duration = timeNow.Sub(item.ConnectedAt) - item.Duration
			items = append(items, item)

			quota, err := n.Client().QueryQuota(item.Subscription, item.Address)
			if err != nil {
				return err
			}

			if quota.Consumed.AddRaw(item.Upload + item.Download).GT(quota.Allocated) {
				if err := n.Service().RemovePeer(key); err != nil {
					return err
				}

				n.Sessions().Delete(peer.Identity)
			}
		}

		if err := n.updateSessions(items); err != nil {
			return err
		}

		for _, item := range items {
			session := n.Sessions().Get(item.Identity)

			session.Upload += item.Upload
			session.Download += item.Download
			session.Duration += item.Duration
			n.Sessions().Set(session)
		}
	}
}

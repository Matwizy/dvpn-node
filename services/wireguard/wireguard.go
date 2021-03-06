package wireguard

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	wgt "github.com/sentinel-official/dvpn-node/services/wireguard/types"
	"github.com/sentinel-official/dvpn-node/types"
)

const (
	InfoLen = 2 + 32
)

var _ types.Service = (*WireGuard)(nil)

type WireGuard struct {
	cfg   *wgt.Config
	info  []byte
	peers *wgt.Peers
	pool  *wgt.IPPool
}

func NewWireGuard(pool *wgt.IPPool) types.Service {
	return &WireGuard{
		pool:  pool,
		cfg:   wgt.NewConfig(),
		info:  make([]byte, InfoLen),
		peers: wgt.NewPeers(),
	}
}

func (w *WireGuard) Type() uint64 {
	return wgt.Type
}

func (w *WireGuard) Initialize(home string) error {
	configFilePath := filepath.Join(home, wgt.ConfigFileName)
	if err := w.cfg.LoadFromPath(configFilePath); err != nil {
		return err
	}

	t, err := template.New("").Parse(configTemplate)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, w.cfg); err != nil {
		return err
	}

	configFilePath = fmt.Sprintf("/etc/wireguard/%s.conf", w.cfg.Interface)
	if err := ioutil.WriteFile(configFilePath, buffer.Bytes(), 0600); err != nil {
		return err
	}

	key, err := wgt.KeyFromString(w.cfg.PrivateKey)
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint16(w.info[:2], w.cfg.ListenPort)
	copy(w.info[2:], key.Public().Bytes())

	return nil
}

func (w *WireGuard) Info() []byte {
	return w.info
}

func (w *WireGuard) Start() error {
	cmd := exec.Command("wg-quick", strings.Split(
		fmt.Sprintf("up %s", w.cfg.Interface), " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (w *WireGuard) Stop() error {
	cmd := exec.Command("wg-quick", strings.Split(
		fmt.Sprintf("down %s", w.cfg.Interface), " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (w *WireGuard) AddPeer(data []byte) (result []byte, err error) {
	identity := base64.StdEncoding.EncodeToString(data)

	v4, v6, err := w.pool.Get()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("wg", strings.Split(
		fmt.Sprintf(`set %s peer %s allowed-ips %s/32,%s/128`,
			w.cfg.Interface, identity, v4.IP(), v6.IP()), " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		w.pool.Release(v4, v6)
		return nil, err
	}

	if peer, ok := w.peers.Get(identity); ok {
		w.peers.Delete(identity)
		w.pool.Release(peer.IPv4, peer.IPv6)
	}

	w.peers.Put(wgt.Peer{
		Identity: identity,
		IPv4:     v4,
		IPv6:     v6,
	})

	result = append(result, v4.Bytes()...)
	result = append(result, v6.Bytes()...)
	return result, nil
}

func (w *WireGuard) RemovePeer(data []byte) error {
	identity := base64.StdEncoding.EncodeToString(data)

	cmd := exec.Command("wg", strings.Split(
		fmt.Sprintf(`set %s peer %s remove`,
			w.cfg.Interface, identity), " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	if peer, ok := w.peers.Get(identity); ok {
		w.peers.Delete(identity)
		w.pool.Release(peer.IPv4, peer.IPv6)
	}

	return nil
}

func (w *WireGuard) Peers() ([]types.Peer, error) {
	output, err := exec.Command("wg", strings.Split(
		fmt.Sprintf("show %s transfer", w.cfg.Interface), " ")...).Output()
	if err != nil {
		return nil, err
	}

	// nolint: prealloc
	var (
		items []types.Peer
		lines = strings.Split(string(output), "\n")
	)

	for _, line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		download, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return nil, err
		}

		upload, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return nil, err
		}

		items = append(items, types.Peer{
			Identity: columns[0],
			Upload:   upload,
			Download: download,
		})
	}

	return items, nil
}

func (w *WireGuard) PeersCount() int {
	return w.peers.Len()
}

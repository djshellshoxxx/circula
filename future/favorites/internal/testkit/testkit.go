// Package testkit holds synthetic fixtures shared by this module's tests.
package testkit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/djshellshoxxx/circula/future/favorites"
)

// Directory is a fake Policy and TradeSource over synthetic members.
type Directory struct {
	mu      sync.Mutex
	members map[string]favorites.Member // key network + "/" + id
	blocks  map[[3]string]bool          // viewer-relative block: network, viewer, other
	trades  map[string]favorites.HistoricalTrade
	Fail    atomic.Bool
	Calls   atomic.Int64
}

func NewDirectory() *Directory {
	return &Directory{members: map[string]favorites.Member{}, blocks: map[[3]string]bool{}, trades: map[string]favorites.HistoricalTrade{}}
}

// Add registers an active, visible member.
func (d *Directory) Add(network, id string) {
	d.Set(favorites.Member{ID: id, Network: network, Active: true, Visible: true, DisplayName: "Member " + id})
}

func (d *Directory) Set(m favorites.Member) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.members[m.Network+"/"+m.ID] = m
}

func (d *Directory) Get(network, id string) favorites.Member {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.members[network+"/"+id]
}

func (d *Directory) Remove(network, id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.members, network+"/"+id)
}

// Block makes other appear Blocked to viewer.
func (d *Directory) Block(network, viewer, other string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.blocks[[3]string{network, viewer, other}] = true
}

func (d *Directory) AddTrade(id string, t favorites.HistoricalTrade) {
	d.mu.Lock()
	defer d.mu.Unlock()
	t.ID = id
	d.trades[t.Network+"/"+id] = t
}

func (d *Directory) Members(_ context.Context, viewer favorites.Actor, ids []string) (map[string]favorites.Member, error) {
	d.Calls.Add(1)
	if d.Fail.Load() {
		return nil, errors.New("policy backend down")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[string]favorites.Member{}
	for _, id := range ids {
		m, ok := d.members[viewer.Network+"/"+id]
		if !ok {
			continue
		}
		if d.blocks[[3]string{viewer.Network, viewer.Member, id}] {
			m.Blocked = true
		}
		out[id] = m
	}
	return out, nil
}

// Trade returns the trade only to its parties.
func (d *Directory) Trade(_ context.Context, viewer favorites.Actor, id string) (favorites.HistoricalTrade, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	t, ok := d.trades[viewer.Network+"/"+id]
	if !ok || (t.Payer != viewer.Member && t.Payee != viewer.Member) {
		return favorites.HistoricalTrade{}, favorites.ErrTradeNotFound
	}
	return t, nil
}

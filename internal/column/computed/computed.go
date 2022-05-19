// Copyright 2019-2020 Grabtaxi Holdings PTE LTE (GRAB), All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file

package computed

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/kelindar/lua"
	"github.com/kelindar/talaria/internal/encoding/typeof"
	"github.com/kelindar/talaria/internal/monitor"
	script "github.com/kelindar/talaria/internal/scripting"
	mlog "github.com/kelindar/talaria/internal/scripting/log"
	mnet "github.com/kelindar/talaria/internal/scripting/net"
	mstats "github.com/kelindar/talaria/internal/scripting/stats"
)

const (
	LuaLoaderTyp    = "lua"
	PluginLoaderTyp = "plugin"
)

// Computed represents a computed column
type Computed interface {
	Name() string // return column 's name
	Type() typeof.Type
	Value(map[string]interface{}) (interface{}, error)
}

// NewComputed creates a new script from a string
func NewComputed(columnName, functionName string, outpuTyp typeof.Type, uriOrCode string, monitor monitor.Monitor) (Computed, error) {
	switch uriOrCode {
	case "make://identifier":
		return newIdentifier(columnName), nil
	case "make://timestamp":
		return newTimestamp(columnName), nil
	}

	pluginLoader := script.NewPluginLoader(functionName)
	luaLoader := script.NewLuaLoader([]lua.Module{
		mlog.New(monitor),
		mstats.New(monitor),
		mnet.New(monitor),
	}, outpuTyp)
	l := script.NewHandlerLoader(pluginLoader, luaLoader)

	h, err := l.LoadHandler(uriOrCode)
	if err != nil {
		return nil, err
	}

	return &loadComputed{
		name:   columnName,
		loader: h,
		typ:    outpuTyp,
	}, nil
}

// ------------------------------------------------------------------------------------------------------------

// identifier represents a computed column that generates an event ID
type identifier struct {
	seq  uint32 // Sequence counter
	rnd  uint32 // Random component
	name string // Name of the column
}

// newIdentifier creates a new ID generator column
func newIdentifier(name string) *identifier {
	b := make([]byte, 4)
	rand.Read(b)
	uniq := binary.BigEndian.Uint32(b)

	return &identifier{
		seq:  0,
		rnd:  uniq,
		name: name,
	}
}

// Name returns the name of the column
func (c *identifier) Name() string {
	return c.name
}

// Type returns the type of the column
func (c *identifier) Type() typeof.Type {
	return typeof.String
}

// Value computes the column value for the row
func (c *identifier) Value(row map[string]interface{}) (interface{}, error) {
	id := make([]byte, 16)
	binary.BigEndian.PutUint64(id[0:8], uint64(time.Now().UTC().UnixNano()))
	binary.BigEndian.PutUint32(id[8:12], atomic.AddUint32(&c.seq, 1))
	binary.BigEndian.PutUint32(id[12:16], c.rnd)
	return hex.EncodeToString(id), nil
}

// ------------------------------------------------------------------------------------------------------------

// Timestamp represents a timestamp computed column
type timestamp struct {
	name string // Name of the column
}

// newIdentifier creates a new ID generator column
func newTimestamp(name string) *timestamp {
	return &timestamp{
		name: name,
	}
}

// Name returns the name of the column
func (c *timestamp) Name() string {
	return c.name
}

// Type returns the type of the column
func (c *timestamp) Type() typeof.Type {
	return typeof.Timestamp
}

// Value computes the column value for the row
func (c *timestamp) Value(row map[string]interface{}) (interface{}, error) {
	return time.Now().UTC().Unix(), nil
}

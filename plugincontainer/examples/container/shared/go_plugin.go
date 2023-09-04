// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import "github.com/hashicorp/go-plugin"

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"counter": &CounterPlugin{},
}

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"istio.io/istio/tools/istio-clean-iptables/pkg/config"
	"istio.io/istio/tools/istio-iptables/pkg/builder"
	common "istio.io/istio/tools/istio-iptables/pkg/cmd"
	"istio.io/istio/tools/istio-iptables/pkg/constants"
	dep "istio.io/istio/tools/istio-iptables/pkg/dependencies"
)

func flushAndDeleteChains(ext dep.Dependencies, cmd string, table string, chains []string) {
	for _, chain := range chains {
		ext.RunQuietlyAndIgnore(cmd, "-t", table, "-F", chain)
		ext.RunQuietlyAndIgnore(cmd, "-t", table, "-X", chain)
	}
}

func removeOldChains(cfg *config.Config, ext dep.Dependencies, cmd string) {
	// Remove the old TCP rules
	for _, table := range []string{constants.NAT, constants.MANGLE} {
		ext.RunQuietlyAndIgnore(cmd, "-t", table, "-D", constants.PREROUTING, "-p", constants.TCP, "-j", constants.ISTIOINBOUND)
	}
	ext.RunQuietlyAndIgnore(cmd, "-t", constants.NAT, "-D", constants.OUTPUT, "-p", constants.TCP, "-j", constants.ISTIOOUTPUT)

	redirectDNS := cfg.RedirectDNS
	// Remove the old DNS UDP rules
	if redirectDNS {
		common.HandleDNSUDP(common.DeleteOps, builder.NewIptablesBuilder(nil), ext, cmd, cfg.ProxyUID, cfg.ProxyGID, cfg.DNSServersV4, cfg.CaptureAllDNS)
	}

	// Flush and delete the istio chains from NAT table.
	chains := []string{constants.ISTIOOUTPUT, constants.ISTIOINBOUND}
	flushAndDeleteChains(ext, cmd, constants.NAT, chains)
	// Flush and delete the istio chains from MANGLE table.
	chains = []string{constants.ISTIOINBOUND, constants.ISTIODIVERT, constants.ISTIOTPROXY}
	flushAndDeleteChains(ext, cmd, constants.MANGLE, chains)

	// Must be last, the others refer to it
	chains = []string{constants.ISTIOREDIRECT, constants.ISTIOINREDIRECT}
	flushAndDeleteChains(ext, cmd, constants.NAT, chains)
}

func cleanup(cfg *config.Config) {
	var ext dep.Dependencies
	if cfg.DryRun {
		ext = &dep.StdoutStubDependencies{}
	} else {
		ext = &dep.RealDependencies{}
	}

	defer func() {
		for _, cmd := range []string{constants.IPTABLESSAVE, constants.IP6TABLESSAVE} {
			// iptables-save is best efforts
			_ = ext.Run(cmd)
		}
	}()

	for _, cmd := range []string{constants.IPTABLES, constants.IP6TABLES} {
		removeOldChains(cfg, ext, cmd)
	}
}

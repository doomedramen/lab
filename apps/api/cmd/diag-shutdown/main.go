// Diagnostic tool to investigate VM shutdown behavior
// Usage: go run ./cmd/diag-shutdown <vmid>
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: diag-shutdown <vmname>  e.g.  diag-shutdown vm-100")
	}
	vmName := os.Args[1]

	conn, err := libvirt.NewConnect("qemu:///session")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	dom, err := conn.LookupDomainByName(vmName)
	if err != nil {
		log.Fatalf("lookup %s: %v", vmName, err)
	}
	defer dom.Free()

	// --- Current state ---
	state, reason, err := dom.GetState()
	if err != nil {
		log.Fatalf("GetState: %v", err)
	}
	fmt.Printf("=== %s ===\n", vmName)
	fmt.Printf("State: %v  Reason: %v\n\n", stateName(state), reason)

	// --- Inspect domain XML for ACPI/power management features ---
	xmlStr, err := dom.GetXMLDesc(0)
	if err != nil {
		log.Fatalf("GetXMLDesc: %v", err)
	}
	var domCfg libvirtxml.Domain
	if err := domCfg.Unmarshal(xmlStr); err != nil {
		log.Fatalf("unmarshal xml: %v", err)
	}

	fmt.Println("--- Features ---")
	if domCfg.Features != nil {
		fmt.Printf("  ACPI:   %v\n", domCfg.Features.ACPI != nil)
		fmt.Printf("  APIC:   %v\n", domCfg.Features.APIC != nil)
		fmt.Printf("  VMPort: %v\n", domCfg.Features.VMPort != nil)
	} else {
		fmt.Println("  (no <features> element)")
	}

	fmt.Println("\n--- On-event/power actions ---")
	if domCfg.OnPoweroff != "" {
		fmt.Printf("  on_poweroff: %s\n", domCfg.OnPoweroff)
	}
	if domCfg.OnReboot != "" {
		fmt.Printf("  on_reboot:   %s\n", domCfg.OnReboot)
	}
	if domCfg.OnCrash != "" {
		fmt.Printf("  on_crash:    %s\n", domCfg.OnCrash)
	}

	// --- Check if QEMU Guest Agent channel is configured ---
	hasAgent := false
	if domCfg.Devices != nil {
		for _, ch := range domCfg.Devices.Channels {
			if ch.Target != nil && ch.Target.VirtIO != nil &&
				ch.Target.VirtIO.Name == "org.qemu.guest_agent.0" {
				hasAgent = true
			}
		}
	}
	fmt.Printf("\n--- QEMU Guest Agent channel: %v ---\n", hasAgent)

	if state != libvirt.DOMAIN_RUNNING {
		fmt.Println("\nVM not running — starting it for the test...")
		if err := dom.Create(); err != nil {
			log.Fatalf("start: %v", err)
		}
		time.Sleep(2 * time.Second)
		state, _, _ = dom.GetState()
		fmt.Printf("State after start: %v\n", stateName(state))
	}

	// --- Try each shutdown flag individually ---
	flags := []struct {
		name  string
		flags libvirt.DomainShutdownFlags
	}{
		{"DEFAULT (0)", libvirt.DOMAIN_SHUTDOWN_DEFAULT},
		{"ACPI_POWER_BTN", libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN},
		{"GUEST_AGENT", libvirt.DOMAIN_SHUTDOWN_GUEST_AGENT},
	}

	flagIdx := 0
	if len(os.Args) >= 3 {
		n, _ := strconv.Atoi(os.Args[2])
		flagIdx = n
	}

	chosen := flags[flagIdx]
	fmt.Printf("\n=== Testing ShutdownFlags(%s) ===\n", chosen.name)

	if err := dom.ShutdownFlags(chosen.flags); err != nil {
		fmt.Printf("ShutdownFlags returned error: %v\n", err)
	} else {
		fmt.Println("ShutdownFlags returned nil (success)")
	}

	// Poll state for 12 seconds
	fmt.Println("\nPolling domain state:")
	for i := range 24 {
		time.Sleep(500 * time.Millisecond)
		s, r, _ := dom.GetState()
		fmt.Printf("  +%.1fs  state=%s reason=%d\n", float64(i+1)*0.5, stateName(s), r)
		if s == libvirt.DOMAIN_SHUTOFF {
			fmt.Println("✓ Domain reached SHUTOFF")
			return
		}
	}

	fmt.Println("✗ Domain still running after 12s — shutdown did not work")
	fmt.Println("\nCleaning up with Destroy...")
	if err := dom.DestroyFlags(libvirt.DOMAIN_DESTROY_DEFAULT); err != nil {
		fmt.Printf("  Destroy error: %v\n", err)
	} else {
		fmt.Println("  Destroyed OK")
	}
}

func stateName(s libvirt.DomainState) string {
	switch s {
	case libvirt.DOMAIN_NOSTATE:
		return "NOSTATE"
	case libvirt.DOMAIN_RUNNING:
		return "RUNNING"
	case libvirt.DOMAIN_BLOCKED:
		return "BLOCKED"
	case libvirt.DOMAIN_PAUSED:
		return "PAUSED"
	case libvirt.DOMAIN_SHUTDOWN:
		return "SHUTDOWN"
	case libvirt.DOMAIN_SHUTOFF:
		return "SHUTOFF"
	case libvirt.DOMAIN_CRASHED:
		return "CRASHED"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "PMSUSPENDED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

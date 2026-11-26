package netops

import (
	"fmt"
	"regexp"

	"github.com/shahriarhaqueabir/WinOps/internal/common"
)

// validInterfaceName matches typical Windows interface names (letters, digits,
// dashes, underscores, dots, spaces, parentheses). It intentionally excludes
// shell metacharacters.
var validInterfaceName = regexp.MustCompile(`^[a-zA-Z0-9 ._()*-]{1,128}$`)

// sanitizeInterfaceName rejects interface names containing shell
// metacharacters before they ever reach exec.Command.
func sanitizeInterfaceName(iface string) (string, error) {
	if iface == "" {
		return "", nil
	}
	if !validInterfaceName.MatchString(iface) {
		return "", fmt.Errorf("invalid interface name")
	}
	return iface, nil
}

// RunNetworkAction executes a named network action with optional parameters.
func RunNetworkAction(action string, params map[string]string) error {
	iface, err := sanitizeInterfaceName(params["interface"])
	if err != nil {
		return err
	}
	switch action {
	case "flush_dns":
		return FlushDNSCache()
	case "renew_dhcp":
		return renewDHCP(iface)
	case "reset_interface":
		return resetInterface(iface)
	case "disable_interface":
		return setInterfaceState(iface, false)
	case "enable_interface":
		return setInterfaceState(iface, true)
	case "clear_arp_cache":
		return clearARPCache()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func renewDHCP(iface string) error {
	if iface == "" {
		iface = "*"
	}
	_, err := common.HiddenCommand("ipconfig", "/renew", iface).CombinedOutput()
	return err
}

func resetInterface(iface string) error {
	if iface == "" {
		return fmt.Errorf("interface name required")
	}
	if err := setInterfaceState(iface, false); err != nil {
		return err
	}
	common.HiddenCommand("timeout", "2", "cmd", "/c", "echo.").Run()
	return setInterfaceState(iface, true)
}

func setInterfaceState(iface string, enable bool) error {
	if iface == "" {
		return fmt.Errorf("interface name required")
	}
	state := "enable"
	if !enable {
		state = "disable"
	}
	_, err := common.HiddenCommand("netsh", "interface", "set", "interface", "name="+iface, state).CombinedOutput()
	return err
}

func clearARPCache() error {
	_, err := common.HiddenCommand("netsh", "interface", "ipv4", "delete", "arpcache").CombinedOutput()
	return err
}
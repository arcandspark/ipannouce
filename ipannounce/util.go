package ipannounce

import (
	"encoding/binary"
	"fmt"
	"log"
	"log/syslog"
	"math/bits"
	"net"
	"regexp"
	"strings"
)

var logg *syslog.Writer

/* Select a host IP for an ipannounce packet.
 * This function will iterate over the host's interfaces for IPs to evaluate.
 *
 * if_pat is a Rexexp that, if provided, will be used to test each interface name.
 * Interfaces whose name's do NOT match if_pat will not have any of their
 * addresses evaluated.
 *
 * selector_ip is an IPv6 address which will be bitwise compared to each
 * host IP address which is evaluated.
 *
 * The host IP address which, when compared to selector_ip, has the most consecutive
 * matching bits starting with the MSB will be returned as the selected source IP.
 *
 * NOTE: This function ONLY compares the most significant 64 bits of each address to
 * the selector_ip.
 *
 * Example:
 * selector_ip    = fc00::
 * candidate IP A = fd35:a2b9:543c:10aa::20
 * candidate IP B = 2605:4415:92bc:115f::20
 *
 * (for readability, only the most significant 32 bits are shown in this example)
 * selector IP    = 1111 1100 0000 0000 0000 0000 0000 0000
 * candidate IP A = 1111 1101 0011 0101 1010 0010 1011 1001
 * candidate IP B = 0010 0110 0000 0101 0100 0100 0001 0101
 *
 * As can be easily seen in binary above, candidate IP A has 7 consecutive matching bits
 * with the selector_ip starting from the MSB. candidate IP B has none. Given this selector,
 * candidate IP A will be returned.
 *
 * The selector_ip can be more specific than shown, and the full first 64 bits will be evaluated.
 * This can be useful for hosts with multiple GUAs or ULAs with different prefixes.
 */
func SelectMatchingIP(selector_ip net.IP, if_pat *regexp.Regexp) (net.IP, error) {
	iflist, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error getting interface list: %v", err)
	}

	// Track most matched bits and the IP that matched that many from the loop below
	// These will be updated when a better match is found
	most_matched_bits := 0
	var best_match_ip net.IP = nil
	selector_netpart := binary.BigEndian.Uint64(selector_ip[:8])

	for i := range iflist {
		if if_pat != nil { // if_pat was provided
			if !if_pat.MatchString(iflist[i].Name) {
				continue
			}
		}

		addrlist, err := iflist[i].Addrs()
		if err != nil {
			return nil, fmt.Errorf("ERROR getting addresses for interface %v: %v", iflist[i].Name, err)
		}

		for j := range addrlist {
			// Addresses returned by Interface.Addrs will have prefix bits on the end of them
			// Example: fd00::1/64
			// net.ParseIP will not parse an IP in a string string with the prefix bits at the end
			// so we strip that off
			addr_parts := strings.Split(addrlist[j].String(), "/")
			if len(addr_parts) < 2 {
				return nil, fmt.Errorf("ERROR Addr split into fewer than two parts on /: %v", addrlist[j].String())
			}

			ifip := net.ParseIP(addr_parts[0])
			if ifip == nil {
				return nil, fmt.Errorf("ERROR could not parse IP: %v", addr_parts[0])
			}

			// I only want to work on IPv6 addresses
			// Sorry Dave, there is no support for your old addressing scheme from the tiny Internet
			if ifip.To4() == nil {
				for b := 0; b < 16; b++ {
					ifip_netpart := binary.BigEndian.Uint64(ifip[:8])
					matching_bits := bits.LeadingZeros64(selector_netpart ^ ifip_netpart)
					if matching_bits > most_matched_bits {
						best_match_ip = ifip
						most_matched_bits = matching_bits
					}
				}
			} // else it was an IPv4 address, do nothing with it
		}
	}

	return best_match_ip, nil
}

func LogSetup() {
	var err error
	logg, err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "ipannounce")
	if err != nil {
		LogErrorf("failed to setup syslog: %v", err)
	}
}

func LogErrorf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	var err error
	if logg != nil {
		err = logg.Err(msg)
	}
	if logg == nil || err != nil {
		log.Default().Printf("ERROR %v", msg)
	}
}

func LogInfof(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	var err error
	if logg != nil {
		err = logg.Info(msg)
	}
	if logg == nil || err != nil {
		log.Default().Printf("INFO %v", msg)
	}
}

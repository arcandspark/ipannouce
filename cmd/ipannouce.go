package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"

	"omt.cx/m/v2/ipannounce"
)

/* Recommended multicast IP selection method:
 * ff12:0000:0000:0000:0000:0000:c06f:fe3a
 * ^^---------------------------------------multicast
 *   ^--------------------------------------0 reserved, 0 rendezvous, 0 prefix, 1 transient
 *    ^-------------------------------------link-local scope, don't route
 *      ^^^^--------------------------------future flags, reserved, plen, all 0s
 *           ^^^^ ^^^^ ^^^^ ^^^^------------64bit network
 *                               ^^^^ ^^^^--32bit group, choose randomly
 *
 * Above example short notation: ff12::c06f:fe3a
 */

func main() {
	fmt.Println("Announce IPv6 address with hostname")

	selector_ip_str := ""
	if_pat_str := ""
	flag.StringVar(&selector_ip_str, "selector", "", "interface address most like this address will be used to transmit annoucement")
	flag.StringVar(&if_pat_str, "ifpat", "", "regex pattern that interface name must match to have any of its addresses selected")
	flag.Parse()

	if selector_ip_str == "" {
		fmt.Println("Use -selector <ipv6 addr> to specify an address for source IP selection")
		os.Exit(1)
	}

	selector_ip := net.ParseIP(selector_ip_str)
	if selector_ip == nil {
		fmt.Println("Provided selector address was not an IP address")
		os.Exit(1)
	}

	if selector_ip.To4() != nil {
		fmt.Println("Provided selector address was not an IPv6 address")
		os.Exit(1)
	}

	var if_pat *regexp.Regexp = nil
	if if_pat_str != "" {
		var err error
		if_pat, err = regexp.Compile(if_pat_str)
		if err != nil {
			fmt.Printf("Could not compile regex for -ifpat: %v\n", err)
			os.Exit(1)
		}
	}

	source_ip, err := ipannounce.SelectSourceIP(selector_ip, if_pat)
	if err != nil {
		log.Fatalf("error selecting source IP: %v", err)
	}
	if source_ip == nil {
		log.Fatal("No source IP could be selected with provided selector and ifpat args")
	}

	fmt.Printf("IP most like selector is: %v\n", source_ip.String())
	hostname, _ := os.Hostname()
	doti := strings.Index(hostname, ".")
	if doti > -1 {
		fmt.Println("Hostname contains a domain part")
		hostname = strings.Split(hostname, ".")[0]
	}
	fmt.Printf("Hostname: %v\n", hostname)
}

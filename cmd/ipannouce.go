package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"

	"omt.cx/m/v2/ipannounce"
)

// IP Annouce default multicast group address: ff15::793e:287a
//
// Default multicast IP selection method:
// ff15:0000:0000:0000:0000:0000:793e:287a
// ^^---------------------------------------multicast
//	 ^--------------------------------------0 reserved, 0 rendezvous, 0 prefix, 1 transient
//	  ^-------------------------------------link-local scope, don't route
//	    ^^^^--------------------------------future flags, reserved, plen, all 0s
//	         ^^^^ ^^^^ ^^^^ ^^^^------------64bit network
//	                             ^^^^ ^^^^--32bit group, choosen randomly

func main() {
	fmt.Println("Announce IPv6 address with hostname")

	var mode, group_str, selector_ip_str, if_pat_str string
	var solport, annport uint
	flag.StringVar(&mode, "mode", "ann", "operation mode")
	flag.StringVar(&group_str, "group", "ff15::793e:287a", "multicast group")
	flag.UintVar(&solport, "solport", 5190, "solicitor port")
	flag.UintVar(&annport, "annport", 5190, "announcer port")
	flag.StringVar(&selector_ip_str, "selector", "", "interface address most like this address will be used to transmit annoucement")
	flag.StringVar(&if_pat_str, "ifpat", "", "regex pattern that interface name must match to have any of its addresses selected")
	flag.Parse()

	// Validate mode
	if mode != "ann" && mode != "sol" {
		fmt.Printf("Unsupported mode %v, must be either ann or sol\n", mode)
		os.Exit(1)
	}

	// Validate common options
	group_ip := net.ParseIP(group_str)
	if group_ip == nil {
		fmt.Printf("error parsing group IP: %v\n", group_str)
		os.Exit(1)
	}

	// Validate solicitor options
	var if_pat *regexp.Regexp = nil
	var selector_ip net.IP
	if mode == "sol" {
		if selector_ip_str == "" {
			fmt.Println("Use -selector <ipv6 addr> to specify an address for source IP selection")
			os.Exit(1)
		}

		selector_ip = net.ParseIP(selector_ip_str)
		if selector_ip == nil {
			fmt.Println("Provided selector address was not an IP address")
			os.Exit(1)
		}

		if selector_ip.To4() != nil {
			fmt.Println("Provided selector address was not an IPv6 address")
			os.Exit(1)
		}

		if if_pat_str != "" {
			var err error
			if_pat, err = regexp.Compile(if_pat_str)
			if err != nil {
				fmt.Printf("Could not compile regex for -ifpat: %v\n", err)
				os.Exit(1)
			}
		}
	}

	sol_listen_addr := net.JoinHostPort("::", fmt.Sprint(solport))
	ann_listen_addr := net.JoinHostPort("::", fmt.Sprint(annport))

	if mode == "sol" {
		inform_ip, err := ipannounce.SelectMatchingIP(selector_ip, if_pat)
		if err != nil {
			log.Fatalf("error selecting source IP: %v", err)
		}
		if inform_ip == nil {
			log.Fatal("No source IP could be selected with provided selector and ifpat args")
		}

		fmt.Printf("Running as solicitor using address %v\n", inform_ip)
		fmt.Printf("Solicitor listening on %v\n", sol_listen_addr)
	} else if mode == "ann" {
		fmt.Printf("Announcer listening on %v\n", ann_listen_addr)
		fmt.Printf("Announcer joining group %v\n", group_ip.String())
		ipannounce.Announcer(ann_listen_addr, group_ip)
	}
}

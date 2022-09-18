package ipannounce

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/ipv6"
)

// Join the specified multicast group address with
// all interfaces on this host that are not "lo"
func JoinGroup(pc *ipv6.PacketConn, group *net.UDPAddr) error {
	iflist, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("error listing interfaces: %v", err)
	}

	for i := range iflist {
		if iflist[i].Name != "lo" {
			if err := pc.JoinGroup(&iflist[i], group); err != nil {
				return fmt.Errorf("error joining %v to group %v: %v",
					iflist[i].Name, group.IP.String(), err)
			}
		}
	}

	return nil
}

func Announcer(listen_addr string, group_ip net.IP) error {
	c, err := net.ListenPacket("udp6", listen_addr)
	if err != nil {
		return fmt.Errorf("error listening on %v: %v", listen_addr, err)
	}
	defer c.Close()

	pc := ipv6.NewPacketConn(c)
	err = JoinGroup(pc, &net.UDPAddr{IP: group_ip})
	if err != nil {
		return err
	}

	b := make([]byte, 1500)

	// BEGIN - Receive Solicitations Loop
	for {
		// Read a datagram from the socket
		n, _, src, err := pc.ReadFrom(b)
		if err != nil {
			return fmt.Errorf("error reading from PacketConn: %v", err)
		}
		fmt.Printf("Message from %v\n%v\n", src.String(), string(b[:n]))

		// Parse the message, it should be JSON
		var s Solicitation
		err = json.Unmarshal(b[:n], &s)
		if err != nil {
			fmt.Printf("error parsing message json: %v\n", err)
			continue
		}

		// Validate the data in the solicitation
		inform_ip := net.ParseIP(s.Inform)
		if inform_ip == nil {
			fmt.Printf("message inform IP could not be parsed: %v\n", s.Inform)
			continue
		}
		if inform_ip.To4() != nil {
			fmt.Printf("message inform IP was not IPv6: %v\n", s.Inform)
			continue
		}

		// Figure out what IP to respond with, and prepare the response
		response_ip, err := SelectMatchingIP(inform_ip, nil)
		if err != nil {
			fmt.Printf("unable to select matching host ip: %v", err)
		}

		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			fmt.Println("unable to get hostname")
			continue
		}

		response := Response{
			IPStr:    response_ip.String(),
			Hostname: hostname,
		}
		response_buf, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("error marshalling response object: %v\n", err)
			continue
		}
		if len(response_buf) > 1400 {
			fmt.Printf("warning, length of response is %v bytes", len(response_buf))
		}
		fmt.Printf("Response:\n%v\n", string(response_buf))

		// Send the response datagram
		dest_str := net.JoinHostPort(inform_ip.String(), fmt.Sprint(s.ResponsePort))
		resp_conn, err := net.Dial("udp", dest_str)
		if err != nil {
			fmt.Printf("error opening socket to %v: %v\n", dest_str, err)
			continue
		}
		// resp_conn.SetDeadline(time.Now().Add(1 * time.Second))
		_, err = resp_conn.Write(response_buf)
		if err != nil {
			fmt.Printf("error writing response to socket: %v\n", err)
		}
		resp_conn.Close()
	}
	// END - Receive Solicitations Loop
}

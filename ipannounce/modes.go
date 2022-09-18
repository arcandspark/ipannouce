package ipannounce

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

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

func Solicitor(listen_addr string, inform_ip net.IP, group_ip net.IP, annport uint64) error {
	c, err := net.ListenPacket("udp6", listen_addr)
	if err != nil {
		return fmt.Errorf("error listening on %v: %v", listen_addr, err)
	}
	defer c.Close()

	// Prepare solicitation message
	_, response_port_str, _ := net.SplitHostPort(listen_addr)
	response_port, _ := strconv.ParseUint(response_port_str, 10, 64)
	sol := Solicitation{
		Inform:       inform_ip.String(),
		ResponsePort: response_port,
	}
	sol_buf, err := json.Marshal(sol)
	if err != nil {
		return fmt.Errorf("error marshalling solicitation object: %v", err)
	}

	// Send message to the group
	group_dst := net.JoinHostPort(group_ip.String(), fmt.Sprint(annport))
	gc, err := net.Dial("udp", group_dst)
	if err != nil {
		return fmt.Errorf("error dialing group %v: %v", group_dst, err)
	}
	// gc.SetWriteDeadline(time.Now().Add(1 * time.Second))
	_, err = gc.Write(sol_buf)
	if err != nil {
		return fmt.Errorf("error sending solicitation to group: %v", err)
	}

	b := make([]byte, 1500)
	end_listen := time.Now().Add(10 * time.Second)
	for time.Now().Before(end_listen) {
		// Read a datagram from the socket
		c.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, _, _ := c.ReadFrom(b)
		// TODO - can we catch err above and determine if it is a timeout?
		// knowing golang the answer is probably look for the work "timeout" in the error string

		var resp Response
		err = json.Unmarshal(b[:n], &resp)
		if err != nil {
			fmt.Printf("error parsing message json: %v\n", err)
			continue
		}

		fmt.Printf("%16v%v", resp.Hostname, resp.IPStr)
	}

	return nil
}

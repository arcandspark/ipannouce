# IP Announce

This program announces a a host's hostname and IP address to a specified IPv6 multicast group. The intent of the program is to support discovery of servers that are using SLAAC to configure interface IPs.

**NOTE** - This program only supports IPv6.

I created this instead of using mDNS (avahi-daemon) because avahi currently (2022-09-17) does not respond to queries with all interface IPs. It seems to randomly send back one of the interface IPs, which can be a link-local, GUA, or ULA.

ipannounce accepts a "selector IP" that is compared against the host's assigned IPs in order to determine what IP address to use as the source of the annoucement. The host IP that shares the most consecutive bits with the selector IP, starting with the MSB, will be the chosen source IP. See the `SelectSourceIP` function for a complete example.

A regex can also be provided to filter which interfaces have their addresses considered as a potential source. I added this to filter out docker bridges on the same subnet as a host interface.

The listener is fairly simple and just logs the announced IPs. I will likely extend it to update DNS.

## Usage

Modes:

* sol - solicitor, sends request to group for IP announcements
* ann - announcer, sends own IP and hostname info back to solicitors

```
ipannouce -mode <sol|ann>        # operation mode
    [-group <multicast_addr>]    # multicast group, default ff15::793e:287a
    [-solport <solicitor_port>]  # port solicitor listens on, default 5190
    [-annport <announcer_port>]  # port announcer listens on, default 5190
    [mode specific options]
```

**solicitor mode options**

```
    -selector <ip_address>       # required - an address used to select the solicitor's address
    [-ifpat <regex>]             # solicitor interface name must match this to be selected
```


## Communication Flow

Actors:

* SOL - "solicitor", a host who want's to know hostnames and IPs within a specific prefix for other nodes on the network
* ANN - "announcer", a host that we may not know a unicast address for yet

Flow:

* ANN starts and listens on all addresses udp/5190
* ANN joins multicast group ff15::793e:287a
* SOL starts and listens on desired addresses udp/5190

periodically...

* SOL sends group a message indicating the address on which it wants a reply

```
{
    "inform": "fd14:225f:526a:15:2187:243c:c398:35d6",
    "response_port": 5190
}
```

* ANN uses the "inform" address to select the IP it will respond with.
* ANN chooses its address that has the most matching bits with the "inform" address, starting with the MSB.
* ANN sends a response back directly to the "inform" address, udp/5190:

```
{
    "address": "fd14:225f:526a:15:734b:34b8:1220:45da",
    "hostname": "someserver49"
}
```

## Possible planned features

* Send all interface IPs
* Cryptographically sign the solicitation and announcement
# IP Announce

This program announces a a host's hostname and IP address to a specified IPv6 multicast group. The intent of the program is to support discovery of servers that are using SLAAC to configure interface IPs.

I created this instead of using mDNS (avahi-daemon) because avahi currently (2022-09-17) does not respond to queries with all interface IPs. It seems to randomly send back one of the interface IPs, which can be a link-local, GUA, or ULA. Also, this program announces IPs independent of being queried, which is a closer fit to my use-case than needing to query explicit hostnames.

ipannounce accepts a "selector IP" that is compared against the host's assigned IPs in order to determine what IP address to use as the source of the annoucement. The host IP that shares the most consecutive bits with the selector IP, starting with the MSB, will be the chosen source IP. See the `SelectSourceIP` function for a complete example.

A regex can also be provided to filter which interfaces have their addresses considered as a potential source. I added this to filter out docker bridges on the same subnet as a host interface.

The listener is fairly simple and just logs the announced IPs. I will likely extend it to update DNS.

### Possible planned features

* Send all interface IPs
* Cryptographically sign the announcement
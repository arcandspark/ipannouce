package ipannounce

type Solicitation struct {
	// IP that the response should be sent to.
	// Also, the IP which will be used to select the announced IP.
	Inform string `json:"inform"`
	// UDP port to which the response should be sent
	ResponsePort uint `json:"response_port"`
}

type Response struct {
	// Announced IP address as a string
	IPStr string `json:"ipstr"`
	// Responder's self-reported host name
	Hostname string `json:"hostname"`
}

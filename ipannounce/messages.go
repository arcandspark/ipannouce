package ipannounce

type Announcement struct {
	IPStr    string `json:ipstr`
	Hostname string `json:hostname`
}

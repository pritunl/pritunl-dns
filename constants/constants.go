package constants

import (
	"time"
)

var (
	DefaultDnsServers = []string{"8.8.8.8:53", "8.8.4.4:53"}
)

const (
	DefaultDatabaseSyncRate = 30 * time.Second
)

package gateway

type Limits struct {
	MaxConnections        int
	MaxConnectionsPerUser int
	CommandRatePerSecond  int
	CommandBurst          int
}

func (l Limits) WithDefaults() Limits {
	if l.MaxConnections <= 0 {
		l.MaxConnections = 10000
	}
	if l.MaxConnectionsPerUser <= 0 {
		l.MaxConnectionsPerUser = 5
	}
	if l.CommandRatePerSecond <= 0 {
		l.CommandRatePerSecond = 20
	}
	if l.CommandBurst <= 0 {
		l.CommandBurst = 40
	}
	if l.CommandBurst < l.CommandRatePerSecond {
		l.CommandBurst = l.CommandRatePerSecond
	}

	return l
}
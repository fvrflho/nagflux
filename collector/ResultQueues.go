package collector

import "github.com/ConSol-Monitoring/nagflux/data"

type ResultQueues map[data.Target]chan Printable

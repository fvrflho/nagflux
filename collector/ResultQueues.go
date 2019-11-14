package collector

import "github.com/ConSol/nagflux/data"

type ResultQueues map[data.Target]chan Printable

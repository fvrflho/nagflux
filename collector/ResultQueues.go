package collector

import "github.com/fvrflho/nagflux/data"

type ResultQueues map[data.Target]chan Printable

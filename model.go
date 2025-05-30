package linode

import (
	"fmt"
	"time"

	"github.com/libdns/libdns"
	"github.com/linode/linodego"
)

type DNS struct {
	Record libdns.RR
	ID     int
}

func (d DNS) RR() libdns.RR {
	return d.Record
}

func fromRecord(record libdns.Record, id int) DNS {
	rr := record.RR()
	return DNS{
		Record: rr,
		ID:     id,
	}
}

func fromLinodeRecord(entry linodego.DomainRecord) DNS {
	return DNS{
		Record: libdns.RR{
			Name: entry.Name,
			TTL:  time.Duration(entry.TTLSec) * time.Second,
			Type: string(entry.Type),
			Data: entry.Target,
		},
		ID: entry.ID,
	}
}

func idFromRecord(record libdns.Record) (int, error) {
	if dns, ok := record.(DNS); ok {
		return dns.ID, nil
	} else {
		return 0, fmt.Errorf("could not get id from record")
	}
}

package linode

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/libdns/libdns"
	"github.com/linode/linodego"
)

func (p *Provider) init(ctx context.Context) {
	p.once.Do(func() {
		p.client = linodego.NewClient(http.DefaultClient)
		if p.APIToken != "" {
			p.client.SetToken(p.APIToken)
		}
		if p.APIURL != "" {
			p.client.SetBaseURL(p.APIURL)
		}
		if p.APIVersion != "" {
			p.client.SetAPIVersion(p.APIVersion)
		}
	})
}

func (p *Provider) getDomainIDByZone(ctx context.Context, zone string) (int, error) {
	f := linodego.Filter{}
	f.AddField(linodego.Eq, "domain", libdns.AbsoluteName(zone, ""))
	filter, err := f.MarshalJSON()
	if err != nil {
		return 0, err
	}
	listOptions := linodego.NewListOptions(0, string(filter))
	domains, err := p.client.ListDomains(ctx, listOptions)
	if err != nil {
		return 0, fmt.Errorf("could not list domains: %v", err)
	}
	if len(domains) == 0 {
		return 0, fmt.Errorf("could not find the domain provided")
	}
	return domains[0].ID, nil
}

func (p *Provider) listDomainRecords(ctx context.Context, zone string, domainID int) ([]libdns.Record, error) {
	listOptions := linodego.NewListOptions(0, "")
	linodeRecords, err := p.client.ListDomainRecords(ctx, domainID, listOptions)
	if err != nil {
		return nil, fmt.Errorf("could not list domain records: %v", err)
	}
	records := make([]libdns.Record, 0, len(linodeRecords))
	for _, linodeRecord := range linodeRecords {
		records = append(records, fromLinodeRecord(linodeRecord))
	}
	return records, nil
}

func (p *Provider) createOrUpdateDomainRecord(ctx context.Context, zone string, domainID int, record libdns.Record) (libdns.Record, error) {
	_, err := idFromRecord(record)
	if err != nil {
		addedRecord, err := p.createDomainRecord(ctx, zone, domainID, record)
		if err != nil {
			return nil, err
		}
		return addedRecord, nil
	}
	updatedRecord, err := p.updateDomainRecord(ctx, zone, domainID, record)
	if err != nil {
		return nil, err
	}
	return updatedRecord, nil
}

func (p *Provider) createDomainRecord(ctx context.Context, zone string, domainID int, record libdns.Record) (libdns.Record, error) {
	rr := record.RR()
	addedLinodeRecord, err := p.client.CreateDomainRecord(ctx, domainID, linodego.DomainRecordCreateOptions{
		Type:   linodego.DomainRecordType(rr.Type),
		Name:   libdns.RelativeName(rr.Name, zone),
		Target: rr.Data,
		TTLSec: int(rr.TTL.Seconds()),
	})
	if err != nil {
		return nil, err
	}
	return mergeWithExistingLibdns(zone, record, *addedLinodeRecord), nil
}

func (p *Provider) updateDomainRecord(ctx context.Context, zone string, domainID int, record libdns.Record) (libdns.Record, error) {
	recordID, err := idFromRecord(record)
	if err != nil {
		return nil, err
	}
	rr := record.RR()
	updatedLinodeRecord, err := p.client.UpdateDomainRecord(ctx, domainID, recordID, linodego.DomainRecordUpdateOptions{
		Type:   linodego.DomainRecordType(rr.Type),
		Name:   libdns.RelativeName(rr.Name, zone),
		Target: rr.Data,
		TTLSec: int(rr.TTL.Seconds()),
	})
	if err != nil {
		return nil, err
	}
	return mergeWithExistingLibdns(zone, record, *updatedLinodeRecord), nil
}

func (p *Provider) deleteDomainRecord(ctx context.Context, domainID int, record libdns.Record) error {
	recordID, err := idFromRecord(record)
	if err != nil {
		return err
	}
	return p.client.DeleteDomainRecord(ctx, domainID, recordID)
}

func convertToLibdns(zone string, linodeRecord linodego.DomainRecord) libdns.Record {
	return mergeWithExistingLibdns(zone, nil, linodeRecord)
}

func mergeWithExistingLibdns(zone string, existingRecord libdns.Record, linodeRecord linodego.DomainRecord) libdns.Record {
	if existingRecord == nil {
		existingRecord = libdns.RR{
			Type: string(linodeRecord.Type),
			Name: libdns.RelativeName(linodeRecord.Name, zone),
			Data: linodeRecord.Target,
			TTL:  time.Duration(linodeRecord.TTLSec) * time.Second,
		}
	}
	return existingRecord
}

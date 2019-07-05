package dns

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDnsSRVRecordSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsSRVRecordSetRead,
		Schema: map[string]*schema.Schema{
			"service": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"proto": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"srv": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"weight": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"target": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func dataSourceDnsSRVRecordSetRead(d *schema.ResourceData, meta interface{}) error {
	service := d.Get("service").(string)
	proto := d.Get("proto").(string)
	domain := d.Get("domain").(string)

	cname, records, err := net.LookupSRV(service, proto, domain)
	if err != nil {
		return fmt.Errorf("error looking up SRV records for %q: %s", cname, err)
	}

	// Sort by priority and weight, might not be needed
	// sort.Sort(byPriorityWeight(records))

	srv := make([]map[string]interface{}, len(records))
	for i, record := range records {
		srv[i] = map[string]interface{}{
			"target":   record.Target,
			"port":     int(record.Port),
			"priority": int(record.Priority),
			"weight":   int(record.Weight),
		}
	}

	if err = d.Set("srv", srv); err != nil {
		return err
	}

	//id := fmt.Sprintf("%q.%q.%q", service, proto, domain)
	d.SetId(cname)

	return nil
}

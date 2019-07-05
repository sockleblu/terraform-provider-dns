package dns

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/miekg/dns"
)

func resourceDnsSRVRecordSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsSRVRecordSetCreate,
		Read:   resourceDnsSRVRecordSetRead,
		Update: resourceDnsSRVRecordSetUpdate,
		Delete: resourceDnsSRVRecordSetDelete,
		Importer: &schema.ResourceImporter{
			State: resourceDnsImport,
		},

		Schema: map[string]*schema.Schema{
			"service": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				//ValidateFunc: validateZone,
			},
			"proto": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				//ValidateFunc: validateName,
			},
			"zone": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateZone,
			},
			"srv": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"weight": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceDnsSRVRecordSetHash,
			},
			"ttl": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  3600,
			},
		},
	}
}

func resourceDnsSRVRecordSetCreate(d *schema.ResourceData, meta interface{}) error {

	d.SetId(resourceFQDN(d))

	return resourceDnsSRVRecordSetUpdate(d, meta)
}

func resourceDnsSRVRecordSetRead(d *schema.ResourceData, meta interface{}) error {

	answers, err := resourceDnsRead(d, meta, dns.TypeSRV)
	if err != nil {
		return err
	}

	if len(answers) > 0 {

		var ttl sort.IntSlice

		srv := schema.NewSet(resourceDnsSRVRecordSetHash, nil)
		for _, record := range answers {
			switch r := record.(type) {
			case *dns.SRV:
				m := map[string]interface{}{
					"priority": int(r.Priority),
					"weight":   int(r.Weight),
					"port":     int(r.Port),
					"target":   r.Target,
				}
				srv.Add(m)
				ttl = append(ttl, int(r.Hdr.Ttl))
			default:
				return fmt.Errorf("didn't get a SRV record")
			}
		}
		sort.Sort(ttl)

		d.Set("srv", srv)
		d.Set("ttl", ttl[0])
	} else {
		d.SetId("")
	}

	return nil
}

func resourceDnsSRVRecordSetUpdate(d *schema.ResourceData, meta interface{}) error {

	if meta != nil {

		ttl := d.Get("ttl").(int)
		fqdn := resourceFQDN(d)

		msg := new(dns.Msg)

		msg.SetUpdate(d.Get("zone").(string))

		if d.HasChange("srv") {
			o, n := d.GetChange("srv")
			os := o.(*schema.Set)
			ns := n.(*schema.Set)
			remove := os.Difference(ns).List()
			add := ns.Difference(os).List()

			// Loop through all the old addresses and remove them
			for _, srv := range remove {
				m := srv.(map[string]interface{})
				rr_remove, _ := dns.NewRR(fmt.Sprintf("%s %d SRV %d %d %d %s", fqdn, ttl, m["priority"], m["weight"], m["port"], m["target"]))
				msg.Remove([]dns.RR{rr_remove})
			}
			// Loop through all the new addresses and insert them
			for _, srv := range add {
				m := srv.(map[string]interface{})
				rr_insert, _ := dns.NewRR(fmt.Sprintf("%s %d SRV %d %d %d %s", fqdn, ttl, m["priority"], m["weight"], m["port"], m["target"]))
				msg.Insert([]dns.RR{rr_insert})
			}

			r, err := exchange(msg, true, meta)
			if err != nil {
				d.SetId("")
				return fmt.Errorf("Error updating DNS record: %s", err)
			}
			if r.Rcode != dns.RcodeSuccess {
				d.SetId("")
				return fmt.Errorf("Error updating DNS record: %v (%s)", r.Rcode, dns.RcodeToString[r.Rcode])
			}
		}

		return resourceDnsSRVRecordSetRead(d, meta)
	} else {
		return fmt.Errorf("update server is not set")
	}
}

func resourceDnsSRVRecordSetDelete(d *schema.ResourceData, meta interface{}) error {

	return resourceDnsDelete(d, meta, dns.TypeSRV)
}

func resourceDnsSRVRecordSetHash(v interface{}) int {
	var buf bytes.Buffer

	if m, ok := v.(map[string]interface{}); ok {
		buf.WriteString(fmt.Sprintf("%d-", m["priority"].(int)))
		buf.WriteString(fmt.Sprintf("%d-", m["weight"].(int)))
		buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
		buf.WriteString(fmt.Sprintf("%s-", m["target"].(string)))
	}

	return hashcode.String(buf.String())
}

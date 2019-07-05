package dns

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/miekg/dns"
)

func TestAccDnsSRVRecordSet_Basic(t *testing.T) {

	var service, proto, zone string
	resourceName := "dns_srv_record_set.foo"
	resourceRoot := "dns_srv_record_set.root"

	deleteSRVRecordSet := func() {
		meta := testAccProvider.Meta()

		msg := new(dns.Msg)

		msg.SetUpdate(zone)

		var name strings.Builder
		name.WriteString(service)
		name.WriteString(proto)

		fqdn := testResourceFQDN(name.String(), zone)

		rr_remove, _ := dns.NewRR(fmt.Sprintf("%s 0 SRV", fqdn))
		msg.RemoveRRset([]dns.RR{rr_remove})

		r, err := exchange(msg, true, meta)
		if err != nil {
			t.Fatalf("Error deleting DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("Error deleting DNS record: %v", r.Rcode)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDnsSRVRecordSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDnsSRVRecordSet_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "srv.#", "1"),
					testAccCheckDnsSRVRecordSetExists(t, resourceName, []interface{}{map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test.example.org."}}, &service, &proto, &zone),
				),
			},
			{
				Config: testAccDnsSRVRecordSet_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "srv.#", "2"),
					testAccCheckDnsSRVRecordSetExists(t, resourceName, []interface{}{map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test.example.org."}, map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test2.example.org."}}, &service, &proto, &zone),
				),
			},
			{
				PreConfig: deleteSRVRecordSet,
				Config:    testAccDnsSRVRecordSet_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "srv.#", "2"),
					testAccCheckDnsSRVRecordSetExists(t, resourceName, []interface{}{map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test2.example.org."}, map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test.example.org."}}, &service, &proto, &zone),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDnsSRVRecordSet_root,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceRoot, "srv.#", "1"),
					testAccCheckDnsSRVRecordSetExists(t, resourceRoot, []interface{}{map[string]interface{}{"priority": 10, "weight": 10, "port": 10, "target": "test.example.org."}}, &service, &proto, &zone),
				),
			},
			{
				ResourceName:      resourceRoot,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckDnsSRVRecordSetDestroy(s *terraform.State) error {
	return testAccCheckDnsDestroy(s, "dns_srv_record_set", dns.TypeSRV)
}

func testAccCheckDnsSRVRecordSetExists(t *testing.T, n string, srv []interface{}, service, proto, zone *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		*service = rs.Primary.Attributes["service"]
		*proto = rs.Primary.Attributes["proto"]
		*zone = rs.Primary.Attributes["zone"]

		var name strings.Builder
		name.WriteString(*service)
		name.WriteString(*proto)

		fqdn := testResourceFQDN(name.String(), *zone)

		meta := testAccProvider.Meta()

		msg := new(dns.Msg)
		msg.SetQuestion(fqdn, dns.TypeSRV)
		r, err := exchange(msg, false, meta)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("Error querying DNS record")
		}

		existing := schema.NewSet(resourceDnsSRVRecordSetHash, nil)
		expected := schema.NewSet(resourceDnsSRVRecordSetHash, srv)
		for _, record := range r.Answer {
			switch r := record.(type) {
			case *dns.SRV:
				m := map[string]interface{}{
					"priority": int(r.Priority),
					"weight":   int(r.Weight),
					"port":     int(r.Port),
					"target":   r.Target,
				}
				existing.Add(m)
			default:
				return fmt.Errorf("didn't get a SRV record")
			}
		}
		if !existing.Equal(expected) {
			return fmt.Errorf("DNS record differs: expected %v, found %v", expected, existing)
		}
		return nil
	}
}

var testAccDnsSRVRecordSet_basic = fmt.Sprintf(`
  resource "dns_srv_record_set" "foo" {
    service = "test"
    proto = "tcp"
    zone = "example.com."
    srv {
      priority = 10
      weight   = 10
      port     = 3306
      target   = "mysql.example.org."
    }
    ttl = 300
  }`)

var testAccDnsSRVRecordSet_update = fmt.Sprintf(`
  resource "dns_mx_record_set" "foo" {
    service = "test"
    proto = "tcp"
    zone = "example.com."
    srv {
      priority = 10
      weight   = 10
      port     = 3306
      target   = "mysql1.example.org."
    }
    srv {
      priority = 20
      weight   = 10
      port     = 3306
      target   = "mysql2.example.org."
    }
    ttl = 300
  }`)

var testAccDnsSRVRecordSet_root = fmt.Sprintf(`
  resource "dns_srv_record_set" "root" {
    service = "test"
    proto = "tcp"
    zone = "example.com."
    srv {
      priority = 10
      weight   = 10
      port     = 3306
      target   = "mysql.example.org."
    }
    ttl = 300
  }`)

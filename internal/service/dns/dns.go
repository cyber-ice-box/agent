package dns

import (
	"bytes"
	"context"
	"fmt"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	"gitlab.com/cyber-ice-box/agent/internal/model/service"
	"slices"
	"strings"
	"text/template"
)

const (
	dnsName       = "dns-server"
	dnsConfigName = "dns-configmap"

	image            = "coredns/coredns:1.10.0"
	coreFile         = "Corefile"
	zoneFile         = "zonefile"
	recordsListLabel = "recordsList"
	coreFileContent  = `. {
    file zonefile
    prometheus     # enable metrics
    errors         # show errors
    log            # enable query logs
}
`
	zonePrefixContent = `$ORIGIN .
@   3600 IN SOA sns.dns.icann.org. noc.dns.icann.org. (
                2017042745 ; serial
                7200       ; refresh (2 hours)
                3600       ; retry (1 hour)
                1209600    ; expire (2 weeks)
                3600       ; minimum (1 hour)
                )

{{range .}}{{.Name}} IN {{.Type}} {{.Data}}
{{end}}
`
)

type k8s interface {
	ApplyDeployment(ctx context.Context, config delivery.ApplyDeploymentConfig) error
	ResetDeployment(ctx context.Context, name, namespace string) error

	ApplyConfigMap(ctx context.Context, name, namespace string, data map[string]string) error
	GetConfigMapData(ctx context.Context, name, namespace string) (map[string]string, error)
}

type Service struct {
	k8s          k8s
	labNamespace string
	records      []service.RecordConfig
}

func New(k8s k8s) *Service {
	return &Service{k8s: k8s}
}

func (dns *Service) Create(ctx context.Context, labId, ip string) error {
	dns.labNamespace = labId
	dns.records = make([]service.RecordConfig, 0)

	config, err := dns.generateZoneConfig()
	if err != nil {
		return err
	}

	if err = dns.setZoneConfig(ctx, config); err != nil {
		return err
	}

	return dns.k8s.ApplyDeployment(ctx, delivery.ApplyDeploymentConfig{
		Name:      dnsName,
		Namespace: labId,
		Label: delivery.Label{
			Key:   "labId",
			Value: labId,
		},
		Image: image,
		Ip:    ip,
		Resources: service.ResourcesConfig{
			Requests: service.ResourceConfig{
				Memory: "50Mi",
				CPU:    "300m",
			},
			Limit: service.ResourceConfig{
				Memory: "50Mi",
				CPU:    "300m",
			},
		},
		Envs:  nil,
		Args:  []string{"-conf", fmt.Sprintf("/%s", coreFile)},
		Ports: nil,
		Volumes: []delivery.Volume{{
			Name:          dnsName,
			ConfigMapName: dnsConfigName,
			Mounts: []delivery.Mount{
				{
					MountPath: fmt.Sprintf("/%s", coreFile),
					SubPath:   coreFile,
				},
				{
					MountPath: fmt.Sprintf("/%s", zoneFile),
					SubPath:   zoneFile,
				}},
		}},
		Privileged:     false,
		CapAdds:        nil,
		ReadinessProbe: nil,
	})
}

func (dns *Service) RefreshRecords(ctx context.Context, labId string, records []service.RecordConfig) error {
	dns.labNamespace = labId
	dns.records = make([]service.RecordConfig, 0)

	if err := dns.getRecords(ctx); err != nil {
		return err
	}

	if err := dns.addRecords(records); err != nil {
		return err
	}
	config, err := dns.generateZoneConfig()
	if err != nil {
		return err
	}

	if err = dns.setZoneConfig(ctx, config); err != nil {
		return err
	}

	return dns.reset(ctx)
}

func (dns *Service) generateZoneConfig() (string, error) {
	var tpl bytes.Buffer

	t, err := template.New("config").Parse(zonePrefixContent)
	if err != nil {
		panic(err)
	}
	err = t.Execute(&tpl, dns.records)
	if err != nil {
		panic(err)
	}
	return tpl.String(), nil
}

func (dns *Service) setZoneConfig(ctx context.Context, config string) error {
	return dns.k8s.ApplyConfigMap(ctx, dnsConfigName, dns.labNamespace, map[string]string{
		coreFile:         coreFileContent,
		zoneFile:         config,
		recordsListLabel: dns.recordsToStr(dns.records),
	})
}

func (dns *Service) getRecords(ctx context.Context) error {
	data, err := dns.k8s.GetConfigMapData(ctx, dnsConfigName, dns.labNamespace)
	if err != nil {
		return err
	}

	if len(data[recordsListLabel]) == 0 {
		return nil
	}

	rData := strings.Split(data[recordsListLabel], ";")
	for _, r := range rData {
		rItem := strings.Split(r, ",")
		dns.records = append(dns.records, service.RecordConfig{
			Type: rItem[0],
			Name: rItem[1],
			Data: rItem[2],
		})
	}

	return nil
}

func (dns *Service) recordsToStr(records []service.RecordConfig) string {
	var strRecords []string

	for _, r := range records {
		strRecords = append(strRecords, strings.Join([]string{r.Type, r.Name, r.Data}, ","))
	}

	return strings.Join(strRecords, ";")
}

func (dns *Service) reset(ctx context.Context) error {
	return dns.k8s.ResetDeployment(ctx, dnsName, dns.labNamespace)
}

func (dns *Service) addRecords(records []service.RecordConfig) error {
	var errs error

	for _, r := range records {
		if slices.Contains(dns.records, r) {
			errs = multierror.Append(errs, fmt.Errorf("record for %s IN %s %s already exists", r.Name, r.Type, r.Data))
		} else {
			dns.records = append(dns.records, r)
		}
	}

	return errs
}

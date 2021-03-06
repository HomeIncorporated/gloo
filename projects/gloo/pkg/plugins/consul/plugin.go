package consul

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/rotisserie/eris"

	"github.com/solo-io/gloo/projects/gloo/pkg/discovery"

	"github.com/solo-io/gloo/projects/gloo/pkg/upstreams/consul"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	"github.com/solo-io/gloo/projects/gloo/pkg/plugins"
	"github.com/solo-io/gloo/projects/gloo/pkg/xds"
)

var _ discovery.DiscoveryPlugin = new(plugin)

var (
	DefaultDnsAddress         = "127.0.0.1:8600"
	DefaultDnsPollingInterval = 5 * time.Second
)

type plugin struct {
	client             consul.ConsulWatcher
	resolver           DnsResolver
	dnsPollingInterval time.Duration
}

func (p *plugin) Resolve(u *v1.Upstream) (*url.URL, error) {
	consulSpec, ok := u.UpstreamType.(*v1.Upstream_Consul)
	if !ok {
		return nil, nil
	}

	spec := consulSpec.Consul

	// default to first datacenter
	var dc string
	if len(spec.DataCenters) > 0 {
		dc = spec.DataCenters[0]
	}

	instances, _, err := p.client.Service(spec.ServiceName, "", &api.QueryOptions{Datacenter: dc, RequireConsistent: true})
	if err != nil {
		return nil, eris.Wrapf(err, "getting service from catalog")
	}

	scheme := "http"
	if u.SslConfig != nil {
		scheme = "https"
	}

	for _, inst := range instances {
		if (len(spec.InstanceTags) == 0) || matchTags(spec.InstanceTags, inst.ServiceTags) {
			ipAddresses, err := getIpAddresses(context.TODO(), inst.ServiceAddress, p.resolver)
			if err != nil {
				return nil, err
			}
			if len(ipAddresses) == 0 {
				return nil, eris.Errorf("DNS result for %s returned an empty list of IPs", inst.ServiceAddress)
			}
			// arbitrarily default to the first result
			ipAddr := ipAddresses[0]
			return url.Parse(fmt.Sprintf("%v://%v:%v", scheme, ipAddr, inst.ServicePort))
		}
	}

	return nil, eris.Errorf("service with name %s and tags %v not found", spec.ServiceName, spec.InstanceTags)
}

func NewPlugin(client consul.ConsulWatcher, resolver DnsResolver, dnsPollingInterval *time.Duration) *plugin {
	pollingInterval := DefaultDnsPollingInterval
	if dnsPollingInterval != nil {
		pollingInterval = *dnsPollingInterval
	}
	return &plugin{client: client, resolver: resolver, dnsPollingInterval: pollingInterval}
}

func (p *plugin) Init(params plugins.InitParams) error {
	return nil
}

func (p *plugin) ProcessUpstream(params plugins.Params, in *v1.Upstream, out *envoyapi.Cluster) error {
	_, ok := in.UpstreamType.(*v1.Upstream_Consul)
	if !ok {
		return nil
	}

	// consul upstreams use EDS
	xds.SetEdsOnCluster(out)

	return nil
}

func matchTags(t1, t2 []string) bool {
	if len(t1) != len(t2) {
		return false
	}
	for _, tag1 := range t1 {
		var found bool
		for _, tag2 := range t2 {
			if tag1 == tag2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

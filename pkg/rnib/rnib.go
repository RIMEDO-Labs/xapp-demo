package rnib

import (
	"context"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

var log = logging.GetLogger("xapp-demo", "rnib")

type TopoClient interface {
	WatchE2Connections(ctx context.Context, ch chan topoapi.Event) error
}

type Options struct {
	TopoAddress string
	TopoPort    int
}

type Cell struct {
	CGI      string
	CellType string
}

func NewClient(options Options) (Client, error) {
	sdkClient, err := toposdk.NewClient(
		toposdk.WithTopoAddress(
			options.TopoAddress,
			options.TopoPort,
		),
	)
	if err != nil {
		return Client{}, err
	}
	return Client{
		client: sdkClient,
	}, nil
}

type Client struct {
	client toposdk.Client
}

func getControlRelationFilter() *topoapi.Filters {
	controlRelationFilter := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: topoapi.CONTROLS,
				},
			},
		},
	}
	return controlRelationFilter
}

func (c *Client) WatchE2Connections(ctx context.Context, ch chan topoapi.Event) error {
	err := c.client.Watch(ctx, ch, toposdk.WithWatchFilters(getControlRelationFilter()))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetE2CellFilter() *topoapi.Filters {
	cellEntityFilter := &topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_In{
				In: &topoapi.InFilter{
					Values: []string{topoapi.E2CELL},
				},
			},
		},
	}
	return cellEntityFilter
}

func (c *Client) GetE2NodeAspects(ctx context.Context, nodeID topoapi.ID) (*topoapi.E2Node, error) {
	object, err := c.client.Get(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	e2Node := &topoapi.E2Node{}
	err = object.GetAspect(e2Node)

	return e2Node, err

}

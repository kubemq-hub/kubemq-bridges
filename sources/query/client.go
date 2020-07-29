package query

import (
	"context"
	"fmt"
	"github.com/kubemq-hub/kubemq-bridges/config"
	"github.com/kubemq-hub/kubemq-bridges/middleware"
	"github.com/kubemq-hub/kubemq-bridges/pkg/logger"

	"github.com/nats-io/nuid"

	"github.com/kubemq-io/kubemq-go"
)

type Client struct {
	name   string
	opts   options
	client *kubemq.Client
	log    *logger.Logger
	target middleware.Middleware
}

func New() *Client {
	return &Client{}

}
func (c *Client) Name() string {
	return c.name
}
func (c *Client) Init(ctx context.Context, cfg config.Spec) error {
	c.name = cfg.Name
	c.log = logger.NewLogger(cfg.Name)
	var err error
	c.opts, err = parseOptions(cfg.Properties)
	if err != nil {
		return err
	}
	c.client, _ = kubemq.NewClient(ctx,
		kubemq.WithAddress(c.opts.host, c.opts.port),
		kubemq.WithClientId(c.opts.clientId),
		kubemq.WithTransportType(kubemq.TransportTypeGRPC),
		kubemq.WithAuthToken(c.opts.authToken),
		kubemq.WithMaxReconnects(c.opts.maxReconnects),
		kubemq.WithAutoReconnect(c.opts.autoReconnect),
		kubemq.WithReconnectInterval(c.opts.reconnectIntervalSeconds))
	return nil
}

func (c *Client) Start(ctx context.Context, target middleware.Middleware) error {
	c.target = target
	group := nuid.Next()
	if c.opts.group != "" {
		group = c.opts.group
	}
	for i := 0; i < c.opts.concurrency; i++ {
		errCh := make(chan error, 1)
		queriesCh, err := c.client.SubscribeToQueries(ctx, c.opts.channel, group, errCh)
		if err != nil {
			return fmt.Errorf("error on subscribing to query channel, %w", err)
		}
		go func(ctx context.Context, queryCh <-chan *kubemq.QueryReceive, errCh chan error) {
			c.run(ctx, queriesCh, errCh)
		}(ctx, queriesCh, errCh)
	}
	return nil
}

func (c *Client) run(ctx context.Context, queryCh <-chan *kubemq.QueryReceive, errCh chan error) {
	for {
		select {
		case query := <-queryCh:

			go func(q *kubemq.QueryReceive) {
				var queryResponse *kubemq.Response
				queryResponse, err := c.processQuery(ctx, query)
				if err != nil {
					queryResponse = c.client.NewResponse().
						SetRequestId(query.Id).
						SetResponseTo(query.ResponseTo).
						SetError(err)
				} else {
					queryResponse.
						SetRequestId(query.Id).
						SetResponseTo(query.ResponseTo)
				}
				err = queryResponse.Send(ctx)
				if err != nil {
					c.log.Errorf("error sending query response %s", err.Error())
				}
			}(query)

		case err := <-errCh:
			c.log.Errorf("error received from kuebmq server, %s", err.Error())
			return
		case <-ctx.Done():
			return

		}
	}
}

func (c *Client) processQuery(ctx context.Context, query *kubemq.QueryReceive) (*kubemq.Response, error) {
	result, err := c.target.Do(ctx, query)
	if err != nil {
		return nil, err
	}
	switch val := result.(type) {
	case *kubemq.CommandResponse:
		return c.parseCommandResponse(val), nil
	case *kubemq.QueryResponse:
		return c.parseQueryResponse(val), nil
	default:
		return c.client.NewResponse(), nil
	}
}
func (c *Client) Stop() error {
	return c.client.Close()
}

func (c *Client) parseCommandResponse(cmd *kubemq.CommandResponse) *kubemq.Response {
	resp := c.client.NewResponse().SetTags(cmd.Tags)
	if cmd.Executed {
		resp.SetExecutedAt(cmd.ExecutedAt)
	} else {
		resp.SetError(fmt.Errorf("%s", cmd.Error))
	}
	return resp
}
func (c *Client) parseQueryResponse(query *kubemq.QueryResponse) *kubemq.Response {
	resp := c.client.NewResponse().SetTags(query.Tags).SetMetadata(query.Metadata).SetBody(query.Body)
	if query.Executed {
		resp.SetExecutedAt(query.ExecutedAt)
	} else {
		resp.SetError(fmt.Errorf("%s", query.Error))
	}
	return resp
}

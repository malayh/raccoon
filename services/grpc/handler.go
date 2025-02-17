package grpc

import (
	"context"
	"errors"
	"fmt"
	"raccoon/collection"
	"raccoon/config"
	"raccoon/identification"
	"raccoon/metrics"
	pb "raccoon/proto"
	"time"

	"google.golang.org/grpc/metadata"
)

type Handler struct {
	C collection.Collector
	pb.UnimplementedEventServiceServer
}

func (h *Handler) SendEvent(ctx context.Context, req *pb.EventRequest) (*pb.EventResponse, error) {
	metadata, _ := metadata.FromIncomingContext(ctx)
	groups := metadata.Get(config.ServerWs.ConnGroupHeader)
	var group string
	if len(groups) > 0 {
		group = groups[0]
	} else {
		group = config.ServerWs.ConnGroupDefault
	}

	var id string
	ids := metadata.Get(config.ServerWs.ConnIDHeader)

	if len(ids) > 0 {
		id = ids[0]
	} else {
		return nil, errors.New("connection id header missing")
	}

	identifier := identification.Identifier{
		ID:    id,
		Group: group,
	}

	timeConsumed := time.Now()

	metrics.Increment("batches_read_total", fmt.Sprintf("status=success,conn_group=%s", identifier.Group))
	h.sendEventCounters(req.Events, identifier.Group)

	h.C.Collect(ctx, &collection.CollectRequest{
		ConnectionIdentifier: identifier,
		TimeConsumed:         timeConsumed,
		EventRequest:         req,
	})

	return &pb.EventResponse{
		Status:   pb.Status_SUCCESS,
		Code:     pb.Code_OK,
		SentTime: time.Now().Unix(),
		Data: map[string]string{
			"req_guid": req.GetReqGuid(),
		},
	}, nil

}

func (h *Handler) sendEventCounters(events []*pb.Event, group string) {
	for _, e := range events {
		metrics.Increment("events_rx_total", fmt.Sprintf("conn_group=%s,event_type=%s", group, e.Type))
	}
}

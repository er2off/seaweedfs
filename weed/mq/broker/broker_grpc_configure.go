package broker

import (
	"context"
	"fmt"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/mq/pub_balancer"
	"github.com/seaweedfs/seaweedfs/weed/mq/topic"
	"github.com/seaweedfs/seaweedfs/weed/pb/mq_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConfigureTopic Runs on any broker, but proxied to the balancer if not the balancer
// It generates an assignments based on existing allocations,
// and then assign the partitions to the brokers.
func (b *MessageQueueBroker) ConfigureTopic(ctx context.Context, request *mq_pb.ConfigureTopicRequest) (resp *mq_pb.ConfigureTopicResponse, err error) {
	if b.currentBalancer == "" {
		return nil, status.Errorf(codes.Unavailable, "no balancer")
	}
	if !b.lockAsBalancer.IsLocked() {
		proxyErr := b.withBrokerClient(false, b.currentBalancer, func(client mq_pb.SeaweedMessagingClient) error {
			resp, err = client.ConfigureTopic(ctx, request)
			return nil
		})
		if proxyErr != nil {
			return nil, proxyErr
		}
		return resp, err
	}

	t := topic.FromPbTopic(request.Topic)
	resp, err = b.readTopicConfFromFiler(t)
	if err != nil {
		glog.V(0).Infof("read topic %s conf: %v", request.Topic, err)
	} else {
		err = b.ensureTopicActiveAssignments(t, resp)
		// no need to assign directly.
		// The added or updated assignees will read from filer directly.
		// The gone assignees will die by themselves.
	}
	if err == nil && len(resp.BrokerPartitionAssignments) == int(request.PartitionCount) {
		glog.V(0).Infof("existing topic partitions %d: %+v", len(resp.BrokerPartitionAssignments), resp.BrokerPartitionAssignments)
	} else {
		resp = &mq_pb.ConfigureTopicResponse{}
		if b.Balancer.Brokers.IsEmpty()	{
			return nil, status.Errorf(codes.Unavailable, pub_balancer.ErrNoBroker.Error())
		}
		resp.BrokerPartitionAssignments = pub_balancer.AllocateTopicPartitions(b.Balancer.Brokers, request.PartitionCount)

		// save the topic configuration on filer
		if err := b.saveTopicConfToFiler(request.Topic, resp); err != nil {
			return nil, fmt.Errorf("configure topic: %v", err)
		}

		b.Balancer.OnPartitionChange(request.Topic, resp.BrokerPartitionAssignments)
	}

	glog.V(0).Infof("ConfigureTopic: topic %s partition assignments: %v", request.Topic, resp.BrokerPartitionAssignments)

	return resp, err
}

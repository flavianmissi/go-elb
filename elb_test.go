package elb_test

import (
	"launchpad.net/goamz/aws"
	"github.com/flaviamissi/go-elb"
	. "launchpad.net/gocheck"
)

type S struct {
	HTTPSuite
	elb *elb.ELB
}

var _ = Suite(&S{})

func (s *S) SetUpSuite(c *C) {
	s.HTTPSuite.SetUpSuite(c)
	auth := aws.Auth{"abc", "123"}
	s.elb = elb.New(auth, aws.Region{ELBEndpoint: testServer.URL})
}

func (s *S) TestCreateLoadBalancer(c *C) {
	testServer.PrepareResponse(200, nil, CreateLoadBalancer)
	createLB := &elb.CreateLoadBalancer{
		Name:       "testLB",
		AvailZones: []string{"us-east-1a", "us-east-1b"},
		Listeners: []elb.Listener{
			{
				InstancePort:     80,
				InstanceProtocol: "http",
				Protocol:         "http",
				LoadBalancerPort: 80,
			},
		},
	}
	resp, err := s.elb.CreateLoadBalancer(createLB)
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Action"), Equals, "CreateLoadBalancer")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testLB")
	c.Assert(values.Get("AvailabilityZones.member.1"), Equals, "us-east-1a")
	c.Assert(values.Get("AvailabilityZones.member.2"), Equals, "us-east-1b")
	c.Assert(values.Get("Listeners.member.1.InstancePort"), Equals, "80")
	c.Assert(values.Get("Listeners.member.1.InstanceProtocol"), Equals, "http")
	c.Assert(values.Get("Listeners.member.1.Protocol"), Equals, "http")
	c.Assert(values.Get("Listeners.member.1.LoadBalancerPort"), Equals, "80")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(resp.DNSName, Equals, "testlb-339187009.us-east-1.elb.amazonaws.com")
}

func (s *S) TestCreateLoadBalancerWithSubnetsAndMoreListeners(c *C) {
	testServer.PrepareResponse(200, nil, CreateLoadBalancer)
	createLB := &elb.CreateLoadBalancer{
		Name:      "testLB",
		Listeners: []elb.Listener{
			{
				InstancePort:     80,
				InstanceProtocol: "http",
				Protocol:         "http",
				LoadBalancerPort: 80,
			},
			{
				InstancePort:     8080,
				InstanceProtocol: "http",
				Protocol:         "http",
				LoadBalancerPort: 8080,
			},
		},
        Subnets:   []string{"subnetid-1", "subnetid-2"},
	}
	_, err := s.elb.CreateLoadBalancer(createLB)
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Listeners.member.1.InstancePort"), Equals, "80")
	c.Assert(values.Get("Listeners.member.1.LoadBalancerPort"), Equals, "80")
	c.Assert(values.Get("Listeners.member.2.InstancePort"), Equals, "8080")
	c.Assert(values.Get("Listeners.member.2.LoadBalancerPort"), Equals, "8080")
	c.Assert(values.Get("Subnets.member.1"), Equals, "subnetid-1")
	c.Assert(values.Get("Subnets.member.2"), Equals, "subnetid-2")
}

func (s *S) TestCreateLoadBalancerWithWrongParamsCombination(c *C) {
	testServer.PrepareResponse(400, nil, CreateLoadBalancerBadRequest)
	createLB := &elb.CreateLoadBalancer{
		Name:       "testLB",
		AvailZones: []string{"us-east-1a", "us-east-1b"},
		Listeners: []elb.Listener{
			{
				InstancePort:     80,
				InstanceProtocol: "http",
				Protocol:         "http",
				LoadBalancerPort: 80,
			},
		},
		Subnets: []string{"subnetid-1", "subnetid2"},
	}
	resp, err := s.elb.CreateLoadBalancer(createLB)
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)
	e, ok := err.(*elb.Error)
	c.Assert(ok, Equals, true)
	c.Assert(e.Message, Equals, "Only one of SubnetIds or AvailabilityZones may be specified")
	c.Assert(e.Code, Equals, "ValidationError")
}

func (s *S) TestDeleteLoadBalancer(c *C) {
	testServer.PrepareResponse(200, nil, DeleteLoadBalancer)
	resp, err := s.elb.DeleteLoadBalancer("testlb")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "DeleteLoadBalancer")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testlb")
	c.Assert(resp.RequestId, Equals, "8d7223db-49d7-11e2-bba9-35ba56032fe1")
}

func (s *S) TestRegisterInstancesWithLoadBalancer(c *C) {
	testServer.PrepareResponse(200, nil, RegisterInstancesWithLoadBalancer)
	resp, err := s.elb.RegisterInstancesWithLoadBalancer([]string{"i-b44db8ca", "i-461ecf38"}, "testlb")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "RegisterInstancesWithLoadBalancer")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testlb")
	c.Assert(values.Get("Instances.member.1.InstanceId"), Equals, "i-b44db8ca")
	c.Assert(values.Get("Instances.member.2.InstanceId"), Equals, "i-461ecf38")
	c.Assert(resp.InstanceIds, DeepEquals, []string{"i-b44db8ca", "i-461ecf38"})
}

func (s *S) TestRegisterInstancesWithLoadBalancerBadRequest(c *C) {
	testServer.PrepareResponse(400, nil, RegisterInstancesWithLoadBalancerBadRequest)
	resp, err := s.elb.RegisterInstancesWithLoadBalancer([]string{"i-b44db8ca", "i-461ecf38"}, "absentLB")
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)
	e, ok := err.(*elb.Error)
	c.Assert(ok, Equals, true)
	c.Assert(e.Message, Equals, "There is no ACTIVE Load Balancer named 'absentLB'")
	c.Assert(e.Code, Equals, "LoadBalancerNotFound")
}

func (s *S) TestDeregisterInstancesFromLoadBalancer(c *C) {
	testServer.PrepareResponse(200, nil, DeregisterInstancesFromLoadBalancer)
	resp, err := s.elb.DeregisterInstancesFromLoadBalancer([]string{"i-b44db8ca", "i-461ecf38"}, "testlb")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "DeregisterInstancesFromLoadBalancer")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testlb")
	c.Assert(values.Get("Instances.member.1.InstanceId"), Equals, "i-b44db8ca")
	c.Assert(values.Get("Instances.member.2.InstanceId"), Equals, "i-461ecf38")
	c.Assert(resp.RequestId, Equals, "d6490837-49fd-11e2-bba9-35ba56032fe1")
}

func (s *S) TestDeregisterInstancesFromLoadBalancerBadRequest(c *C) {
	testServer.PrepareResponse(400, nil, DeregisterInstancesFromLoadBalancerBadRequest)
	resp, err := s.elb.DeregisterInstancesFromLoadBalancer([]string{"i-b44db8ca", "i-461ecf38"}, "testlb")
    c.Assert(resp, IsNil)
    c.Assert(err, NotNil)
	e, ok := err.(*elb.Error)
	c.Assert(ok, Equals, true)
	c.Assert(e.Message, Equals, "There is no ACTIVE Load Balancer named 'absentlb'")
	c.Assert(e.Code, Equals, "LoadBalancerNotFound")
}

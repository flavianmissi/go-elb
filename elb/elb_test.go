package elb_test

import (
	"github.com/flaviamissi/go-elb/aws"
	"github.com/flaviamissi/go-elb/elb"
	. "launchpad.net/gocheck"
	"time"
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
		Name:       "testlb",
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
	defer s.elb.DeleteLoadBalancer(createLB.Name)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Action"), Equals, "CreateLoadBalancer")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testlb")
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
		Name: "testlb",
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
		Subnets:        []string{"subnetid-1", "subnetid-2"},
		SecurityGroups: []string{"sg-1", "sg-2"},
	}
	_, err := s.elb.CreateLoadBalancer(createLB)
	c.Assert(err, IsNil)
	defer s.elb.DeleteLoadBalancer(createLB.Name)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Listeners.member.1.InstancePort"), Equals, "80")
	c.Assert(values.Get("Listeners.member.1.LoadBalancerPort"), Equals, "80")
	c.Assert(values.Get("Listeners.member.2.InstancePort"), Equals, "8080")
	c.Assert(values.Get("Listeners.member.2.LoadBalancerPort"), Equals, "8080")
	c.Assert(values.Get("Subnets.member.1"), Equals, "subnetid-1")
	c.Assert(values.Get("Subnets.member.2"), Equals, "subnetid-2")
	c.Assert(values.Get("SecurityGroups.member.1"), Equals, "sg-1")
	c.Assert(values.Get("SecurityGroups.member.2"), Equals, "sg-2")
}

func (s *S) TestCreateLoadBalancerWithWrongParamsCombination(c *C) {
	testServer.PrepareResponse(400, nil, CreateLoadBalancerBadRequest)
	createLB := &elb.CreateLoadBalancer{
		Name:       "testlb",
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

func (s *S) TestDescribeLoadBalancers(c *C) {
	testServer.PrepareResponse(200, nil, DescribeLoadBalancers)
	resp, err := s.elb.DescribeLoadBalancers()
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "DescribeLoadBalancers")
	t, _ := time.Parse(time.RFC3339, "2012-12-27T11:51:52.970Z")
	expected := &elb.DescribeLoadBalancerResp{
		[]elb.LoadBalancerDescription{
			{
				AvailZones:                []string{"us-east-1a"},
				BackendServerDescriptions: []elb.BackendServerDescriptions(nil),
				CanonicalHostedZoneName:   "testlb-2087227216.us-east-1.elb.amazonaws.com",
				CanonicalHostedZoneNameId: "Z3DZXE0Q79N41H",
				CreatedTime:               t,
				DNSName:                   "testlb-2087227216.us-east-1.elb.amazonaws.com",
				HealthCheck: elb.HealthCheck{
					HealthyThreshold:   10,
					Interval:           30,
					Target:             "TCP:80",
					Timeout:            5,
					UnhealthyThreshold: 2,
				},
				Instances: []elb.Instance(nil),
				ListenerDescriptions: []elb.ListenerDescription{
					{
						Listener: elb.Listener{
							Protocol:         "HTTP",
							LoadBalancerPort: 80,
							InstanceProtocol: "HTTP",
							InstancePort:     80,
						},
					},
				},
				LoadBalancerName: "testlb",
				//Policies:                  elb.Policies(nil),
				Scheme:         "internet-facing",
				SecurityGroups: []string(nil),
				SourceSecurityGroup: elb.SourceSecurityGroup{
					GroupName:  "amazon-elb-sg",
					OwnerAlias: "amazon-elb",
				},
				Subnets: []string(nil),
			},
		},
	}
	c.Assert(resp, DeepEquals, expected)
}

func (s *S) TestDescribeLoadBalancersByName(c *C) {
	testServer.PrepareResponse(200, nil, DescribeLoadBalancers)
	s.elb.DescribeLoadBalancers("somelb")
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "DescribeLoadBalancers")
	c.Assert(values.Get("LoadBalancerNames.member.1"), Equals, "somelb")
}

func (s *S) TestDescribeLoadBalancersBadRequest(c *C) {
	testServer.PrepareResponse(400, nil, DescribeLoadBalancersBadRequest)
	resp, err := s.elb.DescribeLoadBalancers()
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, `^Cannot find Load Balancer absentlb \(LoadBalancerNotFound\)$`)
}

func (s *S) TestDescribeInstanceHealth(c *C) {
	testServer.PrepareResponse(200, nil, DescribeInstanceHealth)
	resp, err := s.elb.DescribeInstanceHealth("testlb", "i-b44db8ca")
	c.Assert(err, IsNil)
	values := testServer.WaitRequest().URL.Query()
	c.Assert(values.Get("Version"), Equals, "2012-06-01")
	c.Assert(values.Get("Signature"), Not(Equals), "")
	c.Assert(values.Get("Timestamp"), Not(Equals), "")
	c.Assert(values.Get("Action"), Equals, "DescribeInstanceHealth")
	c.Assert(values.Get("LoadBalancerName"), Equals, "testlb")
	c.Assert(values.Get("Instances.member.1.InstanceId"), Equals, "i-b44db8ca")
	c.Assert(len(resp.InstanceStates) > 0, Equals, true)
	c.Assert(resp.InstanceStates[0].Description, Equals, "Instance registration is still in progress.")
	c.Assert(resp.InstanceStates[0].InstanceId, Equals, "i-b44db8ca")
	c.Assert(resp.InstanceStates[0].State, Equals, "OutOfService")
	c.Assert(resp.InstanceStates[0].ReasonCode, Equals, "ELB")
}

func (s *S) TestDescribeInstanceHealthBadRequest(c *C) {
	testServer.PrepareResponse(400, nil, DescribeInstanceHealthBadRequest)
	resp, err := s.elb.DescribeInstanceHealth("testlb", "i-foooo")
	c.Assert(err, NotNil)
	c.Assert(resp, IsNil)
	c.Assert(err, ErrorMatches, ".*i-foooo.*(InvalidInstance).*")
}

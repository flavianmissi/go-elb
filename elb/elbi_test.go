package elb_test

import (
	"flag"
	"github.com/flaviamissi/go-elb/aws"
	"github.com/flaviamissi/go-elb/ec2"
	"github.com/flaviamissi/go-elb/elb"
	. "launchpad.net/gocheck"
)

var amazon = flag.Bool("amazon", false, "Enable tests against amazon server")

// AmazonServer represents an Amazon AWS server.
type AmazonServer struct {
	auth aws.Auth
}

func (s *AmazonServer) SetUp(c *C) {
	auth, err := aws.EnvAuth()
	if err != nil {
		c.Fatal(err)
	}
	s.auth = auth
}

var _ = Suite(&AmazonClientSuite{})

// AmazonClientSuite tests the client against a live AWS server.
type AmazonClientSuite struct {
	srv AmazonServer
	ClientTests
}

// ClientTests defines integration tests designed to test the client.
// It is not used as a test suite in itself, but embedded within
// another type.
type ClientTests struct {
	elb *elb.ELB
	ec2 *ec2.EC2
}

func (s *AmazonClientSuite) SetUpSuite(c *C) {
	if !*amazon {
		c.Skip("AmazonClientSuite tests not enabled")
	}
	s.srv.SetUp(c)
	s.elb = elb.New(s.srv.auth, aws.USEast)
	s.ec2 = ec2.New(s.srv.auth, aws.USEast)
}

func (s *ClientTests) TestCreateAndDeleteLoadBalancer(c *C) {
    createLBReq := elb.CreateLoadBalancer{
        Name:       "testLB",
        AvailZones: []string{"us-east-1a"},
        Listeners:  []elb.Listener{
            {
                InstancePort:     80,
                InstanceProtocol: "http",
                LoadBalancerPort: 80,
                Protocol:         "http",
            },
        },
    }
    resp, err := s.elb.CreateLoadBalancer(&createLBReq)
    c.Assert(err, IsNil)
    c.Assert(resp.DNSName, Not(Equals), "")
    deleteResp, err := s.elb.DeleteLoadBalancer(createLBReq.Name)
    c.Assert(err, IsNil)
    c.Assert(deleteResp.RequestId, Not(Equals), "")
}

func (s *ClientTests) TestCreateLoadBalancerError(c *C) {
    createLBReq := elb.CreateLoadBalancer{
        Name:       "testLB",
        AvailZones: []string{"us-east-1a"},
        Subnets:    []string{"subnetid-1"},
        Listeners:  []elb.Listener{
            {
                InstancePort:     80,
                InstanceProtocol: "http",
                LoadBalancerPort: 80,
                Protocol:         "http",
            },
        },
    }
    resp, err := s.elb.CreateLoadBalancer(&createLBReq)
    c.Assert(resp, IsNil)
    c.Assert(err, NotNil)
	e, ok := err.(*elb.Error)
	c.Assert(ok, Equals, true)
	c.Assert(e.Message, Matches, "Only one of .* or .* may be specified")
	c.Assert(e.Code, Equals, "ValidationError")
}

// Cost: 0.02 USD
func (s *ClientTests) TestCreateAndRegisterAndDeregisterInstanceWithLoadBalancer(c *C) {
	options := ec2.RunInstances{
		ImageId:      "ami-ccf405a5",
		InstanceType: "t1.micro",
	}
	resp1, err := s.ec2.RunInstances(&options)
	c.Assert(err, IsNil)
	instId := resp1.Instances[0].InstanceId
    createLBReq := elb.CreateLoadBalancer{
        Name:       "testLB",
        AvailZones: []string{"us-east-1a"},
        Listeners:  []elb.Listener{
            {
                InstancePort:     80,
                InstanceProtocol: "http",
                LoadBalancerPort: 80,
                Protocol:         "http",
            },
        },
    }
    _, err = s.elb.CreateLoadBalancer(&createLBReq)
    c.Assert(err, IsNil)
    defer func () {
        _, err := s.elb.DeleteLoadBalancer(createLBReq.Name)
        c.Check(err, IsNil)
        _, err = s.ec2.TerminateInstances([]string{instId})
        c.Check(err, IsNil)
    }()

    resp, err := s.elb.RegisterInstancesWithLoadBalancer([]string{instId}, createLBReq.Name)
    c.Assert(err, IsNil)
    c.Assert(resp.InstanceIds, DeepEquals, []string{instId})
    resp2, err := s.elb.DeregisterInstancesFromLoadBalancer([]string{instId}, createLBReq.Name)
    c.Assert(err, IsNil)
    c.Assert(resp2, Not(Equals), "")
}

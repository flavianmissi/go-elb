package elb_test

import (
	"launchpad.net/goamz/aws"
	"github.com/flaviamissi/go-elb"
	"github.com/flaviamissi/go-elb/elbtest"
	. "launchpad.net/gocheck"
)

// LocalServer represents a local elbtest fake server.
type LocalServer struct {
	auth   aws.Auth
	region aws.Region
	srv    *elbtest.Server
}

func (s *LocalServer) SetUp(c *C) {
	srv, err := elbtest.NewServer()
	c.Assert(err, IsNil)
	c.Assert(srv, NotNil)
	s.srv = srv
	s.region = aws.Region{ELBEndpoint: srv.URL()}
}

// LocalServerSuite defines tests that will run
// against the local elbtest server. It includes
// selected tests from ClientTests;
// when the elbtest functionality is sufficient, it should
// include all of them, and ClientTests can be simply embedded.
type LocalServerSuite struct {
	srv LocalServer
	ServerTests
	clientTests ClientTests
}

// ServerTests defines a set of tests designed to test
// the elbtest local fake elb server.
// It is not used as a test suite in itself, but embedded within
// another type.
type ServerTests struct {
	elb *elb.ELB
}

// AmazonServerSuite runs the elbtest server tests against a live ELB server.
// It will only be activated if the -all flag is specified.
type AmazonServerSuite struct {
	srv AmazonServer
	ServerTests
}

var _ = Suite(&AmazonServerSuite{})

func (s *AmazonServerSuite) SetUpSuite(c *C) {
	if !*amazon {
		c.Skip("AmazonServerSuite tests not enabled")
	}
	s.srv.SetUp(c)
	s.ServerTests.elb = elb.New(s.srv.auth, aws.USEast)
}

var _ = Suite(&LocalServerSuite{})

func (s *LocalServerSuite) SetUpSuite(c *C) {
	s.srv.SetUp(c)
	s.ServerTests.elb = elb.New(s.srv.auth, s.srv.region)
	s.clientTests.elb = elb.New(s.srv.auth, s.srv.region)
}


func (s *LocalServerSuite) TestCreateLoadBalancer(c *C) {
    s.clientTests.TestCreateAndDeleteLoadBalancer(c)
}

func (s *LocalServerSuite) TestCreateLoadBalancerError(c *C) {
    s.clientTests.TestCreateLoadBalancerError(c)
}

func (s *LocalServerSuite) TestRegisterInstanceWithLoadBalancer(c *C) {
    srv := s.srv.srv
    instId := srv.NewInstance()
    defer srv.RemoveInstance(instId)
    srv.NewLoadBalancer("testlb")
    defer srv.RemoveLoadBalancer("testlb")
    resp, err := s.clientTests.elb.RegisterInstancesWithLoadBalancer([]string{instId}, "testlb")
    c.Assert(err, IsNil)
    c.Assert(resp.InstanceIds, DeepEquals, []string{instId})

}

func (s *LocalServerSuite) TestRegisterInstanceWithLoadBalancerWithAbsentInstance(c *C) {
    srv := s.srv.srv
    srv.NewLoadBalancer("testlb")
    defer srv.RemoveLoadBalancer("testlb")
    resp, err := s.clientTests.elb.RegisterInstancesWithLoadBalancer([]string{"i-212"}, "testlb")
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, `^InvalidInstance found in \[i-212\]. Invalid id: "i-212" \(InvalidInstance\)$`)
    c.Assert(resp, IsNil)
}

func (s *LocalServerSuite) TestRegisterInstanceWithLoadBalancerWithAbsentLoadBalancer(c *C) {
    // the verification if the lb exists is done before the instances, so there is no need to create
    // fixture instances for this test, it'll never get that far
    resp, err := s.clientTests.elb.RegisterInstancesWithLoadBalancer([]string{"i-212"}, "absentlb")
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, `^There is no ACTIVE Load Balancer named 'absentlb' \(LoadBalancerNotFound\)$`)
    c.Assert(resp, IsNil)
}

func (s *LocalServerSuite) TestDeregisterInstanceWithLoadBalancer(c *C) {
    // there is no need to register the instance first, amazon returns the same response
    // in both cases (instance registered or not)
    srv := s.srv.srv
    instId := srv.NewInstance()
    defer srv.RemoveInstance(instId)
    srv.NewLoadBalancer("testlb")
    defer srv.RemoveLoadBalancer("testlb")
    resp, err := s.clientTests.elb.DeregisterInstancesFromLoadBalancer([]string{instId}, "testlb")
    c.Assert(err, IsNil)
    c.Assert(resp.RequestId, Not(Equals), "")
}

func (s *LocalServerSuite) TestDeregisterInstanceWithLoadBalancerWithAbsentLoadBalancer(c *C) {
    resp, err := s.clientTests.elb.DeregisterInstancesFromLoadBalancer([]string{"i-212"}, "absentlb")
    c.Assert(resp, IsNil)
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, `^There is no ACTIVE Load Balancer named 'absentlb' \(LoadBalancerNotFound\)$`)
}

func (s *LocalServerSuite) TestDeregisterInstancewithLoadBalancerWithAbsentInstance(c *C) {
    srv := s.srv.srv
    srv.NewLoadBalancer("testlb")
    defer srv.RemoveLoadBalancer("testlb")
    resp, err := s.clientTests.elb.DeregisterInstancesFromLoadBalancer([]string{"i-212"}, "testlb")
    c.Assert(resp, IsNil)
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, `^InvalidInstance found in \[i-212\]. Invalid id: "i-212" \(InvalidInstance\)$`)
}

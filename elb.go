// This package provides types and functions to interact Elastic Load Balancing service
package elb

import (
	"encoding/xml"
	"fmt"
	"launchpad.net/goamz/aws"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ELB struct {
	aws.Auth
	aws.Region
}

func New(auth aws.Auth, region aws.Region) *ELB {
	return &ELB{auth, region}
}

// The CreateLoadBalancer type encapsulates options for the respective request in AWS.
// The creation of a Load Balancer may differ inside EC2 and VPC.
//
// See http://goo.gl/4QFKi for more details.
type CreateLoadBalancer struct {
	Name           string
	AvailZones     []string
	Listeners      []Listener
	Scheme         string
	SecurityGroups []string
	Subnets        []string
}

// Listener to configure in Load Balancer.
//
// See http://goo.gl/NJQCj for more details.
type Listener struct {
	InstancePort     int
	InstanceProtocol string
	LoadBalancerPort int
	Protocol         string
	SSLCertificateId string
}

// Response to a CreateLoadBalance request.
//
// See http://goo.gl/4QFKi for more details.
type CreateLoadBalancerResp struct {
	DNSName string `xml:"CreateLoadBalancerResult>DNSName"`
}

type SimpleResp struct {
	RequestId string `xml:"ResponseMetadata>RequestId"`
}

// Creates a Load Balancer in Amazon.
//
// See http://goo.gl/4QFKi for more details.
func (elb *ELB) CreateLoadBalancer(options *CreateLoadBalancer) (resp *CreateLoadBalancerResp, err error) {
	params := makeCreateParams(options)
	resp = new(CreateLoadBalancerResp)
	if err := elb.query(params, resp); err != nil {
		return nil, err
	}
	return
}

// Deletes a Load Balancer.
//
// See http://goo.gl/sDmPp for more details.
func (elb *ELB) DeleteLoadBalancer(name string) (resp *SimpleResp, err error) {
	params := map[string]string{
		"Action":           "DeleteLoadBalancer",
		"LoadBalancerName": name,
	}
	resp = new(SimpleResp)
	if err := elb.query(params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RegisterInstancesResp struct {
	InstanceIds []string `xml:"RegisterInstancesWithLoadBalancerResult>Instances>member>InstanceId"`
}

// Register N instances with a given Load Balancer.
//
// See http://goo.gl/x9hru for more details.
func (elb *ELB) RegisterInstancesWithLoadBalancer(instanceIds []string, lbName string) (resp *RegisterInstancesResp, err error) {
	params := map[string]string{
		"Action":           "RegisterInstancesWithLoadBalancer",
		"LoadBalancerName": lbName,
	}
	for i, instanceId := range instanceIds {
		key := fmt.Sprintf("Instances.member.%d.InstanceId", i+1)
		params[key] = instanceId
	}
	resp = new(RegisterInstancesResp)
	if err := elb.query(params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Deregister N instances from a given Load Balancer.
//
// See http://goo.gl/Hgo4U for more details.
func (elb *ELB) DeregisterInstancesFromLoadBalancer(instanceIds []string, lbName string) (resp *SimpleResp, err error) {
	params := map[string]string{
		"Action":           "DeregisterInstancesFromLoadBalancer",
		"LoadBalancerName": lbName,
	}
	for i, instanceId := range instanceIds {
		key := fmt.Sprintf("Instances.member.%d.InstanceId", i+1)
		params[key] = instanceId
	}
	resp = new(SimpleResp)
	if err := elb.query(params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (elb *ELB) query(params map[string]string, resp interface{}) error {
	params["Version"] = "2012-06-01"
	params["Timestamp"] = time.Now().In(time.UTC).Format(time.RFC3339)
	endpoint, err := url.Parse(elb.Region.ELBEndpoint)
	if err != nil {
		return err
	}
	if endpoint.Path == "" {
		endpoint.Path = "/"
	}
	sign(elb.Auth, "GET", endpoint.Path, params, endpoint.Host)
	endpoint.RawQuery = multimap(params).Encode()
	r, err := http.Get(endpoint.String())
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return buildError(r)
	}
	return xml.NewDecoder(r.Body).Decode(resp)
}

// Error encapsulates an error returned by ELB.
type Error struct {
	// HTTP status code
	StatusCode int
	// AWS error code
	Code string
	// The human-oriented error message
	Message string
}

func (err *Error) Error() string {
	if err.Code == "" {
		return err.Message
	}

	return fmt.Sprintf("%s (%s)", err.Message, err.Code)
}

type xmlErrors struct {
	Errors []Error `xml:"Error"`
}

func buildError(r *http.Response) error {
	var (
		err    Error
		errors xmlErrors
	)
	xml.NewDecoder(r.Body).Decode(&errors)
	if len(errors.Errors) > 0 {
		err = errors.Errors[0]
	}
	err.StatusCode = r.StatusCode
	if err.Message == "" {
		err.Message = r.Status
	}
	return &err
}

func multimap(p map[string]string) url.Values {
	q := make(url.Values, len(p))
	for k, v := range p {
		q[k] = []string{v}
	}
	return q
}

func makeCreateParams(createLB *CreateLoadBalancer) map[string]string {
	params := make(map[string]string)
	params["LoadBalancerName"] = createLB.Name
	params["Action"] = "CreateLoadBalancer"
	if createLB.Scheme != "" {
		params["Scheme"] = createLB.Scheme
	}
    for i, s := range createLB.Subnets {
		key := fmt.Sprintf("Subnets.member.%d", i + 1)
        params[key] = s
	}
	for i, l := range createLB.Listeners {
		key := "Listeners.member.%d.%s"
		index := i + 1
		params[fmt.Sprintf(key, index, "InstancePort")] = strconv.Itoa(l.InstancePort)
		params[fmt.Sprintf(key, index, "InstanceProtocol")] = l.InstanceProtocol
		params[fmt.Sprintf(key, index, "Protocol")] = l.Protocol
		params[fmt.Sprintf(key, index, "LoadBalancerPort")] = strconv.Itoa(l.LoadBalancerPort)
	}
	for i, az := range createLB.AvailZones {
		key := fmt.Sprintf("AvailabilityZones.member.%d", i + 1)
		params[key] = az
	}
	return params
}

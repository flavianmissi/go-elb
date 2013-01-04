// Package elbtest implements a fake ELB provider with the capability of
// inducing errors on any given operation, and retrospectively determining what
// operations have been carried out.
package elbtest

import (
	"encoding/xml"
	"fmt"
	"github.com/flaviamissi/go-elb/elb"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type lb struct {
	dnsName   string
	instances []string
}

func (l *lb) addInstance(id string) {
	l.instances = append(l.instances, id)
}

func (l *lb) removeInstance(id string) {
	index := -1
	for i, instance := range l.instances {
		if instance == id {
			index = i
			break
		}
	}
	if index > -1 {
		copy(l.instances[index:], l.instances[index+1:])
		l.instances = l.instances[:len(l.instances)-1]
	}
}

// Server implements an ELB simulator for use in testing.
type Server struct {
	url       string
	listener  net.Listener
	mutex     sync.Mutex
	reqId     int
	lbs       map[string]lb
	lbsReqs   map[string]url.Values
	instances []string
	instCount int
}

// Starts and returns a new server
func NewServer() (*Server, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("cannot listen on localhost: %v", err)
	}
	srv := &Server{
		listener: l,
		url:      "http://" + l.Addr().String(),
		lbsReqs:  map[string]url.Values{},
		lbs:      make(map[string]lb),
	}
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		srv.serveHTTP(w, req)
	}))
	return srv, nil
}

// Quit closes down the server.
func (srv *Server) Quit() {
	srv.listener.Close()
}

// URL returns the URL of the server.
func (srv *Server) URL() string {
	return srv.url
}

type xmlErrors struct {
	XMLName string `xml:"ErrorResponse"`
	Error   elb.Error
}

func (srv *Server) error(w http.ResponseWriter, err *elb.Error) {
	w.WriteHeader(err.StatusCode)
	xmlErr := xmlErrors{Error: *err}
	if e := xml.NewEncoder(w).Encode(xmlErr); e != nil {
		panic(e)
	}
}

func (srv *Server) serveHTTP(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	f := actions[req.Form.Get("Action")]
	if f == nil {
		srv.error(w, &elb.Error{
			StatusCode: 400,
			Code:       "InvalidParameterValue",
			Message:    "Unrecognized Action",
		})
	}
	reqId := fmt.Sprintf("req%0X", srv.reqId)
	srv.reqId++
	if resp, err := f(srv, w, req, reqId); err == nil {
		if err := xml.NewEncoder(w).Encode(resp); err != nil {
			panic(err)
		}
	} else {
		switch err.(type) {
		case *elb.Error:
			srv.error(w, err.(*elb.Error))
		default:
			panic(err)
		}
	}
}

func (srv *Server) createLoadBalancer(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	composition := map[string]string{
		"AvailabilityZones.member.1": "Subnets.member.1",
	}
	if err := srv.validateComposition(req, composition); err != nil {
		return nil, err
	}
	required := []string{
		"Listeners.member.1.InstancePort",
		"Listeners.member.1.InstanceProtocol",
		"Listeners.member.1.Protocol",
		"Listeners.member.1.LoadBalancerPort",
		"LoadBalancerName",
	}
	if err := srv.validate(req, required); err != nil {
		return nil, err
	}
	path := req.FormValue("Path")
	if path == "" {
		path = "/"
	}
	lbName := req.FormValue("LoadBalancerName")
	srv.lbsReqs[lbName] = req.Form
	srv.lbs[lbName] = lb{
		dnsName: fmt.Sprintf("%s-some-aws-stuff.us-east-1.elb.amazonaws.com", lbName),
	}
	return elb.CreateLoadBalancerResp{
		DNSName: srv.lbs[lbName].dnsName,
	}, nil
}

func (srv *Server) deleteLoadBalancer(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	if err := srv.validate(req, []string{"LoadBalancerName"}); err != nil {
		return nil, err
	}
	srv.RemoveLoadBalancer(req.FormValue("LoadBalancerName"))
	return elb.SimpleResp{RequestId: reqId}, nil
}

func (srv *Server) registerInstancesWithLoadBalancer(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	required := []string{"LoadBalancerName", "Instances.member.1.InstanceId"}
	if err := srv.validate(req, required); err != nil {
		return nil, err
	}
	if err := srv.lbExists(req.FormValue("LoadBalancerName")); err != nil {
		return nil, err
	}
	instIds := []string{}
	i := 1
	instId := req.FormValue(fmt.Sprintf("Instances.member.%d.InstanceId", i))
	lb := srv.lbs[req.FormValue("LoadBalancerName")]
	for instId != "" {
		if err := srv.instanceExists(instId); err != nil {
			return nil, err
		}
		instIds = append(instIds, instId)
		lb.addInstance(instId)
		i++
		instId = req.FormValue(fmt.Sprintf("Instances.member.%d.InstanceId", i))
	}
	srv.lbs[req.FormValue("LoadBalancerName")] = lb
	return elb.RegisterInstancesResp{InstanceIds: instIds}, nil
}

func (srv *Server) deregisterInstancesFromLoadBalancer(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	required := []string{"LoadBalancerName"}
	if err := srv.validate(req, required); err != nil {
		return nil, err
	}
	if err := srv.lbExists(req.FormValue("LoadBalancerName")); err != nil {
		return nil, err
	}
	i := 1
	lb := srv.lbs[req.FormValue("LoadBalancerName")]
	instId := req.FormValue(fmt.Sprintf("Instances.member.%d.InstanceId", i))
	for instId != "" {
		if err := srv.instanceExists(instId); err != nil {
			return nil, err
		}
		i++
		lb.removeInstance(instId)
		instId = req.FormValue(fmt.Sprintf("Instances.member.%d.InstanceId", i))
	}
	srv.lbs[req.FormValue("LoadBalancerName")] = lb
	return elb.SimpleResp{RequestId: reqId}, nil
}

// getParameters returns the value all parameters from a request that matches a
// prefix.
//
// For example, for the prefix "Subnets.member.", it will return a slice
// containing the value of keys "Subnets.member.1", "Subnets.member.2" ...
// "Subnets.member.N". The prefix must include the trailing dot.
func (srv *Server) getParameters(prefix string, values url.Values) []string {
	i, key := 1, ""
	var k = func(n int) string {
		return fmt.Sprintf(prefix+"%d", n)
	}
	var result []string
	for i, key = 2, k(i); values.Get(key) != ""; i, key = i+1, k(i) {
		result = append(result, values.Get(key))
	}
	return result
}

func (srv *Server) describeLoadBalancers(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	i := 1
	lbName := req.FormValue(fmt.Sprintf("LoadBalancerNames.member.%d", i))
	for lbName != "" {
		key := fmt.Sprintf("LoadBalancerNames.member.%d", i)
		if req.FormValue(key) != "" {
			if err := srv.lbExists(req.FormValue(key)); err != nil {
				return nil, err
			}
		}
		i++
		lbName = req.FormValue(fmt.Sprintf("LoadBalancerNames.member.%d", i))
	}
	var resp elb.DescribeLoadBalancerResp
	for name, value := range srv.lbsReqs {
		ht := 10
		timeout := 5
		ut := 2
		interval := 30
		target := "TCP:80"
		if v := value.Get("HealthCheck.HealthyThreshold"); v != "" {
			ht, _ = strconv.Atoi(v)
		}
		if v := value.Get("HealthCheck.Timeout"); v != "" {
			timeout, _ = strconv.Atoi(v)
		}
		if v := value.Get("HealthCheck.UnhealthyThreshold"); v != "" {
			ut, _ = strconv.Atoi(v)
		}
		if v := value.Get("HealthCheck.Interval"); v != "" {
			interval, _ = strconv.Atoi(v)
		}
		if v := value.Get("HealthCheck.Target"); v != "" {
			target = v
		}
		hc := elb.HealthCheck{
			HealthyThreshold:   ht,
			Interval:           interval,
			Target:             target,
			Timeout:            timeout,
			UnhealthyThreshold: ut,
		}
		lds := []elb.ListenerDescription{}
		i := 1
		protocol := value.Get(fmt.Sprintf("Listeners.member.%d.Protocol", i))
		for protocol != "" {
			key := fmt.Sprintf("Listeners.member.%d.", i)
			lInstPort, _ := strconv.Atoi(value.Get(key + "InstancePort"))
			lLBPort, _ := strconv.Atoi(value.Get(key + "LoadBalancerPort"))
			lDescription := elb.ListenerDescription{
				Listener: elb.Listener{
					Protocol:         strings.ToUpper(protocol),
					InstanceProtocol: strings.ToUpper(value.Get(key + "InstanceProtocol")),
					LoadBalancerPort: lLBPort,
					InstancePort:     lInstPort,
				},
			}
			i++
			protocol = value.Get(fmt.Sprintf("Listeners.member.%d.Protocol", i))
			lds = append(lds, lDescription)
		}
		ssgGName := "amazon-elb-sg"
		ssgOAlias := "amazon-elb"
		if v := value.Get("SourceSecurityGroup.GroupName"); v != "" {
			ssgGName = v
		}
		if v := value.Get("SourceSecurityGroup.OwnerAlias"); v != "" {
			ssgOAlias = v
		}
		lbDesc := elb.LoadBalancerDescription{
			AvailZones:           srv.getParameters("AvailabilityZones.member.", value),
			Subnets:              srv.getParameters("Subnets.member.", value),
			SecurityGroups:       srv.getParameters("SecurityGroups.member.", value),
			LoadBalancerName:     name,
			HealthCheck:          hc,
			ListenerDescriptions: lds,
			Scheme:               value.Get("Scheme"),
			SourceSecurityGroup: elb.SourceSecurityGroup{
				GroupName:  ssgGName,
				OwnerAlias: ssgOAlias,
			},
		}
		if lbDesc.Scheme == "" {
			lbDesc.Scheme = "internet-facing"
		}
		lbDesc.LoadBalancerName = value.Get("LoadBalancerName")
		lbDesc.DNSName = srv.lbs[lbDesc.LoadBalancerName].dnsName
		if l := len(srv.lbs[lbDesc.LoadBalancerName].instances); l > 0 {
			lbDesc.Instances = make([]elb.Instance, l)
			lb := srv.lbs[lbDesc.LoadBalancerName]
			for i := 0; i < l; i++ {
				lbDesc.Instances[i] = elb.Instance{
					InstanceId: lb.instances[i],
				}
			}
		}
		resp.LoadBalancerDescriptions = append(resp.LoadBalancerDescriptions, lbDesc)
	}
	return resp, nil
}

func (srv *Server) describeInstanceHealth(w http.ResponseWriter, req *http.Request, reqId string) (interface{}, error) {
	if err := srv.lbExists(req.FormValue("LoadBalancerName")); err != nil {
		return nil, err
	}
	resp := elb.DescribeInstanceHealthResp{
		InstanceStates: []elb.InstanceState{},
	}
	i := 1
	instanceId := req.FormValue("Instances.member.1.InstanceId")
	for instanceId != "" {
		if err := srv.instanceExists(instanceId); err != nil {
			return nil, err
		}
		is := elb.InstanceState{
			Description: "Instance is in pending state.",
			InstanceId:  instanceId,
			State:       "OutOfService",
			ReasonCode:  "Instance",
		}
		resp.InstanceStates = append(resp.InstanceStates, is)
		i++
		instanceId = req.FormValue(fmt.Sprintf("Instances.member.%d.InstanceId", i))
	}
	return resp, nil
}

func (srv *Server) instanceExists(id string) error {
	for _, instId := range srv.instances {
		if instId == id {
			return nil
		}
	}
	return &elb.Error{
		StatusCode: 400,
		Code:       "InvalidInstance",
		Message:    fmt.Sprintf("InvalidInstance found in [%s]. Invalid id: \"%s\"", id, id),
	}
}

func (srv *Server) lbExists(name string) error {
	if _, ok := srv.lbs[name]; !ok {
		return &elb.Error{
			StatusCode: 400,
			Code:       "LoadBalancerNotFound",
			Message:    fmt.Sprintf("There is no ACTIVE Load Balancer named '%s'", name),
		}
	}
	return nil
}

func (srv *Server) validate(req *http.Request, required []string) error {
	for _, field := range required {
		if req.FormValue(field) == "" {
			return &elb.Error{
				StatusCode: 400,
				Code:       "ValidationError",
				Message:    fmt.Sprintf("%s is required.", field),
			}
		}
	}
	return nil
}

// Validates the composition of the fields.
//
// Some fields cannot be together in the same request, such as AvailabilityZones and Subnets.
// A sample map with the above requirement would be
//    c := map[string]string{
//        "AvailabilityZones.member.1": "Subnets.member.1",
//    }
//
// The server also requires that at least one of those fields are specified.
func (srv *Server) validateComposition(req *http.Request, composition map[string]string) error {
	for k, v := range composition {
		if req.FormValue(k) != "" && req.FormValue(v) != "" {
			return &elb.Error{
				StatusCode: 400,
				Code:       "ValidationError",
				Message:    fmt.Sprintf("Only one of %s or %s may be specified", k, v),
			}
		}
		if req.FormValue(k) == "" && req.FormValue(v) == "" {
			return &elb.Error{
				StatusCode: 400,
				Code:       "ValidationError",
				Message:    fmt.Sprintf("Either %s or %s must be specified", k, v),
			}
		}
	}
	return nil
}

// Creates a fake instance in the server
func (srv *Server) NewInstance() string {
	srv.instCount++
	instId := fmt.Sprintf("i-%d", srv.instCount)
	srv.instances = append(srv.instances, instId)
	return instId
}

// Removes a fake instance from the server
//
// If no instance is found it does nothing
func (srv *Server) RemoveInstance(instId string) {
	for i, id := range srv.instances {
		if id == instId {
			srv.instances[i], srv.instances = srv.instances[len(srv.instances)-1], srv.instances[:len(srv.instances)-1]
		}
	}
}

// Creates a fake load balancer in the fake server
func (srv *Server) NewLoadBalancer(name string) {
	srv.lbs[name] = lb{
		dnsName: fmt.Sprintf("%s-some-aws-stuff.sa-east-1.amazonaws.com", name),
	}
}

// Removes a fake load balancer from the fake server
func (srv *Server) RemoveLoadBalancer(name string) {
	delete(srv.lbs, name)
	delete(srv.lbsReqs, name)
}

var actions = map[string]func(*Server, http.ResponseWriter, *http.Request, string) (interface{}, error){
	"CreateLoadBalancer":                  (*Server).createLoadBalancer,
	"DeleteLoadBalancer":                  (*Server).deleteLoadBalancer,
	"RegisterInstancesWithLoadBalancer":   (*Server).registerInstancesWithLoadBalancer,
	"DeregisterInstancesFromLoadBalancer": (*Server).deregisterInstancesFromLoadBalancer,
	"DescribeLoadBalancers":               (*Server).describeLoadBalancers,
	"DescribeInstanceHealth":              (*Server).describeInstanceHealth,
}

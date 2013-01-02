package elb_test

var CreateLoadBalancer = `
<CreateLoadBalancerResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <CreateLoadBalancerResult>
        <DNSName>testlb-339187009.us-east-1.elb.amazonaws.com</DNSName>
    </CreateLoadBalancerResult>
    <ResponseMetadata>
        <RequestId>0c3a8e29-490e-11e2-8647-e14ad5151f1f</RequestId>
    </ResponseMetadata>
</CreateLoadBalancerResponse>
`

var CreateLoadBalancerBadRequest = `
<ErrorResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <Error>
        <Type>Sender</Type>
        <Code>ValidationError</Code>
        <Message>Only one of SubnetIds or AvailabilityZones may be specified</Message>
    </Error>
    <RequestId>159253fc-49dc-11e2-a47d-cde463c91a3c</RequestId>
</ErrorResponse>
`

var DeleteLoadBalancer = `
<DeleteLoadBalancerResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <DeleteLoadBalancerResult/>
    <ResponseMetadata>
        <RequestId>8d7223db-49d7-11e2-bba9-35ba56032fe1</RequestId>
    </ResponseMetadata>
</DeleteLoadBalancerResponse>
`

var RegisterInstancesWithLoadBalancer = `
<RegisterInstancesWithLoadBalancerResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <RegisterInstancesWithLoadBalancerResult>
        <Instances>
            <member>
                <InstanceId>i-b44db8ca</InstanceId>
            </member>
            <member>
                <InstanceId>i-461ecf38</InstanceId>
            </member>
        </Instances>
    </RegisterInstancesWithLoadBalancerResult>
    <ResponseMetadata>
        <RequestId>0fc82478-49e1-11e2-b947-8768f15220aa</RequestId>
    </ResponseMetadata>
</RegisterInstancesWithLoadBalancerResponse>
`

var RegisterInstancesWithLoadBalancerBadRequest = `
<ErrorResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <Error>
        <Type>Sender</Type>
        <Code>LoadBalancerNotFound</Code>
        <Message>There is no ACTIVE Load Balancer named 'absentLB'</Message>
    </Error>
    <RequestId>19a0bb97-49f7-11e2-90b4-6bb9ec8331bf</RequestId>
</ErrorResponse>
`

var DeregisterInstancesFromLoadBalancer = `
<DeregisterInstancesFromLoadBalancerResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <DeregisterInstancesFromLoadBalancerResult>
        <Instances/>
    </DeregisterInstancesFromLoadBalancerResult>
    <ResponseMetadata>
        <RequestId>d6490837-49fd-11e2-bba9-35ba56032fe1</RequestId>
    </ResponseMetadata>
</DeregisterInstancesFromLoadBalancerResponse>
`

var DeregisterInstancesFromLoadBalancerBadRequest = `
<ErrorResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
    <Error>
        <Type>Sender</Type>
        <Code>LoadBalancerNotFound</Code>
        <Message>There is no ACTIVE Load Balancer named 'absentlb'</Message>
    </Error>
    <RequestId>498e2b4a-4aa1-11e2-8839-d19a879f2eec</RequestId>
</ErrorResponse>
`

package elb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"tasnim.dev/aws-tui/internal/utils"
)

type ELBAPI interface {
	DescribeLoadBalancers(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	DescribeListeners(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
	DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	DescribeTargetHealth(ctx context.Context, params *elbv2.DescribeTargetHealthInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
	DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
	DescribeLoadBalancerAttributes(ctx context.Context, params *elbv2.DescribeLoadBalancerAttributesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancerAttributesOutput, error)
	DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error)
}

type Client struct {
	api ELBAPI
}

func NewClient(api ELBAPI) *Client {
	return &Client{api: api}
}

func (c *Client) ListLoadBalancers(ctx context.Context) ([]ELBLoadBalancer, error) {
	var lbs []ELBLoadBalancer
	var marker *string

	for {
		out, err := c.api.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeLoadBalancers: %w", err)
		}

		for _, lb := range out.LoadBalancers {
			var state string
			if lb.State != nil {
				state = string(lb.State.Code)
			}
			var createdAt time.Time
			if lb.CreatedTime != nil {
				createdAt = *lb.CreatedTime
			}
			lbs = append(lbs, ELBLoadBalancer{
				Name:      aws.ToString(lb.LoadBalancerName),
				ARN:       aws.ToString(lb.LoadBalancerArn),
				Type:      string(lb.Type),
				State:     state,
				Scheme:    string(lb.Scheme),
				DNSName:   aws.ToString(lb.DNSName),
				VPCID:     aws.ToString(lb.VpcId),
				CreatedAt: createdAt,
			})
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return lbs, nil
}

func (c *Client) ListListeners(ctx context.Context, lbARN string) ([]ELBListener, error) {
	var listeners []ELBListener
	var marker *string

	for {
		out, err := c.api.DescribeListeners(ctx, &elbv2.DescribeListenersInput{
			LoadBalancerArn: aws.String(lbARN),
			Marker:          marker,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeListeners: %w", err)
		}

		for _, l := range out.Listeners {
			var certs []string
			for _, c := range l.Certificates {
				if arn := aws.ToString(c.CertificateArn); arn != "" {
					certs = append(certs, arn)
				}
			}
			listeners = append(listeners, ELBListener{
				ARN:           aws.ToString(l.ListenerArn),
				Port:          int(aws.ToInt32(l.Port)),
				Protocol:      string(l.Protocol),
				DefaultAction: formatAction(l.DefaultActions),
				SSLPolicy:     aws.ToString(l.SslPolicy),
				Certificates:  certs,
			})
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return listeners, nil
}

func (c *Client) ListTargetGroups(ctx context.Context, lbARN string) ([]ELBTargetGroup, error) {
	var tgs []ELBTargetGroup
	var marker *string

	for {
		out, err := c.api.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
			LoadBalancerArn: aws.String(lbARN),
			Marker:          marker,
		})
		if err != nil {
			return nil, fmt.Errorf("DescribeTargetGroups: %w", err)
		}

		for _, tg := range out.TargetGroups {
			item, err := c.buildTargetGroup(ctx, tg)
			if err != nil {
				return nil, err
			}
			tgs = append(tgs, item)
		}

		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return tgs, nil
}

// ListListenerTargetGroups returns target groups referenced by rules of a specific listener.
func (c *Client) ListListenerTargetGroups(ctx context.Context, listenerARN string) ([]ELBTargetGroup, error) {
	rulesOut, err := c.api.DescribeRules(ctx, &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(listenerARN),
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeRules: %w", err)
	}

	seen := map[string]bool{}
	var tgARNs []string
	for _, rule := range rulesOut.Rules {
		for _, action := range rule.Actions {
			arn := aws.ToString(action.TargetGroupArn)
			if arn != "" && !seen[arn] {
				seen[arn] = true
				tgARNs = append(tgARNs, arn)
			}
		}
	}

	if len(tgARNs) == 0 {
		return nil, nil
	}

	out, err := c.api.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: tgARNs,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTargetGroups: %w", err)
	}

	tgs := make([]ELBTargetGroup, 0, len(out.TargetGroups))
	for _, tg := range out.TargetGroups {
		item, err := c.buildTargetGroup(ctx, tg)
		if err != nil {
			return nil, err
		}
		tgs = append(tgs, item)
	}
	return tgs, nil
}

func (c *Client) buildTargetGroup(ctx context.Context, tg elbtypes.TargetGroup) (ELBTargetGroup, error) {
	item := ELBTargetGroup{
		Name:       aws.ToString(tg.TargetGroupName),
		ARN:        aws.ToString(tg.TargetGroupArn),
		Protocol:   string(tg.Protocol),
		Port:       int(aws.ToInt32(tg.Port)),
		TargetType: string(tg.TargetType),
	}

	healthOut, err := c.api.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: tg.TargetGroupArn,
	})
	if err != nil {
		return ELBTargetGroup{}, fmt.Errorf("DescribeTargetHealth for %s: %w", aws.ToString(tg.TargetGroupArn), err)
	}

	for _, th := range healthOut.TargetHealthDescriptions {
		if th.TargetHealth != nil {
			switch th.TargetHealth.State {
			case elbtypes.TargetHealthStateEnumHealthy:
				item.HealthyCount++
			case elbtypes.TargetHealthStateEnumUnhealthy:
				item.UnhealthyCount++
			}
		}
	}
	return item, nil
}

func (c *Client) ListListenerRules(ctx context.Context, listenerARN string) ([]ELBListenerRule, error) {
	out, err := c.api.DescribeRules(ctx, &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(listenerARN),
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeRules: %w", err)
	}

	rules := make([]ELBListenerRule, 0, len(out.Rules))
	for _, r := range out.Rules {
		priority := aws.ToString(r.Priority)
		isDefault := r.IsDefault != nil && *r.IsDefault

		var conditions []string
		for _, c := range r.Conditions {
			field := aws.ToString(c.Field)
			switch {
			case c.HostHeaderConfig != nil && len(c.HostHeaderConfig.Values) > 0:
				conditions = append(conditions, "host: "+joinStrings(c.HostHeaderConfig.Values))
			case c.PathPatternConfig != nil && len(c.PathPatternConfig.Values) > 0:
				conditions = append(conditions, "path: "+joinStrings(c.PathPatternConfig.Values))
			case c.HttpRequestMethodConfig != nil && len(c.HttpRequestMethodConfig.Values) > 0:
				conditions = append(conditions, "method: "+joinStrings(c.HttpRequestMethodConfig.Values))
			case c.SourceIpConfig != nil && len(c.SourceIpConfig.Values) > 0:
				conditions = append(conditions, "source-ip: "+joinStrings(c.SourceIpConfig.Values))
			default:
				if field != "" {
					conditions = append(conditions, field)
				}
			}
		}

		var actions []string
		for _, a := range r.Actions {
			actions = append(actions, formatAction([]elbtypes.Action{a}))
		}

		rules = append(rules, ELBListenerRule{
			ARN:        aws.ToString(r.RuleArn),
			Priority:   priority,
			Conditions: conditions,
			Actions:    actions,
			IsDefault:  isDefault,
		})
	}
	return rules, nil
}

func (c *Client) ListTargets(ctx context.Context, targetGroupARN string) ([]ELBTarget, error) {
	out, err := c.api.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupARN),
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTargetHealth: %w", err)
	}

	targets := make([]ELBTarget, 0, len(out.TargetHealthDescriptions))
	for _, th := range out.TargetHealthDescriptions {
		t := ELBTarget{}
		if th.Target != nil {
			t.ID = aws.ToString(th.Target.Id)
			t.Port = int(aws.ToInt32(th.Target.Port))
			t.AZ = aws.ToString(th.Target.AvailabilityZone)
		}
		if th.TargetHealth != nil {
			t.HealthState = string(th.TargetHealth.State)
			t.HealthReason = string(th.TargetHealth.Reason)
			t.HealthDesc = aws.ToString(th.TargetHealth.Description)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func (c *Client) GetLoadBalancerAttributes(ctx context.Context, lbARN string) ([]ELBAttribute, error) {
	out, err := c.api.DescribeLoadBalancerAttributes(ctx, &elbv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(lbARN),
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeLoadBalancerAttributes: %w", err)
	}

	attrs := make([]ELBAttribute, 0, len(out.Attributes))
	for _, a := range out.Attributes {
		attrs = append(attrs, ELBAttribute{
			Key:   aws.ToString(a.Key),
			Value: aws.ToString(a.Value),
		})
	}
	return attrs, nil
}

func (c *Client) GetResourceTags(ctx context.Context, arns []string) (map[string]map[string]string, error) {
	out, err := c.api.DescribeTags(ctx, &elbv2.DescribeTagsInput{
		ResourceArns: arns,
	})
	if err != nil {
		return nil, fmt.Errorf("DescribeTags: %w", err)
	}

	result := make(map[string]map[string]string, len(out.TagDescriptions))
	for _, td := range out.TagDescriptions {
		arn := aws.ToString(td.ResourceArn)
		tags := make(map[string]string, len(td.Tags))
		for _, t := range td.Tags {
			tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
		}
		result[arn] = tags
	}
	return result, nil
}

func joinStrings(ss []string) string {
	if len(ss) == 1 {
		return ss[0]
	}
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func formatAction(actions []elbtypes.Action) string {
	if len(actions) == 0 {
		return "—"
	}
	a := actions[0]
	switch a.Type {
	case elbtypes.ActionTypeEnumForward:
		name := utils.SecondToLast(aws.ToString(a.TargetGroupArn))
		return "forward → " + name
	case elbtypes.ActionTypeEnumRedirect:
		if a.RedirectConfig != nil {
			return "redirect → " + aws.ToString(a.RedirectConfig.Host)
		}
		return "redirect"
	case elbtypes.ActionTypeEnumFixedResponse:
		if a.FixedResponseConfig != nil {
			return "fixed-response " + aws.ToString(a.FixedResponseConfig.StatusCode)
		}
		return "fixed-response"
	default:
		return string(a.Type)
	}
}


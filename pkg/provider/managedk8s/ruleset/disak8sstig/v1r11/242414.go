// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1r11

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/diki/pkg/internal/utils"
	kubeutils "github.com/gardener/diki/pkg/kubernetes/utils"
	"github.com/gardener/diki/pkg/rule"
	"github.com/gardener/diki/pkg/shared/ruleset/disak8sstig/option"
	sharedv1r11 "github.com/gardener/diki/pkg/shared/ruleset/disak8sstig/v1r11"
)

var _ rule.Rule = &Rule242414{}

type Rule242414 struct {
	Client  client.Client
	Options *option.Options242414
}

func (r *Rule242414) ID() string {
	return sharedv1r11.ID242414
}

func (r *Rule242414) Name() string {
	return "The Kubernetes cluster must use non-privileged host ports for user pods (MEDIUM 242414)"
}

func (r *Rule242414) Run(ctx context.Context) (rule.RuleResult, error) {
	pods, err := kubeutils.GetPods(ctx, r.Client, "", labels.NewSelector(), 300)
	if err != nil {
		return rule.SingleCheckResult(r, rule.ErroredCheckResult(err.Error(), rule.NewTarget("kind", "podList"))), nil
	}

	namespaces, err := kubeutils.GetNamespaces(ctx, r.Client)
	if err != nil {
		return rule.SingleCheckResult(r, rule.ErroredCheckResult(err.Error(), rule.NewTarget("kind", "namespaceList"))), nil
	}
	checkResults := r.checkPods(pods, namespaces)

	return rule.RuleResult{
		RuleID:       r.ID(),
		RuleName:     r.Name(),
		CheckResults: checkResults,
	}, nil
}

func (r *Rule242414) checkPods(pods []corev1.Pod, namespaces map[string]corev1.Namespace) []rule.CheckResult {
	checkResults := []rule.CheckResult{}
	for _, pod := range pods {
		target := rule.NewTarget("name", pod.Name, "namespace", pod.Namespace, "kind", "pod")
		for _, container := range pod.Spec.Containers {
			uses := false
			for _, port := range container.Ports {
				if port.HostPort != 0 && port.HostPort < 1024 {
					target = target.With("details", fmt.Sprintf("containerName: %s, port: %d", container.Name, port.HostPort))
					if accepted, justification := r.accepted(pod, namespaces[pod.Namespace], port.HostPort); accepted {
						msg := "Container accepted to use hostPort < 1024."
						if justification != "" {
							msg = justification
						}
						checkResults = append(checkResults, rule.AcceptedCheckResult(msg, target))
					} else {
						checkResults = append(checkResults, rule.FailedCheckResult("Container uses hostPort < 1024.", target))
					}
					uses = true
				}
			}
			if !uses {
				checkResults = append(checkResults, rule.PassedCheckResult("Container does not use hostPort < 1024.", target))
			}
		}
	}
	return checkResults
}

func (r *Rule242414) accepted(pod corev1.Pod, namespace corev1.Namespace, hostPort int32) (bool, string) {
	if r.Options == nil {
		return false, ""
	}

	for _, acceptedPod := range r.Options.AcceptedPods {
		if utils.MatchLabels(pod.Labels, acceptedPod.PodMatchLabels) &&
			utils.MatchLabels(namespace.Labels, acceptedPod.NamespaceMatchLabels) {
			for _, acceptedHostPort := range acceptedPod.Ports {
				if acceptedHostPort == hostPort {
					return true, acceptedPod.Justification
				}
			}
		}
	}

	return false, ""
}

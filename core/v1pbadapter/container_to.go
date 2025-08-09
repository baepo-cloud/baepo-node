package v1pbadapter

import (
	"github.com/baepo-cloud/baepo-node/core/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
)

func ToContainerSpec(spec *corev1pb.ContainerSpec) *types.ContainerSpec {
	if spec == nil {
		return nil
	}

	return &types.ContainerSpec{
		Name:          spec.Name,
		Image:         spec.Image,
		LazyloadImage: spec.LazyloadImage,
		Env:           spec.Env,
		Command:       spec.Command,
		Healthcheck:   ToContainerHealthcheckSpec(spec.Healthcheck),
		WorkingDir:    spec.WorkingDir,
		Restart:       ToContainerRestartSpec(spec.Restart),
	}
}

func ToContainerHealthcheckSpec(spec *corev1pb.ContainerHealthcheckSpec) *types.ContainerHealthcheckSpec {
	if spec == nil {
		return nil
	}

	healthcheck := &types.ContainerHealthcheckSpec{
		InitialDelaySeconds: spec.InitialDelaySeconds,
		PeriodSeconds:       spec.PeriodSeconds,
	}
	switch typeSpec := spec.Type.(type) {
	case *corev1pb.ContainerHealthcheckSpec_Http:
		healthcheck.Http = ToContainerHttpHealthcheckSpec(typeSpec.Http)
	}
	return healthcheck
}

func ToContainerHttpHealthcheckSpec(spec *corev1pb.ContainerHealthcheckSpec_HttpHealthcheckSpec) *types.ContainerHttpHealthcheckSpec {
	if spec == nil {
		return nil
	}

	return &types.ContainerHttpHealthcheckSpec{
		Method:  spec.Method,
		Path:    spec.Path,
		Port:    spec.Port,
		Headers: spec.Headers,
	}
}

func ToContainerRestartSpec(spec *corev1pb.ContainerRestartSpec) *types.ContainerRestartSpec {
	if spec == nil {
		return nil
	}

	return &types.ContainerRestartSpec{
		Policy:     ToContainerRestartPolicy(spec.Policy),
		MaxRetries: spec.MaxRetries,
	}
}

func ToContainerRestartPolicy(policy corev1pb.ContainerRestartSpec_Policy) types.ContainerRestartPolicy {
	switch policy {
	case corev1pb.ContainerRestartSpec_Policy_No:
		return types.ContainerRestartPolicyNo
	case corev1pb.ContainerRestartSpec_Policy_OnFailure:
		return types.ContainerRestartPolicyOnFailure
	case corev1pb.ContainerRestartSpec_Policy_Always:
		return types.ContainerRestartPolicyAlways
	default:
		return types.ContainerRestartPolicyUnknown
	}
}

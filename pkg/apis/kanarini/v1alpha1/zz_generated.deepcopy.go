// +build !ignore_autogenerated

// Generated file, do not modify manually!

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeployment) DeepCopyInto(out *CanaryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeployment.
func (in *CanaryDeployment) DeepCopy() *CanaryDeployment {
	if in == nil {
		return nil
	}
	out := new(CanaryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CanaryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeploymentCondition) DeepCopyInto(out *CanaryDeploymentCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeploymentCondition.
func (in *CanaryDeploymentCondition) DeepCopy() *CanaryDeploymentCondition {
	if in == nil {
		return nil
	}
	out := new(CanaryDeploymentCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeploymentList) DeepCopyInto(out *CanaryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CanaryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeploymentList.
func (in *CanaryDeploymentList) DeepCopy() *CanaryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(CanaryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CanaryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeploymentSpec) DeepCopyInto(out *CanaryDeploymentSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	in.Template.DeepCopyInto(&out.Template)
	in.Tracks.DeepCopyInto(&out.Tracks)
	if in.ProgressDeadlineSeconds != nil {
		in, out := &in.ProgressDeadlineSeconds, &out.ProgressDeadlineSeconds
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeploymentSpec.
func (in *CanaryDeploymentSpec) DeepCopy() *CanaryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(CanaryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeploymentStatus) DeepCopyInto(out *CanaryDeploymentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]CanaryDeploymentCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CanaryDeploymentReadyStatusCheckpoint != nil {
		in, out := &in.CanaryDeploymentReadyStatusCheckpoint, &out.CanaryDeploymentReadyStatusCheckpoint
		*out = new(DeploymentReadyStatusCheckpoint)
		(*in).DeepCopyInto(*out)
	}
	if in.LatestSuccessfulDeploymentSnapshot != nil {
		in, out := &in.LatestSuccessfulDeploymentSnapshot, &out.LatestSuccessfulDeploymentSnapshot
		*out = new(DeploymentSnapshot)
		(*in).DeepCopyInto(*out)
	}
	if in.LatestFailedDeploymentSnapshot != nil {
		in, out := &in.LatestFailedDeploymentSnapshot, &out.LatestFailedDeploymentSnapshot
		*out = new(DeploymentSnapshot)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeploymentStatus.
func (in *CanaryDeploymentStatus) DeepCopy() *CanaryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(CanaryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryDeploymentTracks) DeepCopyInto(out *CanaryDeploymentTracks) {
	*out = *in
	in.Canary.DeepCopyInto(&out.Canary)
	in.Stable.DeepCopyInto(&out.Stable)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryDeploymentTracks.
func (in *CanaryDeploymentTracks) DeepCopy() *CanaryDeploymentTracks {
	if in == nil {
		return nil
	}
	out := new(CanaryDeploymentTracks)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CanaryTrackDeploymentSpec) DeepCopyInto(out *CanaryTrackDeploymentSpec) {
	*out = *in
	in.TrackDeploymentSpec.DeepCopyInto(&out.TrackDeploymentSpec)
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = make([]MetricSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CanaryTrackDeploymentSpec.
func (in *CanaryTrackDeploymentSpec) DeepCopy() *CanaryTrackDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(CanaryTrackDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrossVersionObjectReference) DeepCopyInto(out *CrossVersionObjectReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrossVersionObjectReference.
func (in *CrossVersionObjectReference) DeepCopy() *CrossVersionObjectReference {
	if in == nil {
		return nil
	}
	out := new(CrossVersionObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentReadyStatusCheckpoint) DeepCopyInto(out *DeploymentReadyStatusCheckpoint) {
	*out = *in
	in.LatestReadyTimestamp.DeepCopyInto(&out.LatestReadyTimestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentReadyStatusCheckpoint.
func (in *DeploymentReadyStatusCheckpoint) DeepCopy() *DeploymentReadyStatusCheckpoint {
	if in == nil {
		return nil
	}
	out := new(DeploymentReadyStatusCheckpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentSnapshot) DeepCopyInto(out *DeploymentSnapshot) {
	*out = *in
	in.Timestamp.DeepCopyInto(&out.Timestamp)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentSnapshot.
func (in *DeploymentSnapshot) DeepCopy() *DeploymentSnapshot {
	if in == nil {
		return nil
	}
	out := new(DeploymentSnapshot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalMetricSource) DeepCopyInto(out *ExternalMetricSource) {
	*out = *in
	in.Metric.DeepCopyInto(&out.Metric)
	in.Target.DeepCopyInto(&out.Target)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalMetricSource.
func (in *ExternalMetricSource) DeepCopy() *ExternalMetricSource {
	if in == nil {
		return nil
	}
	out := new(ExternalMetricSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricIdentifier) DeepCopyInto(out *MetricIdentifier) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricIdentifier.
func (in *MetricIdentifier) DeepCopy() *MetricIdentifier {
	if in == nil {
		return nil
	}
	out := new(MetricIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricSpec) DeepCopyInto(out *MetricSpec) {
	*out = *in
	if in.Object != nil {
		in, out := &in.Object, &out.Object
		*out = new(ObjectMetricSource)
		(*in).DeepCopyInto(*out)
	}
	if in.External != nil {
		in, out := &in.External, &out.External
		*out = new(ExternalMetricSource)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricSpec.
func (in *MetricSpec) DeepCopy() *MetricSpec {
	if in == nil {
		return nil
	}
	out := new(MetricSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricTarget) DeepCopyInto(out *MetricTarget) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		x := (*in).DeepCopy()
		*out = &x
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricTarget.
func (in *MetricTarget) DeepCopy() *MetricTarget {
	if in == nil {
		return nil
	}
	out := new(MetricTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectMetricSource) DeepCopyInto(out *ObjectMetricSource) {
	*out = *in
	out.DescribedObject = in.DescribedObject
	in.Target.DeepCopyInto(&out.Target)
	in.Metric.DeepCopyInto(&out.Metric)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectMetricSource.
func (in *ObjectMetricSource) DeepCopy() *ObjectMetricSource {
	if in == nil {
		return nil
	}
	out := new(ObjectMetricSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrackDeploymentSpec) DeepCopyInto(out *TrackDeploymentSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrackDeploymentSpec.
func (in *TrackDeploymentSpec) DeepCopy() *TrackDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(TrackDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

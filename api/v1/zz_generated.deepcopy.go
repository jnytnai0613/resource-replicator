//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
MIT License
Copyright (c) 2023 Junya Taniai

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterDetector) DeepCopyInto(out *ClusterDetector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDetector.
func (in *ClusterDetector) DeepCopy() *ClusterDetector {
	if in == nil {
		return nil
	}
	out := new(ClusterDetector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterDetector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterDetectorList) DeepCopyInto(out *ClusterDetectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterDetector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDetectorList.
func (in *ClusterDetectorList) DeepCopy() *ClusterDetectorList {
	if in == nil {
		return nil
	}
	out := new(ClusterDetectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterDetectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterDetectorSpec) DeepCopyInto(out *ClusterDetectorSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDetectorSpec.
func (in *ClusterDetectorSpec) DeepCopy() *ClusterDetectorSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterDetectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterDetectorStatus) DeepCopyInto(out *ClusterDetectorStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDetectorStatus.
func (in *ClusterDetectorStatus) DeepCopy() *ClusterDetectorStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterDetectorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentSpecApplyConfiguration) DeepCopyInto(out *DeploymentSpecApplyConfiguration) {
	clone := in.DeepCopy()
	*out = *clone
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressSpecApplyConfiguration) DeepCopyInto(out *IngressSpecApplyConfiguration) {
	clone := in.DeepCopy()
	*out = *clone
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PerResourceApplyStatus) DeepCopyInto(out *PerResourceApplyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PerResourceApplyStatus.
func (in *PerResourceApplyStatus) DeepCopy() *PerResourceApplyStatus {
	if in == nil {
		return nil
	}
	out := new(PerResourceApplyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Replicator) DeepCopyInto(out *Replicator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Replicator.
func (in *Replicator) DeepCopy() *Replicator {
	if in == nil {
		return nil
	}
	out := new(Replicator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Replicator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicatorList) DeepCopyInto(out *ReplicatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Replicator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicatorList.
func (in *ReplicatorList) DeepCopy() *ReplicatorList {
	if in == nil {
		return nil
	}
	out := new(ReplicatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicatorSpec) DeepCopyInto(out *ReplicatorSpec) {
	*out = *in
	if in.DeploymentSpec != nil {
		in, out := &in.DeploymentSpec, &out.DeploymentSpec
		*out = (*in).DeepCopy()
	}
	if in.ConfigMapData != nil {
		in, out := &in.ConfigMapData, &out.ConfigMapData
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ServiceSpec != nil {
		in, out := &in.ServiceSpec, &out.ServiceSpec
		*out = (*in).DeepCopy()
	}
	if in.IngressSpec != nil {
		in, out := &in.IngressSpec, &out.IngressSpec
		*out = (*in).DeepCopy()
	}
	if in.TargetCluster != nil {
		in, out := &in.TargetCluster, &out.TargetCluster
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicatorSpec.
func (in *ReplicatorSpec) DeepCopy() *ReplicatorSpec {
	if in == nil {
		return nil
	}
	out := new(ReplicatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicatorStatus) DeepCopyInto(out *ReplicatorStatus) {
	*out = *in
	if in.Applied != nil {
		in, out := &in.Applied, &out.Applied
		*out = make([]PerResourceApplyStatus, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicatorStatus.
func (in *ReplicatorStatus) DeepCopy() *ReplicatorStatus {
	if in == nil {
		return nil
	}
	out := new(ReplicatorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceSpecApplyConfiguration) DeepCopyInto(out *ServiceSpecApplyConfiguration) {
	clone := in.DeepCopy()
	*out = *clone
}

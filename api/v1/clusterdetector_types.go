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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterDetectorSpec defines the desired state of ClusterDetector
type ClusterDetectorSpec struct {
	// The kubeconfig file context,cluster,user
	Context string `json:"context,omitempty"`
	Cluster string `json:"cluster,omitempty"`
	User    string `json:"user,omitempty"`
}

// ClusterDetectorStatus defines the observed state of ClusterDetector
type ClusterDetectorStatus struct {
	// If communication to the remote Kubernetes cluster is possible,
	// Running is set; if not, Unknown is set.
	ClusterStatus string `json:"clusterstatus,omitempty"`

	// An error message is output when communication with a remote Kubernetes cluster is not possible.
	// Output only when the wide option of the Kubectl get command is given.
	Reason string `json:"reason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=cd
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="CONTEXT",type="string",JSONPath=".spec.context",description=
//+kubebuilder:printcolumn:name="CLUSTER",type="string",JSONPath=".spec.cluster"
//+kubebuilder:printcolumn:name="USER",type="string",JSONPath=".spec.user"
//+kubebuilder:printcolumn:name="CLUSTERSTATUS",type="string",JSONPath=".status.clusterstatus"
//+kubebuilder:printcolumn:name="REASON",type="string",priority=1,JSONPath=".status.reason"
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterDetector is the Schema for the clusterdetectors API
type ClusterDetector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDetectorSpec   `json:"spec,omitempty"`
	Status ClusterDetectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterDetectorList contains a list of ClusterDetector
type ClusterDetectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDetector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterDetector{}, &ClusterDetectorList{})
}

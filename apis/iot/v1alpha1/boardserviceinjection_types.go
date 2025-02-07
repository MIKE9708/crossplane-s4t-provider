/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// BoardServiceInjectionParameters are the configurable fields of a BoardServiceInjection.
type BoardServiceInjectionParameters struct {
	BoardUuid   string `json:"boardUuid,omitempty"`
	ServiceUuid string `json:"serviceUuid,omitempty"`
}

// BoardServiceInjectionObservation are the observable fields of a BoardServiceInjection.
type BoardServiceInjectionObservation struct {
	ServiceUuid string `json:"serviceUuid,omitempty"`
	BoardUuid   string `json:"boardUuid,omitempty"`
}

// A BoardServiceInjectionSpec defines the desired state of a BoardServiceInjection.
type BoardServiceInjectionSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BoardServiceInjectionParameters `json:"forProvider"`
}

// A BoardServiceInjectionStatus represents the observed state of a BoardServiceInjection.
type BoardServiceInjectionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BoardServiceInjectionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A BoardServiceInjection is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,s4t}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BoardServiceInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BoardServiceInjectionSpec   `json:"spec"`
	Status BoardServiceInjectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BoardServiceInjectionList contains a list of BoardServiceInjection
type BoardServiceInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BoardServiceInjection `json:"items"`
}

// BoardServiceInjection type metadata.
var (
	BoardServiceInjectionKind             = reflect.TypeOf(BoardServiceInjection{}).Name()
	BoardServiceInjectionGroupKind        = schema.GroupKind{Group: Group, Kind: BoardServiceInjectionKind}.String()
	BoardServiceInjectionKindAPIVersion   = BoardServiceInjectionKind + "." + SchemeGroupVersion.String()
	BoardServiceInjectionGroupVersionKind = SchemeGroupVersion.WithKind(BoardServiceInjectionKind)
)

func init() {
	SchemeBuilder.Register(&BoardServiceInjection{}, &BoardServiceInjectionList{})
}

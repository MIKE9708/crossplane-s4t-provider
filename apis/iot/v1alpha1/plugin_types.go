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
	"k8s.io/apimachinery/pkg/runtime"
)

// PluginParameters are the configurable fields of a Plugin.
type PluginParameters struct {
	// +kubebuilder:validation:Immutable
	Uuid       string               `json:"uuid,omitempty"`
	Name       string               `json:"name"`
	Parameters runtime.RawExtension `json:"parameters"`
	Code       string               `json:"code"`
	// +kubebuilder:validation:Immutable
	Version string `json:"version,omitempty"`
}

// PluginObservation are the observable fields of a Plugin.
type PluginObservation struct {
	Name string `json:"name"`
}

// A PluginSpec defines the desired state of a Plugin.
type PluginSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PluginParameters `json:"forProvider"`
}

// A PluginStatus represents the observed state of a Plugin.
type PluginStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PluginObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=".spec.Name"
// A Plugin is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,s4t}
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PluginSpec   `json:"spec"`
	Status PluginStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PluginList contains a list of Plugin
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Plugin `json:"items"`
}

// Plugin type metadata.
var (
	PluginKind             = reflect.TypeOf(Plugin{}).Name()
	PluginGroupKind        = schema.GroupKind{Group: Group, Kind: PluginKind}.String()
	PluginKindAPIVersion   = PluginKind + "." + SchemeGroupVersion.String()
	PluginGroupVersionKind = SchemeGroupVersion.WithKind(PluginKind)
)

func init() {
	SchemeBuilder.Register(&Plugin{}, &PluginList{})
}

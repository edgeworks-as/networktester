/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworktestSpec defines the desired state of Networktest
type NetworktestSpec struct {
	Interval string `json:"interval"` // Default 1h
	Timeout  int    `json:"timeout"`

	// +optional
	Http *HttpProbe `json:"http"`

	// +optional
	TCP *TCPProbe `json:"tcp"`
}

type HttpProbe struct {
	URL string `json:"url"`
}

type TCPProbe struct {
	Address string `json:"address"`
	Port    int    `json:"port"`

	// +optional
	Data string `json:"data,omitempty"`
}

func (s NetworktestSpec) GetInterval() string {
	if s.Interval == "" {
		return "1h"
	}
	return s.Interval
}

// NetworktestStatus defines the observed state of Networktest
type NetworktestStatus struct {
	Active     bool               `json:"active,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	LastRun *metav1.Time `json:"lastRun"`

	// +optional
	NextRun *metav1.Time `json:"nextRun"`

	// +optional
	LastResult *string `json:"lastResult"`

	// +optional
	Message *string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.lastResult",name=LastResult,type=string
//+kubebuilder:printcolumn:JSONPath=".status.lastRun",name=LastRun,type=string

// Networktest is the Schema for the networktests API
type Networktest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworktestSpec   `json:"spec,omitempty"`
	Status NetworktestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NetworktestList contains a list of Networktest
type NetworktestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Networktest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Networktest{}, &NetworktestList{})
}

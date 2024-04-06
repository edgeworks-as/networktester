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
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworktestSpec defines the desired state of Networktest
type NetworktestSpec struct {

	// +kubebuilder:default:="1h"
	// interval defines how often the probing will be done. Defaults to 1h. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	Interval string `json:"interval"` // Default 1h

	// +kubebuilder:default:=5
	// timeout in seconds until the probe is considered failed. Default is 5 seconds.
	Timeout int `json:"timeout"`

	// +kubebuilder:default:=true
	// enabled lets you disable rules without deleting them. Default true.
	Enabled bool `json:"enabled,omitempty"`

	// +optional
	// http defines settings for probing using http client
	Http *HttpProbe `json:"http"`

	// +optional
	// tcp defines settings for probing using plain sockets
	TCP *TCPProbe `json:"tcp"`

	// +optional
	// limit number of probe result transitions to keep in the status. Default 0 - no limit.
	HistoryLimit int `json:"historyLimit"`
}

type HttpProbe struct {
	// url must be valid http/https url
	URL string `json:"url"`

	// failOnCodes lists the HTTP codes that should fail the test. Empty list means a successful HTTP request means the test is good.
	FailOnCodes []int `json:"failOnCodes,omitempty"`

	// tlsSkipVerify allows optional https without verifying server certificate (default: false)
	// +optional
	TlsSkipVerify bool `json:"tlsSkipVerify,omitempty"`
}

type TCPProbe struct {
	// address must be valid IP address or host name
	Address string `json:"address"`

	// port must be valid port
	Port int `json:"port"`

	// +optional
	Data string `json:"data,omitempty"`
}

func (s *NetworktestSpec) GetAddress() string {
	if s.Http != nil {
		return fmt.Sprintf("%s", s.Http.URL)
	} else if s.TCP != nil {
		return fmt.Sprintf("tcp://%s:%d", s.TCP.Address, s.TCP.Port)
	} else {
		return "<undefined>"
	}
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

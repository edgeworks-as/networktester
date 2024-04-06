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

package controllers

import (
	"context"
	"edgeworks.no/networktester/pkg/testers"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	edgeworksnov1 "edgeworks.no/networktester/api/v1"
)

var testResult = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "networktester_probe",
		Help: "Result of Networktester probe run",
	}, []string{"namespace", "name", "address"})

func init() {
	metrics.Registry.Register(testResult)
}

// NetworktestReconciler reconciles a Networktest object
type NetworktestReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Tests       sync.Map
	TriggerChan chan struct{}
}

type Probe struct {
	NextRun    time.Time
	Name       types.NamespacedName
	Generation int64
}

func calcNextRun(i string) time.Time {
	now := time.Now()
	interval, _ := time.ParseDuration(i)
	return now.Add(interval)
}

//+kubebuilder:rbac:groups=edgeworks.no,resources=networktests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edgeworks.no,resources=networktests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edgeworks.no,resources=networktests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Networktest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *NetworktestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var test edgeworksnov1.Networktest
	if err := r.Get(ctx, req.NamespacedName, &test); err != nil {
		if k8errors.IsNotFound(err) {
			ctrl.Log.V(1).Info(fmt.Sprintf("Removed %s", req.NamespacedName.String()))
			r.Tests.Delete(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		ctrl.Log.Error(err, "Failed to get Networktest")
		return ctrl.Result{}, err
	}

	if !test.Status.Active && test.Spec.Enabled {
		accepted := true
		var message string

		// Verify and set status
		if test.Spec.Http != nil && test.Spec.Http.URL != "" {
			if _, err := url.Parse(test.Spec.Http.URL); err != nil {
				message = fmt.Errorf("Failed to parse URL: %v", err).Error()
				accepted = false
			}

		} else if test.Spec.TCP != nil && test.Spec.TCP.Address != "" {
			if test.Spec.TCP.Port <= 0 {
				message = fmt.Errorf("invalid port: %d", test.Spec.TCP.Port).Error()
				accepted = false
			}
		}

		test.Status.Active = accepted
		test.Status.Message = &message
	} else if !test.Spec.Enabled && test.Status.Active {
		test.Status.Conditions = []metav1.Condition{}
		test.Status.Active = false
		test.Status.NextRun = nil
		test.Status.LastRun = nil
		test.Status.LastResult = nil
		disabled := "Disabled"
		test.Status.Message = &disabled
	}

	if err := r.Status().Update(ctx, &test); err != nil {
		ctrl.Log.Error(err, "Failed to update status of Networktest")
		return ctrl.Result{}, err
	}

	if test.Status.Active {
		// Either add or replace probe
		if probe, found := r.Tests.Load(req.NamespacedName.String()); !found {
			probe := Probe{
				Name:       req.NamespacedName,
				Generation: test.Generation,
				NextRun:    time.Now(),
			}

			r.Tests.Store(req.NamespacedName.String(), &probe)
			ctrl.Log.V(1).Info(fmt.Sprintf("Added %s", req.NamespacedName.String()))
			r.TriggerChan <- struct{}{}
		} else {
			p := probe.(*Probe)
			if p.Generation != test.Generation {
				p.NextRun = time.Now()
				p.Generation = test.Generation
				r.Tests.Swap(req.NamespacedName.String(), p)
				ctrl.Log.V(1).Info(fmt.Sprintf("Updated %s", req.NamespacedName.String()))
				r.TriggerChan <- struct{}{}
			}
		}
	} else {
		// Remove probe
		r.Tests.Delete(req.NamespacedName.String())
		ctrl.Log.V(1).Info(fmt.Sprintf("Deactivated %s", req.NamespacedName.String()))
	}

	return ctrl.Result{}, nil
}

func (r *NetworktestReconciler) tester() {
	for {
		now := time.Now()
		r.Tests.Range(func(n, p any) bool {
			probe := p.(*Probe)
			if probe.NextRun.Before(now) {
				go r.performTest(probe)
			}
			return true
		})

		select {
		case <-r.TriggerChan:
		case <-time.After(30 * time.Second):
		}
	}
}

func (r *NetworktestReconciler) performTest(p *Probe) {

	// Get resource, so we update the same as we are testing
	var t edgeworksnov1.Networktest
	if err := r.Get(context.Background(), p.Name, &t); err != nil {
		ctrl.Log.Error(err, "failed to get Networktest")
		return
	}
	ctrl.Log.V(1).Info("Testing", "namespace", t.Namespace, "name", t.Name, "generation", t.ObjectMeta.Generation)

	// Calculate next run time before doing t, to ensure we keep up with the interval start to start
	p.NextRun = calcNextRun(t.Spec.Interval)
	now := metav1.NewTime(time.Now())
	currentResourceVersion := t.ResourceVersion

	// Perform t
	result, err := testers.PerformTest(&t)
	if err != nil {
		ctrl.Log.Info("Unknown probe type", "namespace", t.Namespace, "name", t.Name)
		return
	}

	// Update metrics
	testResult.WithLabelValues(p.Name.Namespace, p.Name.Name, t.Spec.GetAddress()).Set(getCondValue(result))

	// Get again in case updated in the mean time
	if err := r.Get(context.Background(), p.Name, &t); err != nil {
		ctrl.Log.Error(err, "failed to get Networktest")
		return
	}

	if currentResourceVersion != t.ResourceVersion {
		ctrl.Log.Info("Definition changed during testing. Skipping writing status.", "namespace", t.Namespace, "name", t.Name)
	}

	t.Status.LastResult = result.String()
	t.Status.Message = &result.Message
	t.Status.LastRun = &now

	next := metav1.NewTime(p.NextRun)
	t.Status.NextRun = &next

	cond := metav1.Condition{
		Type:               "Probe",
		Reason:             "Probe",
		Status:             getCondStatus(result),
		ObservedGeneration: t.ObjectMeta.Generation,
		LastTransitionTime: now,
		Message:            *t.Status.Message,
	}

	if len(t.Status.Conditions) == 0 {
		t.Status.Conditions = append(t.Status.Conditions, cond)
	} else {
		if t.Status.Conditions[len(t.Status.Conditions)-1].Status != cond.Status || t.Status.Conditions[len(t.Status.Conditions)-1].ObservedGeneration != cond.ObservedGeneration {
			t.Status.Conditions = append(t.Status.Conditions, cond)
		}
	}

	if t.Spec.HistoryLimit != 0 && len(t.Status.Conditions) > t.Spec.HistoryLimit {
		t.Status.Conditions = t.Status.Conditions[len(t.Status.Conditions)-t.Spec.HistoryLimit:]
	}

	if err = r.Status().Update(context.Background(), &t); err != nil {
		ctrl.Log.Info("Could not update status: "+err.Error(), "namespace", t.Namespace, "name", t.Name)
	}

}

func getCondStatus(result testers.TestResult) metav1.ConditionStatus {
	if result.Success {
		return "True"
	} else {
		return "False"
	}
}

func getCondValue(result testers.TestResult) float64 {
	switch result.Success {
	case true:
		return 1
	default:
		return 0
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworktestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	go r.tester()
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeworksnov1.Networktest{}).
		Complete(r)
}

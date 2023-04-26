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
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	edgeworksnov1 "edgeworks.no/networktester/api/v1"
)

// NetworktestReconciler reconciles a Networktest object
type NetworktestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Tests  map[string]*Probe
}

type Probe struct {
	Resource *edgeworksnov1.Networktest
	NextRun  time.Time
}

func (p Probe) CalcNextRun() time.Time {
	now := time.Now()
	interval, _ := time.ParseDuration(p.Resource.Spec.Interval)
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
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		ctrl.Log.Error(err, "Failed to get Networktest")
		return ctrl.Result{}, err
	}

	//ctrl.Log.Info("Got object: " + req.NamespacedName.String() + " (version: " + test.ObjectMeta.ResourceVersion + ")")

	if !test.Status.Accepted {
		// Verify and set status
		if test.Spec.Http != nil && test.Spec.Http.URL != "" {
			if _, err := url.Parse(test.Spec.Http.URL); err != nil {
				return ctrl.Result{}, fmt.Errorf("Failed to parse URL: %v", err)
			}

		} else if test.Spec.TCP != nil && test.Spec.TCP.Address != "" {
			if test.Spec.TCP.Port <= 0 {
				return ctrl.Result{}, fmt.Errorf("invalid port: %d", test.Spec.TCP.Port)
			}
		}

		test.Status.Accepted = true

		if err := r.Status().Update(ctx, &test); err != nil {
			ctrl.Log.Error(err, "Failed to update status of Networktest")
			return ctrl.Result{}, err
		}
	}

	if test.Status.Accepted {
		if probe, found := r.Tests[req.NamespacedName.String()]; !found {
			probe := Probe{
				Resource: test.DeepCopy(),
				NextRun:  time.Now(),
			}
			r.Tests[req.NamespacedName.String()] = &probe
		} else {
			if probe.Resource.ResourceVersion != test.ResourceVersion {
				probe.Resource = test.DeepCopy()
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NetworktestReconciler) tester() {

	for {
		now := time.Now()
		for _, probe := range r.Tests {
			if probe.NextRun.Before(now) {
				go r.performTest(probe)
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func (r *NetworktestReconciler) performTest(p *Probe) {
	res := p.Resource
	ctrl.Log.V(1).Info(fmt.Sprintf("Testing %s", res.Name))

	p.NextRun = p.CalcNextRun()

	var result testers.TestResult
	if res.Spec.Http != nil {
		result = testers.DoHttpTest(res)
	} else if res.Spec.TCP != nil {
		result = testers.DoTCPTest(res)
	} else {
		ctrl.Log.Info("Unknown probe type")
		return
	}

	var test edgeworksnov1.Networktest
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: res.ObjectMeta.Namespace, Name: res.ObjectMeta.Name}, &test); err == nil {
		test.Status.LastResult = result.String()
		test.Status.Message = &result.Message
		now := metav1.NewTime(time.Now())
		test.Status.LastRun = &now

		next := metav1.NewTime(p.NextRun)
		test.Status.NextRun = &next

		r.Status().Update(context.Background(), &test)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworktestReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.Tests = make(map[string]*Probe)

	go r.tester()

	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeworksnov1.Networktest{}).
		Complete(r)
}

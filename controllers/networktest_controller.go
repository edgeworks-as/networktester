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
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net"
	"net/http"
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
	Tests  map[string]*edgeworksnov1.Networktest
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
				return ctrl.Result{}, fmt.Errorf("failed to parse URL: %v", err)
			}
		} else if test.Spec.TCP != nil && test.Spec.TCP.Address != "" {
			if addr := net.ParseIP(test.Spec.TCP.Address); addr == nil {
				return ctrl.Result{}, fmt.Errorf("failed to parse IP: %s", test.Spec.TCP.Address)
			}

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
		if ct, found := r.Tests[req.NamespacedName.String()]; !found {
			r.Tests[req.NamespacedName.String()] = test.DeepCopy()
		} else {
			if ct.ResourceVersion != test.ResourceVersion {
				r.Tests[req.NamespacedName.String()] = test.DeepCopy()
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NetworktestReconciler) tester() {

	for {
		for n, t := range r.Tests {
			ctrl.Log.Info(fmt.Sprintf("Testing %s", n))

			var result testResult

			if t.Spec.Http != nil {
				result = doHttpTest(t)
			} else {
				ctrl.Log.Info("Unknown probe type")
				continue
			}

			var test edgeworksnov1.Networktest
			if err := r.Get(context.Background(), types.NamespacedName{Namespace: t.ObjectMeta.Namespace, Name: t.ObjectMeta.Name}, &test); err == nil {
				test.Status.LastResult = result.String()
				test.Status.Message = &result.message
				now := metav1.NewTime(time.Now())
				test.Status.LastRun = &now

				d, _ := time.ParseDuration(test.Spec.GetInterval())
				next := metav1.NewTime(time.Now().Add(d))
				test.Status.NextRun = &next

				r.Status().Update(context.Background(), &test)
			}
		}

		time.Sleep(10 * time.Second)
	}
}

type testResult struct {
	success bool
	message string
}

const (
	success = "Success"
	failed  = "Failed"
)

func (t testResult) String() *string {
	var res string
	switch t.success {
	case true:
		res = success
	default:
		res = failed
	}
	return &res
}

func doHttpTest(t *edgeworksnov1.Networktest) testResult {
	timeout, _ := time.ParseDuration(fmt.Sprintf("%ds", t.Spec.Timeout))
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, t.Spec.Http.URL, nil)
	if err != nil {
		return testResult{
			success: false,
			message: err.Error(),
		}
	}

	c := http.Client{}
	res, err := c.Do(r)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return testResult{
				success: false,
				message: fmt.Errorf("timeout: %v", err).Error(),
			}
		}

		return testResult{
			success: false,
			message: err.Error(),
		}
	}

	return testResult{
		success: res.StatusCode == 200,
		message: fmt.Sprintf("http result: %s", res.Status),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworktestReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.Tests = make(map[string]*edgeworksnov1.Networktest)

	go r.tester()

	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeworksnov1.Networktest{}).
		Owns(&v1.Pod{}).
		Complete(r)
}

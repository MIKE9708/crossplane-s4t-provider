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

package device

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	s4t "github.com/MIKE9708/s4t-sdk-go/pkg/api"
	read_config "github.com/MIKE9708/s4t-sdk-go/pkg/read_conf"

	boards "github.com/MIKE9708/s4t-sdk-go/pkg/api/data/board"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/provider-s4t/apis/iot/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-s4t/apis/v1alpha1"
	"github.com/crossplane/provider-s4t/internal/features"
	"k8s.io/apimachinery/pkg/types"
)

const (
	errNotDevice    = "managed resource is not a Device custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type S4TService struct {
	S4tClient *s4t.Client
}

var (
	newS4TService = func(creds []byte) (*S4TService, error) {
		var result map[string]string
		err := json.Unmarshal(creds, &result)
		if err != nil {
			return nil, errors.Wrap(err, errNewClient)
		}
		auth_req := read_config.FormatAuthRequ(
			result["username"],
			result["password"],
			result["domain"],
		)
		s4t_client, err := s4t.GetClientConnection(*auth_req)
		return &S4TService{
			S4tClient: s4t_client,
		}, err
	}
)

// Setup adds a controller that reconciles Device managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.DeviceGroupKind)
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.DeviceGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newS4TService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Device{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*S4TService, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Device)
	if !ok {
		return nil, errors.New(errNotDevice)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}
	// cr.GetProviderConfigReference().Name
	pc_domain := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: "s4t-provider-domain"}, pc_domain); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}
	cd_domain := pc_domain.Spec.Credentials
	data_domain, err := resource.CommonCredentialExtractor(ctx, cd_domain.Source, c.kube, cd_domain.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}
	svc, err := c.newServiceFn(data_domain)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return &external{service: svc}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service *S4TService
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Device)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDevice)
	}
	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)
	board, err := c.service.S4tClient.GetBoardDetail(cr.Spec.ForProvider.Uuid)
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client Board Get %q", err)
		return managed.ExternalObservation{}, err
	}

	if board.Uuid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	if cr.Spec.ForProvider.Code != board.Code {
		return managed.ExternalObservation{ResourceUpToDate: false, ResourceExists: true}, nil
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Device)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDevice)
	}
	board := boards.Board{}

	board.Name = cr.Spec.ForProvider.Name
	board.Code = cr.Spec.ForProvider.Code
	board.Type = cr.Spec.ForProvider.Type

	for _, location := range cr.Spec.ForProvider.Location {
		board.Location = append(board.Location, &boards.Location{
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Altitude:  location.Altitude,
		})
	}

	res, err := c.service.S4tClient.CreateBoard(board)
	// log.Printf("\n\n####ERROR-LOG#### %s\n\n", res)
	if err != nil {
		// log.Printf("\n\n####ERROR-LOG#### %s\n\n", res)
		log.Printf("####ERROR-LOG####  Error s4t client Board Create %q", err)
		return managed.ExternalCreation{}, errors.New(errNewClient)
	}

	cr.Spec.ForProvider.Uuid = res.Uuid
	cr.Spec.ForProvider.Type = res.Type
	cr.Spec.ForProvider.Agent = res.Agent
	cr.Spec.ForProvider.Status = res.Status
	cr.Spec.ForProvider.Session = res.Session

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, err
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Device)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDevice)
	}

	fmt.Printf("Updating: %+v", cr)
	resp, err := c.service.S4tClient.PatchBoard(cr.Spec.ForProvider.Uuid,
		map[string]interface{}{
			"code": cr.Spec.ForProvider.Code,
		})
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client Board Update %q", err)
		return managed.ExternalUpdate{}, errors.New(errNewClient)
	}

	cr.Spec.ForProvider.Uuid = resp.Uuid
	cr.Spec.ForProvider.Code = resp.Code
	cr.Spec.ForProvider.Status = resp.Status
	cr.Spec.ForProvider.Name = resp.Name
	cr.Spec.ForProvider.Session = resp.Session
	cr.Spec.ForProvider.Wstunip = resp.Wstunip
	cr.Spec.ForProvider.Type = resp.Type
	cr.Spec.ForProvider.LRversion = resp.LRversion

	// MUST BE ADAPTED
	cr.Status.Uuid = resp.Uuid
	cr.Status.Status = resp.Status

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, err
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Device)
	if !ok {
		return errors.New(errNotDevice)
	}
	fmt.Printf("Deleting: %+v", cr)
	err := c.service.S4tClient.DeleteBoard(cr.Spec.ForProvider.Uuid)
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client Board Delete %q", err)
	}
	return err
}

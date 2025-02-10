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

package boardplugininjection

import (
	"context"
	"fmt"
	"github.com/MIKE9708/s4t-sdk-go/pkg/api"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/provider-s4t/apis/iot/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-s4t/apis/v1alpha1"
	"github.com/crossplane/provider-s4t/internal/features"
	"github.com/pkg/errors"
	_ "k8s.io/apimachinery/pkg/types"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotBoardPluginInjection = "managed resource is not a BoardPluginInjection custom resource"
	errTrackPCUsage            = "cannot track ProviderConfig usage"
	errGetPC                   = "cannot get ProviderConfig"
	errGetCreds                = "cannot get credentials"
	errNewClient               = "cannot create new Service"
)

type S4TService struct {
	S4tClient *s4t.Client
}

var (
	newS4TService = func(_ []byte) (*S4TService, error) {
		s4t := s4t.Client{}
		s4t_client, err := s4t.GetClientConnection()
		return &S4TService{
			S4tClient: s4t_client,
		}, err
	}
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.BoardPluginInjectionGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.BoardPluginInjectionGroupVersionKind),
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
		For(&v1alpha1.BoardPluginInjection{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*S4TService, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	// cr, ok := mg.(*v1alpha1.BoardPluginInjection)
	// if !ok {
	// 	return nil, errors.New(errNotBoardPluginInjection)
	// }

	// if err := c.usage.Track(ctx, mg); err != nil {
	// 	return nil, errors.Wrap(err, errTrackPCUsage)
	// }

	// pc := &apisv1alpha1.ProviderConfig{}
	// if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
	// 	return nil, errors.Wrap(err, errGetPC)
	// }

	// cd := pc.Spec.Credentials
	// data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	// if err != nil {
	// 	return nil, errors.Wrap(err, errGetCreds)
	// }

	// svc, err := c.newServiceFn(data)
	// if err != nil {
	// 	return nil, errors.Wrap(err, errNewClient)
	// }

	// return &external{service: svc}, nil
	return nil, nil
}

type external struct {
	service *S4TService
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.BoardPluginInjection)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBoardPluginInjection)
	}
	fmt.Printf("Observing: %+v", cr)

	injectedPlugin, err := c.service.S4tClient.GetBoardPlugins(cr.Spec.ForProvider.BoardUuid)
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client Board Get %q", err)
		return managed.ExternalObservation{}, err
	}

	found := false
	for _, plugin := range injectedPlugin {
		if plugin.UUID == cr.Spec.ForProvider.PluginUuid {
			found = true
			break
		}
	}
	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.BoardPluginInjection)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBoardPluginInjection)
	}

	fmt.Printf("Creating: %+v", cr)
	err := c.service.S4tClient.InjectPLuginBoard(
		cr.Spec.ForProvider.BoardUuid,
		map[string]interface{}{
			"plugin": cr.Spec.ForProvider.PluginUuid,
		})
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client BoardPlugin Inject %q", err)
		return managed.ExternalCreation{}, err
	}
	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.BoardPluginInjection)
	if !ok {
		return errors.New(errNotBoardPluginInjection)
	}

	fmt.Printf("Deleting: %+v", cr)
	err := c.service.S4tClient.RemoveInjectedPlugin(cr.Spec.ForProvider.PluginUuid, cr.Spec.ForProvider.BoardUuid)
	if err != nil {
		log.Printf("####ERROR-LOG#### Error s4t client BoardPlugin Delete %q", err)
		return err
	}
	return nil
}

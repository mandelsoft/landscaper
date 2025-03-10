// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	"github.com/gardener/landscaper/apis/config"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// NewEnvironmentController creates a new landscaper agent EnvironmentController.
func NewEnvironmentController(log logr.Logger, c client.Client, scheme *runtime.Scheme, config *config.LandscaperConfiguration) reconcile.Reconciler {
	return &EnvironmentController{
		log:    log,
		client: c,
		scheme: scheme,
		config: config,
		dm: &DeployerManagement{
			log:    log,
			client: c,
			scheme: scheme,
			config: config.DeployerManagement,
		},
	}
}

type EnvironmentController struct {
	log    logr.Logger
	config *config.LandscaperConfiguration
	client client.Client
	scheme *runtime.Scheme
	dm     *DeployerManagement
}

func (con *EnvironmentController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())

	env := &lsv1alpha1.Environment{}
	if err := con.client.Get(ctx, req.NamespacedName, env); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	registrations := &lsv1alpha1.DeployerRegistrationList{}
	if err := con.client.List(ctx, registrations); err != nil {
		return reconcile.Result{}, err
	}

	if !env.DeletionTimestamp.IsZero() {
		var (
			errList []error
			mut     = sync.Mutex{}
			wg      = sync.WaitGroup{}
		)
		for _, registration := range registrations.Items {
			wg.Add(1)
			go func(registration lsv1alpha1.DeployerRegistration) {
				defer wg.Done()
				if err := con.dm.Delete(ctx, &registration, env); err != nil {
					mut.Lock()
					errList = append(errList, err)
					mut.Unlock()
				}
			}(registration)
		}
		wg.Wait()
		err := errors.NewAggregate(errList)
		if err != nil {
			return reconcile.Result{}, err
		}
		controllerutil.RemoveFinalizer(env, lsv1alpha1.LandscaperDMFinalizer)
		if err := con.client.Update(ctx, env); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
		}
		return reconcile.Result{}, nil
	}

	// ensure finalizer
	if !controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer) {
		controllerutil.AddFinalizer(env, lsv1alpha1.LandscaperDMFinalizer)

		if err := con.client.Update(ctx, env); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to add finalizer: %w", err)
		}
	}

	// ensure the target
	targetTemplate := env.Spec.HostTarget
	target := &lsv1alpha1.Target{}
	target.Name = env.Name
	target.Namespace = con.config.DeployerManagement.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, con.client, target, func() error {
		target.Annotations = targetTemplate.Annotations
		target.Labels = targetTemplate.Labels
		target.Spec = lsv1alpha1.TargetSpec{
			Type:          targetTemplate.Type,
			Configuration: targetTemplate.Configuration,
		}
		return controllerutil.SetControllerReference(env, target, con.scheme)
	}); err != nil {
		return reconcile.Result{}, err
	}

	for _, registration := range registrations.Items {
		if err := con.dm.Reconcile(ctx, &registration, env); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// NewDeployerRegistrationController creates a new landscaper agent DeployerRegistrationController.
func NewDeployerRegistrationController(log logr.Logger, c client.Client, scheme *runtime.Scheme, config *config.LandscaperConfiguration) reconcile.Reconciler {
	return &DeployerRegistrationController{
		log:    log,
		client: c,
		dm: &DeployerManagement{
			log:    log,
			client: c,
			scheme: scheme,
			config: config.DeployerManagement,
		},
	}
}

type DeployerRegistrationController struct {
	log    logr.Logger
	client client.Client
	dm     *DeployerManagement
}

func (con *DeployerRegistrationController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())

	registration := &lsv1alpha1.DeployerRegistration{}
	if err := con.client.Get(ctx, req.NamespacedName, registration); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(registration, lsv1alpha1.LandscaperDMFinalizer) {
		controllerutil.AddFinalizer(registration, lsv1alpha1.LandscaperDMFinalizer)

		if err := con.client.Update(ctx, registration); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to add finalizer: %w", err)
		}
	}

	environments := &lsv1alpha1.EnvironmentList{}
	if err := con.client.List(ctx, environments); err != nil {
		return reconcile.Result{}, err
	}

	if !registration.DeletionTimestamp.IsZero() {
		var (
			errList []error
			mut     = sync.Mutex{}
			wg      = sync.WaitGroup{}
		)
		for _, env := range environments.Items {
			wg.Add(1)
			go func(env lsv1alpha1.Environment) {
				defer wg.Done()
				if err := con.dm.Delete(ctx, registration, &env); err != nil {
					mut.Lock()
					errList = append(errList, err)
					mut.Unlock()
				}
			}(env)
		}
		wg.Wait()
		err := errors.NewAggregate(errList)
		if err != nil {
			return reconcile.Result{}, err
		}
		controllerutil.RemoveFinalizer(registration, lsv1alpha1.LandscaperDMFinalizer)
		if err := con.client.Update(ctx, registration); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
		}
		return reconcile.Result{}, nil
	}

	for _, env := range environments.Items {
		if err := con.dm.Reconcile(ctx, registration, &env); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

type InstallationController struct {
	log    logr.Logger
	config *config.LandscaperConfiguration
	client client.Client
	dm     *DeployerManagement
}

// NewInstallationController creates a new landscaper agent InstallationController.
// This controller only reconciles deployer installations and its main purpose is cleanup.
func NewInstallationController(log logr.Logger, c client.Client, scheme *runtime.Scheme, config *config.LandscaperConfiguration) reconcile.Reconciler {
	return &InstallationController{
		log:    log,
		config: config,
		client: c,
		dm: &DeployerManagement{
			log:    log,
			client: c,
			scheme: scheme,
			config: config.DeployerManagement,
		},
	}
}

func (con *InstallationController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())

	if req.Namespace != con.config.DeployerManagement.Namespace {
		return reconcile.Result{}, nil
	}

	inst := &lsv1alpha1.Installation{}
	if err := con.client.Get(ctx, req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if inst.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}
	if !kutil.HasLabel(inst, lsv1alpha1.DeployerEnvironmentLabelName) {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, con.dm.CleanupInstallation(ctx, inst)
}

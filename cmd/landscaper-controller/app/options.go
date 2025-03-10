// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	goflag "flag"
	"io/ioutil"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	ctrl "sigs.k8s.io/controller-runtime"

	deployerconfig "github.com/gardener/landscaper/pkg/deployermanagement/config"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logger"
	"github.com/gardener/landscaper/pkg/api"
)

// Options describes the options to configure the Landscaper controller.
type Options struct {
	Log                      logr.Logger
	ConfigPath               string
	landscaperKubeconfigPath string

	Config   *config.LandscaperConfiguration
	Deployer deployerconfig.Options
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.ConfigPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.landscaperKubeconfigPath, "landscaper-kubeconfig", "", "Specify the path to the landscaper kubeconfig cluster")
	o.Deployer.AddFlags(fs)
	logger.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all Options and flags and initializes the basic functions
func (o *Options) Complete(ctx context.Context) error {
	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.Log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	o.Config, err = o.parseConfigurationFile(ctx)
	if err != nil {
		return err
	}
	if err := o.Deployer.Complete(); err != nil {
		return err
	}

	err = o.validate() // validate Options
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) parseConfigurationFile(ctx context.Context) (*config.LandscaperConfiguration, error) {
	decoder := serializer.NewCodecFactory(api.ConfigScheme).UniversalDecoder()

	configv1alpha1 := &v1alpha1.LandscaperConfiguration{}

	if len(o.ConfigPath) != 0 {
		data, err := ioutil.ReadFile(o.ConfigPath)
		if err != nil {
			return nil, err
		}

		if _, _, err := decoder.Decode(data, nil, configv1alpha1); err != nil {
			return nil, err
		}
	}

	api.ConfigScheme.Default(configv1alpha1)

	config := &config.LandscaperConfiguration{}
	err := api.ConfigScheme.Convert(configv1alpha1, config, ctx)
	if err != nil {
		return nil, err
	}
	api.ConfigScheme.Default(config)

	return config, nil
}

// validates the Options
func (o *Options) validate() error {
	return nil
}

package main

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster-sdk/plugin"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/klog/v2"
	"kubeops.dev/vcluster-plugin/api"
	"kubeops.dev/vcluster-plugin/hooks"
	"kubeops.dev/vcluster-plugin/syncers"
)

func main() {
	ctx := plugin.MustInit()
	cfg, err := Init(ctx)
	if err != nil {
		klog.Fatalf("validate config: %v", err)
	}
	plugin.MustRegister(syncers.NewCAProviderClassSyncer(ctx))
	plugin.MustRegister(hooks.NewPodHook(cfg))
	// plugin.MustRegister(hooks.NewSecretHook())
	plugin.MustStart()
}

func Init(ctx *synccontext.RegisterContext) (*api.PluginConfig, error) {
	// set suffix
	// https://github.com/loft-sh/vcluster/blob/v0.19.7/cmd/vcluster/cmd/start.go#L61-L68
	translate.VClusterName = ctx.Config.Name
	if translate.VClusterName == "" {
		translate.VClusterName = "vcluster"
	}

	// https://github.com/loft-sh/vcluster-sdk/blob/main/e2e/test_plugin/main.go#L43
	var cfg api.PluginConfig
	klog.Info(os.Getenv("PLUGIN_CONFIG"))
	err := plugin.UnmarshalConfig(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

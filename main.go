package main

import (
	"github.com/loft-sh/vcluster-sdk/plugin"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"kubeops.dev/vcluster-plugin/hooks"
	"kubeops.dev/vcluster-plugin/syncers"
)

func main() {
	ctx := plugin.MustInit()
	Init(ctx)
	plugin.MustRegister(syncers.NewCAProviderClassSyncer(ctx))
	plugin.MustRegister(hooks.NewPodHook())
	// plugin.MustRegister(hooks.NewSecretHook())
	plugin.MustStart()
}

func Init(ctx *synccontext.RegisterContext) {
	// set suffix
	// https://github.com/loft-sh/vcluster/blob/v0.19.7/cmd/vcluster/cmd/start.go#L61-L68
	translate.VClusterName = ctx.Options.Name
	if translate.VClusterName == "" {
		translate.VClusterName = ctx.Options.DeprecatedSuffix
	}
	if translate.VClusterName == "" {
		translate.VClusterName = "vcluster"
	}
}

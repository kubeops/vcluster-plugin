package hooks

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster-sdk/plugin"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	synctypes "github.com/loft-sh/vcluster/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	api "kubeops.dev/csi-driver-cacerts/apis/cacerts/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
WARNING!!!
SecretHook does not work unless CAProviderClass CR is created before the secret it references.
It seems that there is no way to ModifyController from hooks.
As a result, this hook is not used.

xref: https://github.com/loft-sh/vcluster/blob/v0.19/pkg/controllers/resources/secrets/syncer.go#L91C24-L100
*/

func NewSecretHook() plugin.ClientHook {
	return &secretHook{}
}

// Purpose of this hook is to sync secrets from the virtual cluster to the host cluster
// that are used in a CAProviderClass, without directly setting the annotation on the secret.
type secretHook struct {
	rtx *synccontext.RegisterContext
}

var _ synctypes.Initializer = &secretHook{}

func (s *secretHook) Init(ctx *synccontext.RegisterContext) error {
	s.rtx = ctx
	return nil
}

func (s *secretHook) Name() string {
	return "secret-hook"
}

func (s *secretHook) Resource() client.Object {
	return &corev1.Secret{}
}

var _ plugin.MutateGetVirtual = &secretHook{}

func (s *secretHook) MutateGetVirtual(ctx context.Context, obj client.Object) (client.Object, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("object %s/%s is not a secret", obj.GetNamespace(), obj.GetName())
	}
	if used, err := s.isUsedByCAProviderClass(ctx, secret); err != nil {
		return nil, err
	} else if !used {
		return secret, nil
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	// Force sync the secret to the host cluster
	secret.Annotations["vcluster.loft.sh/force-sync"] = "true"
	return secret, nil
}

func (s *secretHook) isUsedByCAProviderClass(ctx context.Context, secret *corev1.Secret) (bool, error) {
	var cpcList api.CAProviderClassList
	if err := s.rtx.VirtualManager.GetClient().List(ctx, &cpcList); err != nil {
		return false, err
	}
	for _, cpc := range cpcList.Items {
		for _, ref := range cpc.Spec.Refs {
			if ptr.Deref(ref.APIGroup, "") == "" && ref.Kind == "Secret" {
				ns := ref.Namespace
				if ns == "" {
					ns = cpc.Namespace
				}
				if ns == secret.Namespace && ref.Name == secret.Name {
					return true, nil
				}
			}
		}

	}
	return false, nil
}

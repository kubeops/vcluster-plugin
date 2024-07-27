package hooks

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"kubeops.dev/csi-driver-cacerts/apis/cacerts"
	"strings"
	"unicode"

	"github.com/loft-sh/vcluster-sdk/plugin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPodHook() plugin.ClientHook {
	return &podHook{}
}

type podHook struct{}

func (p *podHook) Name() string {
	return "pod-hook"
}

func (p *podHook) Resource() client.Object {
	return &corev1.Pod{}
}

var _ plugin.MutateCreatePhysical = &podHook{}

func (p *podHook) MutateCreatePhysical(ctx context.Context, obj client.Object) (client.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("object %v is not a pod", obj)
	}

	if err := p.mutateCACertVolumes(pod); err != nil {
		return nil, err
	}
	return pod, nil
}

var _ plugin.MutateUpdatePhysical = &podHook{}

func (p *podHook) MutateUpdatePhysical(ctx context.Context, obj client.Object) (client.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("object %v is not a pod", obj)
	}

	if err := p.mutateCACertVolumes(pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func (p *podHook) mutateCACertVolumes(pod *corev1.Pod) error {
	for i, vol := range pod.Spec.Volumes {
		if vol.CSI != nil && vol.CSI.Driver == cacerts.GroupName {
			caProviderClasses := vol.CSI.VolumeAttributes["caProviderClasses"]
			providerKeys := strings.FieldsFunc(caProviderClasses, func(r rune) bool {
				return r == ',' || r == ';' || unicode.IsSpace(r)
			})
			providerNames := sets.New[string]()
			for _, key := range providerKeys {
				if vNamespace, vName, err := cache.SplitMetaNamespaceKey(key); err != nil {
					return errors.Wrapf(err, "invalid provider class in pod %s/%s volume %s", pod.Namespace, pod.Name, vol.Name)
				} else {
					// WARNING: This will not work with multi-namespace translate mode
					if vNamespace == "" {
						parts := strings.SplitN(pod.Name, "-x-", 3)
						if len(parts) != 3 {
							return fmt.Errorf("can't parse virtual namespace for pod %s/%s", pod.Namespace, pod.Name)
						}
						vNamespace = parts[1]
					}

					pName := translate.Default.PhysicalName(vName, vNamespace)
					providerNames.Insert(pName)
				}
			}
			vol.CSI.VolumeAttributes["caProviderClasses"] = strings.Join(sets.List(providerNames), ",")

			pod.Spec.Volumes[i] = vol
		}
	}
	return nil
}

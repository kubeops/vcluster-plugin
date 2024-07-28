package syncers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/scheme"
	synctypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"k8s.io/apimachinery/pkg/api/equality"
	api "kubeops.dev/csi-driver-cacerts/apis/cacerts/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Make sure our scheme is registered
	_ = api.AddToScheme(scheme.Scheme)
}

func NewCAProviderClassSyncer(ctx *synccontext.RegisterContext) synctypes.Base {
	return &cpcSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, api.ResourceCAProviderClass, &api.CAProviderClass{}),
	}
}

type cpcSyncer struct {
	translator.NamespacedTranslator
}

var _ synctypes.Initializer = &cpcSyncer{}

func (s *cpcSyncer) Init(ctx *synccontext.RegisterContext) error {
	out, err := os.ReadFile("manifests/crds.yaml")
	if err != nil {
		return err
	}

	gvk := api.SchemeGroupVersion.WithKind(api.ResourceKindCAProviderClass)
	err = util.EnsureCRD(ctx.Context, ctx.PhysicalManager.GetConfig(), out, gvk)
	if err != nil {
		return err
	}

	_, _, err = translate.EnsureCRDFromPhysicalCluster(ctx.Context, ctx.PhysicalManager.GetConfig(), ctx.VirtualManager.GetConfig(), api.SchemeGroupVersion.WithKind(api.ResourceKindCAProviderClass))
	return err
}

var _ synctypes.Syncer = &cpcSyncer{}

func (s *cpcSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx.Context, vObj.(*api.CAProviderClass)))
}

func (s *cpcSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	return s.SyncToHostUpdate(ctx, vObj, s.translateUpdate(ctx.Context, pObj.(*api.CAProviderClass), vObj.(*api.CAProviderClass)))
}

func (s *cpcSyncer) translate(ctx context.Context, vObj *api.CAProviderClass) *api.CAProviderClass {
	newObj := s.TranslateMetadata(ctx, vObj).(*api.CAProviderClass)

	newObj.Spec = s.translateSpec(vObj).Spec
	return newObj
}

func (s *cpcSyncer) translateSpec(vObj *api.CAProviderClass) *api.CAProviderClass {
	var pObj api.CAProviderClass

	pObj.Spec.Refs = make([]api.TypedObjectReference, len(vObj.Spec.Refs))
	for i, ref := range vObj.Spec.Refs {
		switch ref.Kind {
		case "ClusterIssuer":
			ref.Name = translate.Default.PhysicalNameClusterScoped(ref.Name)
		case "Issuer", "Secret":
			vNamespace := ref.Namespace
			if vNamespace == "" {
				vNamespace = vObj.GetNamespace()
			}
			ref.Name = translate.Default.PhysicalName(ref.Name, vNamespace)
			ref.Namespace = translate.Default.PhysicalNamespace(vNamespace)
		}
		pObj.Spec.Refs[i] = ref
	}

	return &pObj
}

func (s *cpcSyncer) translateUpdate(ctx context.Context, pObj, vObj *api.CAProviderClass) *api.CAProviderClass {
	var updated *api.CAProviderClass

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	// check spec
	updatedSpecObj := s.translateSpec(vObj)
	if !equality.Semantic.DeepEqual(updatedSpecObj.Spec, pObj.Spec) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec = updatedSpecObj.Spec
	}

	return updated
}

func printDiff(original, updated client.Object) error {
	if updated == nil {
		return nil
	}
	originalBytes, err := json.Marshal(original)
	if err != nil {
		return err
	}

	updatedBytes, err := json.Marshal(updated)
	if err != nil {
		return err
	}

	differ := diff.New()
	d, err := differ.Compare(originalBytes, updatedBytes)
	if err != nil {
		return err
	}

	if d.Modified() {
		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       true,
		}

		f := formatter.NewAsciiFormatter(original, config)
		result, err := f.Format(d)
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	}

	return nil
}

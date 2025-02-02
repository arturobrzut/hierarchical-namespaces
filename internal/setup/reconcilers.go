package setup

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"sigs.k8s.io/hierarchical-namespaces/internal/anchor"
	"sigs.k8s.io/hierarchical-namespaces/internal/crd"
	"sigs.k8s.io/hierarchical-namespaces/internal/forest"
	"sigs.k8s.io/hierarchical-namespaces/internal/hierarchyconfig"
	"sigs.k8s.io/hierarchical-namespaces/internal/hncconfig"
)

// CreateReconcilers creates all reconcilers.
//
// This function is called both from main.go as well as from the integ tests.
func CreateReconcilers(mgr ctrl.Manager, f *forest.Forest, maxReconciles int, useFakeClient bool) error {
	crd.Setup(mgr, useFakeClient)

	hcChan := make(chan event.GenericEvent)
	anchorChan := make(chan event.GenericEvent)

	// Create Anchor reconciler.
	ar := &anchor.Reconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("anchor").WithName("reconcile"),
		Forest:   f,
		Affected: anchorChan,
	}
	if err := ar.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("cannot create anchor reconciler: %s", err.Error())
	}

	// Create the HNC Config reconciler.
	hnccfgr := &hncconfig.Reconciler{
		Client:                 mgr.GetClient(),
		Log:                    ctrl.Log.WithName("hncconfig").WithName("reconcile"),
		Manager:                mgr,
		Forest:                 f,
		Trigger:                make(chan event.GenericEvent),
		HierarchyConfigUpdates: hcChan,
	}
	if err := hnccfgr.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("cannot create Config reconciler: %s", err.Error())
	}

	// Create the HC reconciler with a pointer to the Anchor reconciler.
	hcr := &hierarchyconfig.Reconciler{
		Client:              mgr.GetClient(),
		Log:                 ctrl.Log.WithName("hierarchyconfig").WithName("reconcile"),
		Forest:              f,
		AnchorReconciler:    ar,
		HNCConfigReconciler: hnccfgr,
		Affected:            hcChan,
	}
	if err := hcr.SetupWithManager(mgr, maxReconciles); err != nil {
		return fmt.Errorf("cannot create Hierarchy reconciler: %s", err.Error())
	}

	return nil
}

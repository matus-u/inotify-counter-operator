package main

import (
	"context"
	"inotify-counter-operator/watcher"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const GroupName = "count-watcher-operator.com"
const GroupVersion = "v1"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes) // Initializes the SchemeBuilder with the addKnownTypes function
	AddToScheme   = SchemeBuilder.AddToScheme               // Provides a shorthand for adding types to the scheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	// Register the custom resources Application and ApplicationList with the scheme
	scheme.AddKnownTypes(SchemeGroupVersion,
		&FileCountWatcherList{},
		&FileCountWatcher{},
	)

	// Add the group version to the scheme for metav1 objects
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil // Return nil to indicate success
}

type FileCountWatcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []FileCountWatcher `json:"items"`
}

type FileCountWatcherSpec struct {
	Dir string `json:"dir"`
}

type FileCountWatcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FileCountWatcherSpec `json:"spec,omitempty"`
}

func (in *FileCountWatcher) DeepCopyInto(out *FileCountWatcher) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = FileCountWatcherSpec{
		Dir: in.Spec.Dir,
	}
}

func (in *FileCountWatcher) DeepCopyObject() runtime.Object {
	out := FileCountWatcher{}
	in.DeepCopyInto(&out)
	return &out
}

func (in *FileCountWatcherList) DeepCopyObject() runtime.Object {
	out := FileCountWatcherList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]FileCountWatcher, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

type FileWatchCounterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *FileWatchCounterReconciler) Reconcile(ct context.Context, req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	// Get the FileWatchCounterReconciler custom resource
	var cw FileCountWatcher
	err := r.Get(ctx, req.NamespacedName, &cw)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Println("unable to fetch FileWatchCounterReconciler")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil

	}

	if err := r.checkFinalizer(ctx, &cw); err != nil {
		return ctrl.Result{}, err
	}

	log.Println(err, "after finalizer setting")
	// kind of dummy state handling, but for learning reasons ok
	// for real case we would need to store state in some other Status in CountWatcher handle it
	if !cw.DeletionTimestamp.IsZero() {
		log.Println("stop watcher")
		watcher.StopWatcher()
	} else {
		log.Println("start watcher", cw.Spec.Dir)
		watcher.StartWatcher(cw.Spec.Dir)
	}

	return reconcile.Result{}, nil
}
func (r *FileWatchCounterReconciler) checkFinalizer(ctx context.Context, cw *FileCountWatcher) error {
	name := "fileWatchCounterFinalizer"
	if cw.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(cw, name) {
			controllerutil.AddFinalizer(cw, name)
			return r.Update(ctx, cw)
		}
	} else {
		if controllerutil.ContainsFinalizer(cw, name) {
			controllerutil.RemoveFinalizer(cw, name)
			return r.Update(ctx, cw)
		}
	}
	return nil
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	AddToScheme(scheme)
}

func main() {
	ctrllog.SetLogger(zap.New(zap.UseDevMode(true)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Println(err, "start manager error")
		os.Exit(1)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&FileCountWatcher{}).
		Complete(&FileWatchCounterReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})

	if err != nil {
		log.Println(err, "create controller error")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Println("running manager error")
		os.Exit(1)
	}
}

//func test_main() {
//
//	watcher.StartWatcher("./")
//
//	go func() {
//		time.Sleep(time.Second * 10)
//		watcher.StopWatcher()
//	}()
//
//	time.Sleep(time.Second * 12)
//
//}

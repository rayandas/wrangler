package generators

import (
	"fmt"
	"io"
	"strings"

	args2 "github.com/rancher/wrangler/pkg/controller-gen/args"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

func TypeGo(gv schema.GroupVersion, name *types.Name, args *args.GeneratorArgs, customArgs *args2.CustomArgs) generator.Generator {
	return &typeGo{
		name:       name,
		gv:         gv,
		args:       args,
		customArgs: customArgs,
		DefaultGen: generator.DefaultGen{
			OptionalName: strings.ToLower(name.Name),
		},
	}
}

type typeGo struct {
	generator.DefaultGen

	name       *types.Name
	gv         schema.GroupVersion
	args       *args.GeneratorArgs
	customArgs *args2.CustomArgs
}

func (f *typeGo) Imports(context *generator.Context) []string {
	packages := append(Imports,
		fmt.Sprintf("%s \"%s\"", f.gv.Version, f.name.Package))

	return packages
}

func (f *typeGo) Init(c *generator.Context, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	if err := f.DefaultGen.Init(c, w); err != nil {
		return err
	}

	t := c.Universe.Type(*f.name)
	m := map[string]interface{}{
		"type":       f.name.Name,
		"lowerName":  namer.IL(f.name.Name),
		"plural":     plural.Name(t),
		"version":    f.gv.Version,
		"namespaced": namespaced(t),
		"hasStatus":  hasStatus(t),
		"statusType": statusType(t),
	}

	sw.Do(typeBody, m)
	return sw.Error()
}

func statusType(t *types.Type) string {
	for _, m := range t.Members {
		if m.Name == "Status" {
			return m.Type.Name.Name
		}
	}
	return ""
}

func hasStatus(t *types.Type) bool {
	for _, m := range t.Members {
		if m.Name == "Status" && m.Type.Name.Package == t.Name.Package {
			return true
		}
	}
	return false
}

var typeBody = `
// {{.type}}Controller interface for managing {{.type}} resources.
type {{.type}}Controller interface {
    generic.ControllerMeta
	{{.type}}Client

	// OnChange runs the given handler when the controller detects a resource was changed.
	OnChange(ctx context.Context, name string, sync {{.type}}Handler)

	// OnRemove runs the given handler when the controller detects a resource was changed.
	OnRemove(ctx context.Context, name string, sync {{.type}}Handler)

	// Enqueue adds the resource with the given name to the worker queue of the controller.
	Enqueue({{ if .namespaced}}namespace, {{end}}name string)

	// EnqueueAfter runs Enqueue after the provided duration.
	EnqueueAfter({{ if .namespaced}}namespace, {{end}}name string, duration time.Duration)

	// Cache returns a cache for the resource type T.
	Cache() {{.type}}Cache
}

// {{.type}}Client interface for managing {{.type}} resources in Kubernetes.
type {{.type}}Client interface {
	// Create creates a new object and return the newly created Object or an error.
	Create(*{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error)

	// Update updates the object and return the newly updated Object or an error.
	Update(*{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error)
{{ if .hasStatus -}}

	// UpdateStatus updates the Status field of a the object and return the newly updated Object or an error.
	// Will always return an error if the object does not have a status field.
	UpdateStatus(*{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error)
{{- end }}

	// Delete deletes the Object in the given name.
	Delete({{ if .namespaced}}namespace, {{end}}name string, options *metav1.DeleteOptions) error

	// Get will attempt to retrieve the resource with the specified name.
	Get({{ if .namespaced}}namespace, {{end}}name string, options metav1.GetOptions) (*{{.version}}.{{.type}}, error)
	
	// List will attempt to find multiple resources.
	List({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (*{{.version}}.{{.type}}List, error)
	
	// Watch will start watching resources.
	Watch({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (watch.Interface, error)
	
	// Patch will patch the resource with the matching name.
	Patch({{ if .namespaced}}namespace, {{end}}name string, pt types.PatchType, data []byte, subresources ...string) (result *{{.version}}.{{.type}}, err error)
}

// {{.type}}Cache interface for retrieving {{.type}} resources in memory.
type {{.type}}Cache interface {
	// Get returns the resources with the specified name from the cache.
	Get({{ if .namespaced}}namespace, {{end}}name string) (*{{.version}}.{{.type}}, error)
	
	// List will attempt to find resources from the Cache.
	List({{ if .namespaced}}namespace string, {{end}}selector labels.Selector) ([]*{{.version}}.{{.type}}, error)

	// AddIndexer adds  a new Indexer to the cache with the provided name.
	// If you call this after you already have data in the store, the results are undefined.
	AddIndexer(indexName string, indexer {{.type}}Indexer)
	
	// GetByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value.
	GetByIndex(indexName, key string) ([]*{{.version}}.{{.type}}, error)
}
// {{.type}}Handler is function for performing any potential modifications to a {{.type}} resource.
type {{.type}}Handler func(string, *{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error)

// {{.type}}Indexer computes a set of indexed values for the provided object.
type {{.type}}Indexer func(obj *{{.version}}.{{.type}}) ([]string, error)

// {{.type}}GenericController wraps wrangler/pkg/generic.{{ if not .namespaced}}NonNamespaced{{end}}Controller so that the function definitions adhere to {{.type}}Controller interface.
type {{.type}}GenericController struct {
	generic.{{ if not .namespaced}}NonNamespaced{{end}}ControllerInterface[*{{.version}}.{{.type}}, *{{.version}}.{{.type}}List]
}

// OnChange runs the given resource handler when the controller detects a resource was changed.
func (c *{{.type}}GenericController) OnChange(ctx context.Context, name string, sync {{.type}}Handler) {
	c.{{ if not .namespaced}}NonNamespaced{{end}}ControllerInterface.OnChange(ctx, name, generic.ObjectHandler[*{{.version}}.{{.type}}](sync))
}

// OnRemove runs the given object handler when the controller detects a resource was changed.
func (c *{{.type}}GenericController) OnRemove(ctx context.Context, name string, sync {{.type}}Handler) {
	c.{{ if not .namespaced}}NonNamespaced{{end}}ControllerInterface.OnRemove(ctx, name, generic.ObjectHandler[*{{.version}}.{{.type}}](sync))
}

// Cache returns a cache of resources in memory.
func (c *{{.type}}GenericController) Cache() {{.type}}Cache {
	return &{{.type}}GenericCache{
		c.{{ if not .namespaced}}NonNamespaced{{end}}ControllerInterface.Cache(),
	}
}

// {{.type}}GenericCache wraps wrangler/pkg/generic.{{ if not .namespaced}}NonNamespaced{{end}}Cache so the function definitions adhere to {{.type}}Cache interface.
type {{.type}}GenericCache struct {
	generic.{{ if not .namespaced}}NonNamespaced{{end}}CacheInterface[*{{.version}}.{{.type}}]
}

// AddIndexer adds  a new Indexer to the cache with the provided name.
// If you call this after you already have data in the store, the results are undefined.
func (c {{.type}}GenericCache) AddIndexer(indexName string, indexer {{.type}}Indexer) {
	c.{{ if not .namespaced}}NonNamespaced{{end}}CacheInterface.AddIndexer(indexName, generic.Indexer[*{{.version}}.{{.type}}](indexer))
}

{{ if .hasStatus -}}
type {{.type}}StatusHandler func(obj *{{.version}}.{{.type}}, status {{.version}}.{{.statusType}}) ({{.version}}.{{.statusType}}, error)

type {{.type}}GeneratingHandler func(obj *{{.version}}.{{.type}}, status {{.version}}.{{.statusType}}) ([]runtime.Object, {{.version}}.{{.statusType}}, error)

func From{{.type}}HandlerToHandler(sync {{.type}}Handler) generic.Handler {
	return generic.FromObjectHandlerToHandler(generic.ObjectHandler[*{{.version}}.{{.type}},](sync))
}

func Register{{.type}}StatusHandler(ctx context.Context, controller {{.type}}Controller, condition condition.Cond, name string, handler {{.type}}StatusHandler) {
	statusHandler := &{{.lowerName}}StatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, From{{.type}}HandlerToHandler(statusHandler.sync))
}

func Register{{.type}}GeneratingHandler(ctx context.Context, controller {{.type}}Controller, apply apply.Apply,
	condition condition.Cond, name string, handler {{.type}}GeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &{{.lowerName}}GeneratingHandler{
		{{.type}}GeneratingHandler: handler,
		apply:                            apply,
		name:                             name,
		gvk:                              controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	Register{{.type}}StatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type {{.lowerName}}StatusHandler struct {
	client    {{.type}}Client
	condition condition.Cond
	handler   {{.type}}StatusHandler
}

func (a *{{.lowerName}}StatusHandler) sync(key string, obj *{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type {{.lowerName}}GeneratingHandler struct {
	{{.type}}GeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *{{.lowerName}}GeneratingHandler) Remove(key string, obj *{{.version}}.{{.type}}) (*{{.version}}.{{.type}}, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &{{.version}}.{{.type}}{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *{{.lowerName}}GeneratingHandler) Handle(obj *{{.version}}.{{.type}}, status {{.version}}.{{.statusType}}) ({{.version}}.{{.statusType}}, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.{{.type}}GeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
{{- end }}
`

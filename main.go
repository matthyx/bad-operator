package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"
	"sync"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	// get all resources from the spdx.softwarecomposition.kubescape.io/v1beta1 group
	disco := discovery.NewDiscoveryClientForConfigOrDie(config)
	gv := schema.GroupVersion{
		Group:   "spdx.softwarecomposition.kubescape.io",
		Version: "v1beta1",
	}
	resources, err := disco.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		panic(err)
	}

	var ks []schema.GroupVersionResource
	for _, r := range resources.APIResources {
		ks = append(ks, schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: r.Name,
		})
	}

	// list all resources, constantly
	client := dynamic.NewForConfigOrDie(config)
	var wg sync.WaitGroup
	for _, res := range ks {
		wg.Add(1)
		go func(res schema.GroupVersionResource) {
			for {
				list, err := client.Resource(res).Namespace("").List(context.Background(), v1.ListOptions{})
				if err != nil {
					panic(err)
				}
				for _, item := range list.Items {
					// get full item (list only returns metadata)
					fullItem, err := client.Resource(res).Namespace(item.GetNamespace()).Get(context.Background(), item.GetName(), v1.GetOptions{})
					if err != nil {
						panic(err)
					}
					log.Println(res.Resource, fullItem.GetName())
				}
			}
		}(res)
	}
	wg.Wait()
}

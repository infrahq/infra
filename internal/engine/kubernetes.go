package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/google/shlex"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Kubernetes struct {
	mu     sync.Mutex
	config *rest.Config
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	config, err := rest.InClusterConfig()
	if err != nil {
		return k, err
	}

	k.config = config

	return k, err
}

func (k *Kubernetes) UpdatePermissions(rbs []RoleBinding) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	subjects := make(map[string][]rbacv1.Subject)

	for _, rb := range rbs {
		subjects[rb.Role] = append(subjects[rb.Role], rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "User",
			Name:     rb.User,
		})
	}

	crbs := []*rbacv1.ClusterRoleBinding{}
	for role, subjs := range subjects {
		crbs = append(crbs, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "infra-" + role,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "infra",
				},
			},
			Subjects: subjs,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     role,
			},
		})
	}

	// TODO (jmorganca): find and delete empty rolebindings
	// Create empty crbs for roles with no users
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return err
	}

	existingCrbs, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"})
	if err != nil {
		return err
	}

	// Create or update CRBs for users
	for _, crb := range crbs {
		_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Delete any CRBs managed by infra that aren't in the config
	var toDelete []rbacv1.ClusterRoleBinding
	for _, e := range existingCrbs.Items {
		var found bool
		for _, crb := range crbs {
			if crb.Name == e.Name {
				found = true
			}
		}

		if !found {
			toDelete = append(toDelete, e)
		}
	}

	for _, td := range toDelete {
		err := clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), td.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) CA() ([]byte, error) {
	contents, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func (k *Kubernetes) ExecCat(pod string, namespace string, file string) (string, error) {
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}

	cmd := []string{
		"/bin/cat",
		file,
	}
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod).Namespace(namespace).SubResource("exec")
	req.VersionedParams(
		&v1.PodExecOptions{
			Command: cmd,
			Stdout:  true,
		},
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: io.Writer(&buf),
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Endpoint gets the cluster endpoint from within the pod
func (k *Kubernetes) Endpoint() (string, error) {
	// Create empty crbs for roles with no users
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}

	var endpoint string

	// Get the full command line for kube-proxy pods
	// if --master is specified, use that
	// if --kubeconfig is specified, exec + cat to read that
	// if --config is specified, exec + cat the file the kubeconfig location, and read the kubeconfig

	pods1, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "k8s-app=kube-proxy",
	})
	if err != nil {
		return "", err
	}

	pods2, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=kube-proxy",
	})
	if err != nil {
		return "", err
	}

	pods := append(pods1.Items, pods2.Items...)

	if len(pods) == 0 {
		return "", errors.New("no kube-proxy pods to inspect")
	}

	pod := pods[0]

	var command []string
	for _, c := range pod.Spec.Containers {
		if c.Name == "kube-proxy" {
			command = c.Command
			break
		}
	}

	var args []string
	for _, c := range command {
		split, err := shlex.Split(c)
		if err != nil {
			continue
		}
		args = append(args, split...)
	}

	var opts struct {
		Master     string `long:"master"`
		Config     string `long:"config"`
		Kubeconfig string `long:"kubeconfig"`
	}

	p := flags.NewParser(&opts, flags.IgnoreUnknown)
	_, err = p.ParseArgs(args)
	if err != nil {
		return "", err
	}

	fmt.Println("Read kube-proxy opts: ", opts)

	switch {
	case opts.Master != "":
		endpoint = opts.Master
	case opts.Config != "":
		contents, err := k.ExecCat(pod.Name, "kube-system", opts.Config)
		if err != nil {
			return "", err
		}
		var raw map[string]interface{}
		err = yaml.Unmarshal([]byte(contents), &raw)
		if err != nil {
			return "", err
		}

		clientConnection, ok := raw["clientConnection"].(map[interface{}]interface{})
		if !ok {
			return "", errors.New("invalid kube-proxy config format")
		}
		kubeconfig, ok := clientConnection["kubeconfig"].(string)
		if !ok {
			return "", errors.New("invalid kube-proxy config format")
		}

		opts.Kubeconfig = kubeconfig
		fallthrough
	case opts.Kubeconfig != "":
		contents, err := k.ExecCat(pod.Name, "kube-system", opts.Kubeconfig)
		if err != nil {
			return "", err
		}

		cfg, err := clientcmd.NewClientConfigFromBytes([]byte(contents))
		if err != nil {
			return "", err
		}

		rc, err := cfg.ClientConfig()
		if err != nil {
			return "", err
		}

		endpoint = rc.Host
	default:
		fmt.Println("Warning, could not find parse kube-proxy opts, args: ", args)
	}

	// Rewrite docker desktop
	if endpoint == "https://vm.docker.internal:6443" {
		endpoint = "https://kubernetes.docker.internal:6443"
	}

	// Rewrite digital ocean
	if strings.HasSuffix(endpoint, ".internal.k8s.ondigitalocean.com") {
		endpoint = strings.Replace(endpoint, ".internal.k8s.ondigitalocean.com", ".k8s.ondigitalocean.com", -1)
	}

	// TODO (jmorganca): minikube

	// Could not get endpoint - must be passed via flag
	return endpoint, nil
}

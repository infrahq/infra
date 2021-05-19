package server

import (
	"context"
	"errors"
	"fmt"
	"sync"

	bolt "go.etcd.io/bbolt"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubernetes struct {
	Config *rest.Config
	mu     sync.Mutex
}

type RoleBinding struct {
	User string
	Role string
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	config, err := rest.InClusterConfig()
	if err == rest.ErrNotInCluster {
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}

		fmt.Println("Using out-of-cluster Kubeconfig")
	} else if err != nil {
		return nil, err
	} else {
		fmt.Println("Using in-cluster Kubeconfig")
	}

	k.Config = config

	return k, err
}

func (k *Kubernetes) UpdatePermissions(db *bolt.DB, cfg *Config) error {
	if db == nil || cfg == nil {
		return errors.New("parameter cannot be nil")
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	var users []User

	err := db.View(func(tx *bolt.Tx) (err error) {
		users, err = ListUsers(tx)
		return err
	})
	if err != nil {
		return err
	}

	rbs := []RoleBinding{}
	for _, user := range users {
		permission := PermissionForEmail(user.Email, cfg)
		if permission != "" {
			rbs = append(rbs, RoleBinding{User: user.Email, Role: permission})
		}
	}

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
			},
			Subjects: subjs,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     role,
			},
		})
	}

	if k.Config != nil {
		clientset, err := kubernetes.NewForConfig(k.Config)
		if err != nil {
			return err
		}

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
	}

	return nil
}

package main

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

type Kubernetes struct {
	Config *rest.Config
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	// TODO(jmorganca): support remote cluster for testing
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	k.Config = config

	return k, err
}

// TODO(jmorganca): protect this from race conditions
func (k *Kubernetes) UpdateRoleBindings(users []string) error {
	subjects := []rbacv1.Subject{}

	for _, u := range users {
		subjects = append(subjects, rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "User",
			Name:     u,
		})
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "infra-view",
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "view",
		},
	}

	if k.Config != nil {
		clientset, err := kubernetes.NewForConfig(k.Config)
		if err != nil {
			fmt.Println(err)
		}

		_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Cluster role binding added")
				}
			} else {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Cluster role binding patched")
		}
	}
	return nil
}

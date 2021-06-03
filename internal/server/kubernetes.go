package server

import (
	"context"
	"errors"
	"sync"

	"gorm.io/gorm"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

type Kubernetes struct {
	Config *rest.Config
	mu     sync.Mutex
	db     *gorm.DB
}

type RoleBinding struct {
	User string
	Role string
}

func NewKubernetes(db *gorm.DB) (*Kubernetes, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	k := &Kubernetes{db: db}

	config, err := rest.InClusterConfig()
	if err != nil {
		return k, err
	}

	k.Config = config

	return k, err
}

func (k *Kubernetes) UpdatePermissions() error {
	if k.Config == nil {
		return errors.New("invalid kubernetes config")
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	var permissions []Permission
	err := k.db.Preload("User").Preload("Role").Find(&permissions).Error
	if err != nil {
		return err
	}

	rbs := []RoleBinding{}
	numUsers := make(map[string]int)
	for _, permission := range permissions {
		rbs = append(rbs, RoleBinding{User: permission.UserEmail, Role: permission.Role.KubernetesRole})
		numUsers[permission.Role.KubernetesRole] += 1
	}

	var roles []Role
	err = k.db.Find(&roles).Error
	if err != nil {
		return err
	}

	emptyRoles := []string{}
	for _, r := range roles {
		if numUsers[r.KubernetesRole] == 0 {
			emptyRoles = append(emptyRoles, r.KubernetesRole)
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

	// Create empty crbs for roles with no users
	for _, e := range emptyRoles {
		crbs = append(crbs, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "infra-" + e,
			},
			Subjects: []rbacv1.Subject{},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     e,
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

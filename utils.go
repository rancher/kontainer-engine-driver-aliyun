package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	errs "github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultNamespace = "default"
	clusterAdmin     = "cluster-admin"
	netesDefault     = "netes-default"
)

func TickerContext(ctx context.Context, duration time.Duration) <-chan time.Time {
	ticker := time.NewTicker(duration)
	go func() {
		<-ctx.Done()
		ticker.Stop()
	}()
	return ticker.C
}

// GenerateServiceAccountToken generate a serviceAccountToken for clusterAdmin given a rest clientset
func GenerateServiceAccountToken(clientset kubernetes.Interface) (string, error) {
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: netesDefault,
		},
	}

	_, err := clientset.CoreV1().ServiceAccounts(defaultNamespace).Create(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
		return "", fmt.Errorf("error creating service account: %v", err)
	}

	adminRole := &v1beta1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterAdmin,
		},
		Rules: []v1beta1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				NonResourceURLs: []string{"*"},
				Verbs:           []string{"*"},
			},
		},
	}
	clusterAdminRole, err := clientset.RbacV1beta1().ClusterRoles().Get(clusterAdmin, metav1.GetOptions{})
	if err != nil {
		clusterAdminRole, err = clientset.RbacV1beta1().ClusterRoles().Create(adminRole)
		if err != nil {
			return "", fmt.Errorf("error creating admin role: %v", err)
		}
	}

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "netes-default-clusterRoleBinding",
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: "default",
				APIGroup:  v1.GroupName,
			},
		},
		RoleRef: v1beta1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterAdminRole.Name,
			APIGroup: v1beta1.GroupName,
		},
	}
	if _, err = clientset.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return "", fmt.Errorf("error creating role bindings: %v", err)
	}

	start := time.Millisecond * 250
	for i := 0; i < 5; i++ {
		time.Sleep(start)
		if serviceAccount, err = clientset.CoreV1().ServiceAccounts(defaultNamespace).Get(serviceAccount.Name, metav1.GetOptions{}); err != nil {
			return "", fmt.Errorf("error getting service account: %v", err)
		}

		if len(serviceAccount.Secrets) > 0 {
			secret := serviceAccount.Secrets[0]
			secretObj, err := clientset.CoreV1().Secrets(defaultNamespace).Get(secret.Name, metav1.GetOptions{})
			if err != nil {
				return "", fmt.Errorf("error getting secret: %v", err)
			}
			if token, ok := secretObj.Data["token"]; ok {
				return string(token), nil
			}
		}
		start = start * 2
	}

	return "", errs.New("failed to fetch serviceAccountToken")
}

// NewCsAPIRequest constructs general aliyun sdk request
func NewCsAPIRequest(apiName, method string) *requests.CommonRequest {
	requests.NewCommonRequest()
	request := requests.NewCommonRequest()
	request.Version = "2015-12-15"
	request.ApiName = apiName
	request.Method = method
	request.SetDomain("cs.aliyuncs.com")
	request.SetScheme(requests.HTTPS)
	request.SetContentType("application/json;charset=utf-8")
	return request
}

func ProcessRequest(svc *cs.Client, request *requests.CommonRequest, obj interface{}) error {
	response, err := svc.ProcessCommonRequest(request)
	if err != nil && !strings.Contains(err.Error(), "JsonUnmarshalError") {
		return err
	}
	if obj != nil {
		if err := json.Unmarshal(response.GetHttpContentBytes(), obj); err != nil {
			return err
		}
	}
	return nil
}

// compareVersion compares two version strings
// returns -1 if v1 < v2, 1 if v1 > v2, 0 otherwise.
func compareVersion(v1, v2 string) int {
	var (
		currTab  = strings.Split(v1, ".")
		otherTab = strings.Split(v2, ".")
	)

	max := len(currTab)
	if len(otherTab) > max {
		max = len(otherTab)
	}
	for i := 0; i < max; i++ {
		var currInt, otherInt int

		if len(currTab) > i {
			currInt, _ = strconv.Atoi(getHeadDigits(currTab[i]))
		}
		if len(otherTab) > i {
			otherInt, _ = strconv.Atoi(getHeadDigits(otherTab[i]))
		}
		if currInt > otherInt {
			return 1
		}
		if otherInt > currInt {
			return -1
		}
	}
	return 0
}

// getHeadDigits returns the digit part at the beginning of a sting
func getHeadDigits(raw string) string {
	for idx, r := range raw {
		if !unicode.IsDigit(r) {
			return raw[:idx]
		}
	}
	return raw
}

// versionGreaterThan checks if a version is greater than another
func versionGreaterThan(v, other string) bool {
	return compareVersion(v, other) == 1
}

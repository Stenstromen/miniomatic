package k8sclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stenstromen/miniomatic/db"
	"github.com/stenstromen/miniomatic/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

const namespace = "miniomatic"

func boolPtr(b bool) *bool { return &b }

func getK8sClient() (*kubernetes.Clientset, error) {
	configFile := os.Getenv("KUBECONFIG_FILE")
	if configFile == "" {
		configFile = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", configFile)
	if err != nil {
		log.Fatalf("failed to get Kubernetes config: %v", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create Kubernetes client: %v", err)
	}
	return client, nil
}

func ensureNamespace(client *kubernetes.Clientset) error {
	_, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("failed to create namespace %s: %v", namespace, err)
		}
		log.Printf("Created namespace %s", namespace)
	} else if err != nil {
		log.Fatalf("failed to get namespace %s: %v", namespace, err)
	}
	return nil
}

func createMinioSecret(client *kubernetes.Clientset, randnum, namespace, rootPassword string) error {
	// Define the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: randnum + "-minio-secrets",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"rootPassword": []byte(rootPassword),
		},
	}

	// Create the secret in the Kubernetes cluster
	_, err := client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("failed to create secret: %v", err)
	}
	return nil
}

func ResizeMinioPVC(randnum, storage string) error {
	if err := db.UpdateStatus(randnum, "resizing"); err != nil {
		return fmt.Errorf("failed to update status to resizing: %v", err)
	}

	client, err := getK8sClient()
	if err != nil {
		return err
	}

	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), randnum+"-minio-pvc", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to get PVC: %v", err)
	}

	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(storage)

	_, err = client.CoreV1().PersistentVolumeClaims(namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
	if err != nil {
		log.Fatalf("failed to update PVC: %v", err)
	}

	if err := db.UpdateStatus(randnum, "ready"); err != nil {
		return fmt.Errorf("failed to update status to ready: %v", err)
	}

	return nil
}

func CreateMinioResources(creds model.Credentials, clusterIssuer, storageClassName, storage string) error {
	randnum, rootUser, rootPassword := creds.RandNum, creds.RootUser, creds.RootPassword
	wildcard_domain := randnum + "." + os.Getenv("WILDCARD_DOMAIN")

	// Get the Kubernetes configuration.
	client, err := getK8sClient()
	if err != nil {
		return err
	}

	// Create the Minio Secret
	if err := createMinioSecret(client, randnum, namespace, rootPassword); err != nil {
		log.Fatalf("Failed to create secret: %v", err)
		return err
	}

	if err := ensureNamespace(client); err != nil {
		return err
	}

	// Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: randnum + "-minio-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": randnum + "minio"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": randnum + "minio"},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: boolPtr(false),
					Containers: []corev1.Container{
						{
							Name:  randnum + "-minio",
							Image: "minio/minio:latest",
							Args:  []string{"server", "/data"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MINIO_ROOT_USER",
									Value: rootUser,
								},
								{
									Name: "MINIO_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: randnum + "-minio-secrets",
											},
											Key: "rootPassword",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9000,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/live",
										Port:   intstr.FromInt(9000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/ready",
										Port:   intstr.FromInt(9000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: randnum + "-minio-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = client.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create deployment: %v", err)
	}

	// Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "s-" + randnum + "-minio-service",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
				},
			},
			Selector: map[string]string{
				"app": randnum + "minio",
			},
		},
	}
	_, err = client.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create service:", err)
	}

	// Ingress
	pathTypePrefix := networkingv1.PathTypePrefix
	host := wildcard_domain
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: randnum + "-minio-ingress",
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer":                     clusterIssuer,
				"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
				"nginx.ingress.kubernetes.io/proxy-buffering":        "off",
				"nginx.ingress.kubernetes.io/ignore-invalid-headers": "off",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To("nginx"),
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "s-" + randnum + "-minio-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 9000,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{host},
					SecretName: host + "-tls",
				},
			},
		},
	}
	_, err = client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create ingress:", err)
	}

	// PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: randnum + "-minio-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storage),
				},
			},
		},
	}
	_, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create PVC:", err)
	}

	return nil
}

func DeleteMinioResources(randnum string) error {
	client, err := getK8sClient()
	if err != nil {
		return err
	}

	// Delete Ingress
	err = client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), randnum+"-minio-ingress", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("failed to delete ingress: %v", err)
	}

	// Delete Service
	err = client.CoreV1().Services(namespace).Delete(context.TODO(), "s-"+randnum+"-minio-service", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("failed to delete service: %v", err)
	}

	// Delete Deployment
	err = client.AppsV1().Deployments(namespace).Delete(context.TODO(), randnum+"-minio-deployment", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("failed to delete deployment: %v", err)
	}

	// Delete Secret
	err = client.CoreV1().Secrets(namespace).Delete(context.TODO(), randnum+"-minio-secrets", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("failed to delete secret: %v", err)
	}

	// Delete TLS Secret
	err = client.CoreV1().Secrets(namespace).Delete(context.TODO(), randnum+"."+os.Getenv("WILDCARD_DOMAIN")+"-tls", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalln("failed to delete TLS secret:", err)
	}

	// Delete PVC
	err = client.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), randnum+"-minio-pvc", metav1.DeleteOptions{})
	if err != nil {
		log.Fatalln("failed to delete PVC:", err)
	}

	return nil
}

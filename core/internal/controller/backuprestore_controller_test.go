/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	// Переименован стандартный пакет errors
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors" // Переименован Kubernetes errors
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1 "github.com/AntonShadrinNN/oiler-backup/api/v1"
)

var _ = Describe("BackupRestore Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		backuprestore := &backupv1.BackupRestore{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind BackupRestore")
			err := k8sClient.Get(ctx, typeNamespacedName, backuprestore)
			if err != nil && k8sErrors.IsNotFound(err) { // Используем k8sErrors
				resource := &backupv1.BackupRestore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: backupv1.BackupRestoreSpec{
						DatabaseURI:    "localhost",
						DatabasePort:   5432,
						DatabaseUser:   "admin",
						DatabasePass:   "password",
						DatabaseName:   "testdb",
						DatabaseType:   "postgres",
						S3Endpoint:     "http://minio-service:9000",
						S3AccessKey:    "access-key",
						S3SecretKey:    "secret-key",
						S3BucketName:   "backups",
						BackupRevision: "0",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance BackupRestore")
			resource := &backupv1.BackupRestore{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &BackupRestoreReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				DatabaseControllers: map[string]string{
					"postgres": "postgres-controller-address",
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Проверяем, что статус ресурса был обновлен
			err = k8sClient.Get(ctx, typeNamespacedName, backuprestore)
			Expect(err).NotTo(HaveOccurred())
			Expect(backuprestore.Status.Status).To(Equal("Success"))
		})

		It("should fail if the database type is not supported", func() {
			By("Creating a resource with an unsupported database type")
			resource := &backupv1.BackupRestore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unsupported-db",
					Namespace: "default",
				},
				Spec: backupv1.BackupRestoreSpec{
					DatabaseURI:    "localhost",
					DatabasePort:   5432,
					DatabaseUser:   "admin",
					DatabasePass:   "password",
					DatabaseName:   "testdb",
					DatabaseType:   "unsupported",
					S3Endpoint:     "http://minio-service:9000",
					S3AccessKey:    "access-key",
					S3SecretKey:    "secret-key",
					S3BucketName:   "backups",
					BackupRevision: "0",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			controllerReconciler := &BackupRestoreReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
				DatabaseControllers: map[string]string{
					"postgres": "postgres-controller-address",
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "unsupported-db",
					Namespace: "default",
				},
			})
			Expect(err).To(HaveOccurred())

			// Проверяем, что статус ресурса не обновлен
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "unsupported-db",
				Namespace: "default",
			}, backuprestore)
			Expect(err).NotTo(HaveOccurred())
			Expect(backuprestore.Status.Status).To(Equal(""))
		})

		It("should handle missing ConfigMap gracefully", func() {
			By("Deleting the ConfigMap to simulate a missing configuration")
			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "database-config",
				Namespace: "default",
			}, configMap)
			if err == nil {
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			controllerReconciler := &BackupRestoreReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})
	})
})

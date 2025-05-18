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

	backupv1 "github.com/oiler-backup/core/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("BackupRequest Controller", func() {
	var (
		ctx        context.Context
		reconciler *BackupRequestReconciler
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = backupv1.AddToScheme(scheme)

		reconciler = &BackupRequestReconciler{
			Client: k8sClient,
			Scheme: scheme,
		}
		appCfg.OperatorNamespace = "oiler-backup-system"
	})

	Context("When reconciling a BackupRequest", func() {
		const resourceName = "test-br"
		nsName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oiler-backup-system",
				},
			}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: ns.Namespace,
				Name:      ns.Name,
			}, &corev1.Namespace{}); apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ns)).To(Succeed())
			}
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "database-config",
					Namespace: "oiler-backup-system",
				},
				Data: map[string]string{
					"postgres": "adadpter.addr",
				},
			}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: cm.Namespace,
				Name:      cm.Name,
			}, &corev1.ConfigMap{}); apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, cm)).To(Succeed())
			}
			br := &backupv1.BackupRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: backupv1.BackupRequestSpec{
					DatabaseType:   "postgres",
					DatabaseURI:    "localhost",
					DatabasePort:   5432,
					DatabaseUser:   "user",
					DatabasePass:   "pass",
					DatabaseName:   "db",
					Schedule:       "0 0 * * *",
					StorageClass:   "standard",
					S3Endpoint:     "s3.example.com",
					S3AccessKey:    "key",
					S3SecretKey:    "secret",
					S3BucketName:   "bucket",
					MaxBackupCount: 7,
				},
			}
			Expect(k8sClient.Create(ctx, br)).To(Succeed())
		})

		AfterEach(func() {
			br := &backupv1.BackupRequest{}
			if err := k8sClient.Get(ctx, nsName, br); err == nil {
				Expect(k8sClient.Delete(ctx, br)).To(Succeed())
			}

		})

		It("should fail if database type is unsupported", func() {
			br := &backupv1.BackupRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unsupported-db",
					Namespace: "default",
				},
				Spec: backupv1.BackupRequestSpec{
					DatabaseType: "unknown",
				},
			}
			Expect(k8sClient.Create(ctx, br)).To(Succeed())

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "unsupported-db",
					Namespace: "default",
				},
			}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database unknown is not supported"))

			err = k8sClient.Get(ctx, req.NamespacedName, br)
			Expect(err).ToNot(HaveOccurred())
			Expect(br.Status.Status).To(Equal(FAILURE))
		})

		It("should handle failed grpc call", func() {
			req := reconcile.Request{NamespacedName: nsName}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())

			br := &backupv1.BackupRequest{}
			Expect(k8sClient.Get(ctx, nsName, br)).To(Succeed())
			Expect(br.Status.Status).To(Equal(FAILURE))
		})
	})
})

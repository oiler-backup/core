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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DatabaseSpec struct {
	URI    string `json:"uri"`
	Port   int    `json:"port"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	DbName string `json:"dbName"`
	DbType string `json:"dbType"`
}

type S3Auth struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}
type S3Spec struct {
	Endpoint   string `json:"endpoint"`
	Auth       S3Auth `json:"auth"`
	BucketName string `json:"bucketName"`
}

// BackupRequestSpec defines the desired state of BackupRequest.
type BackupRequestSpec struct {
	DbSpec DatabaseSpec `json:"dbSpec"`
	S3Spec S3Spec       `json:"s3Spec"`

	Schedule       string `json:"schedule"`
	MaxBackupCount int64  `json:"maxBackupCount"`
}

type CreatedCronJobData struct {
	Name      string `json:"name,required"`      //nolint:staticcheck
	Namespace string `json:"namespace,required"` //nolint:staticcheck
}

// BackupRequestStatus defines the observed state of BackupRequest.
type BackupRequestStatus struct {
	Status         string             `json:"status,omitempty"`
	LastBackupTime *metav1.Time       `json:"lastBackupTime,omitempty"`
	CronJobData    CreatedCronJobData `json:"cronJobData,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=br
// BackupRequest is the Schema for the backuprequests API.
type BackupRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupRequestSpec   `json:"spec,omitempty"`
	Status BackupRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=brs
// BackupRequestList contains a list of BackupRequest.
type BackupRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupRequest{}, &BackupRequestList{})
}

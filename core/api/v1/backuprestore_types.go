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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupRestoreSpec defines the desired state of BackupRestore.
type BackupRestoreSpec struct {
	DatabaseURI  string `json:"dbUri"`
	DatabasePort int    `json:"databasePort"`
	DatabaseUser string `json:"databaseUser"`
	DatabasePass string `json:"databasePass"`
	DatabaseName string `json:"databaseName"`
	DatabaseType string `json:"databaseType"`

	S3Endpoint     string `json:"s3Endpoint"`
	S3AccessKey    string `json:"s3AccessKey"`
	S3SecretKey    string `json:"s3SecretKey"`
	S3BucketName   string `json:"s3BucketName"`
	BackupRevision string `json:"backupRevision"` // переделать на int
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// BackupRestoreStatus defines the observed state of BackupRestore.
type BackupRestoreStatus struct {
	Status          string       `json:"status,omitempty"`
	LastRestoreTime *metav1.Time `json:"lastRestoreTime,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// BackupRestore is the Schema for the backuprestores API.
type BackupRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupRestoreSpec   `json:"spec,omitempty"`
	Status BackupRestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// BackupRestoreList contains a list of BackupRestore.
type BackupRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupRestore{}, &BackupRestoreList{})
}

package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// AccessKeyIDName is the environment variable name for the S3 access key ("username")
	AccessKeyIDName = "AWS_ACCESS_KEY_ID"
	// SecretAccessKeyName is the environment variable name for the S3 secret key ("password")
	SecretAccessKeyName = "AWS_SECRET_ACCESS_KEY"
)

// BucketParameters are the configurable fields of a Bucket.
type BucketParameters struct {
	// +kubebuilder:validation:Required

	// CredentialsSecretRef contains the reference of the Secret where the credentials of the S3 user are stored.
	// The secret must contain the keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
	CredentialsSecretRef corev1.SecretReference `json:"credentialsSecretRef"`

	// +kubebuilder:validation:Required

	// EndpointURL is the URL where to create the bucket.
	// If the scheme is omitted (`http/s`), HTTPS is assumed.
	// Changing the endpoint after initial creation might have no effect.
	EndpointURL string `json:"endpointURL"`

	// BucketName is the name of the bucket to create.
	// Defaults to `metadata.name` if unset.
	// Cannot be changed after bucket is created.
	// Name must be acceptable by the S3 protocol, which follows RFC 1123.
	// Be aware that S3 providers may require a unique name across the platform or region.
	BucketName string `json:"bucketName,omitempty"`

	// +kubebuilder:validation:Required

	// Region is the name of the region where the bucket shall be created.
	// The region must be available in the S3 endpoint.
	// Cannot be changed after bucket is created.
	Region string `json:"region"`
}

// BucketSpec defines the desired state of a Bucket.
type BucketSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BucketParameters `json:"forProvider"`
}

// BucketObservation are the observable fields of a Bucket.
type BucketObservation struct {
	// BucketName is the name of the actual bucket.
	BucketName string `json:"bucketName,omitempty"`
	// Region is the region identifier of the actual bucket.
	Region string `json:"region,omitempty"`
}

// BucketStatus represents the observed state of a Bucket.
type BucketStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BucketObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="External Name",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.forProvider.endpointURL"
// +kubebuilder:printcolumn:name="Bucket Name",type="string",JSONPath=".status.atProvider.bucketName"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.forProvider.region"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,cloudscale}

// Bucket is the API for creating S3 buckets.
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec"`
	Status BucketStatus `json:"status,omitempty"`
}

// GetBucketName returns the spec.forProvider.bucketName if given, otherwise defaults to metadata.name.
func (in *Bucket) GetBucketName() string {
	if in.Spec.ForProvider.BucketName == "" {
		return in.Name
	}
	return in.Spec.ForProvider.BucketName
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

// Bucket type metadata.
var (
	BucketKind             = reflect.TypeOf(Bucket{}).Name()
	BucketGroupKind        = schema.GroupKind{Group: Group, Kind: BucketKind}.String()
	BucketKindAPIVersion   = BucketKind + "." + SchemeGroupVersion.String()
	BucketGroupVersionKind = SchemeGroupVersion.WithKind(BucketKind)
)

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}

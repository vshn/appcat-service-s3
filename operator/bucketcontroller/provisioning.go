package bucketcontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/apis/conditions"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProvisioningPipeline provisions Buckets using S3 client.
type ProvisioningPipeline struct{}

// BucketFinalizer is the name of the finalizer to protect unchecked deletions.
const BucketFinalizer = "s3.appcat.vshn.io/bucket-protection"

// NewProvisioningPipeline returns a new instance of ProvisioningPipeline.
func NewProvisioningPipeline() *ProvisioningPipeline {
	return &ProvisioningPipeline{}
}

// CredentialsSecretKey identifies the loaded credentials secret in the context.
type CredentialsSecretKey struct{}

// Run executes the business logic.
func (p *ProvisioningPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(BucketKey{}, BucketFinalizer)),
			pipeline.NewStepFromFunc("prevent bucket rename", preventBucketRename),
			pipeline.If(bucketNotExisting, pipeline.NewPipeline().WithNestedSteps("provision bucket",
				pipeline.NewStepFromFunc("fetch credentials secret", fetchCredentialsSecret),
				pipeline.NewStepFromFunc("validate secret", validateSecret),
				pipeline.NewStepFromFunc("create S3 client", CreateS3Client),
				pipeline.NewStepFromFunc("create bucket", CreateS3Bucket),
				pipeline.NewStepFromFunc("set bucket name in status", steps.UpdateStatusFn(BucketKey{})),
				pipeline.NewStepFromFunc("emit event", emitCreationEvent),
			)),
			pipeline.NewStepFromFunc("set status condition", steps.MarkObjectReadyFn(BucketKey{})),
		).
		WithFinalizer(steps.ErrorHandlerFn(BucketKey{}, conditions.ReasonProvisioningFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func bucketNotExisting(ctx context.Context) bool {
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*cloudscalev1.Bucket)

	return bucket.Status.AtProvider.BucketName == ""
}

func preventBucketRename(ctx context.Context) error {
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*cloudscalev1.Bucket)

	if bucket.Status.AtProvider.BucketName == "" {
		// we don't know the previous bucket name
		return nil
	}
	if bucket.Status.AtProvider.BucketName != bucket.GetBucketName() {
		return fmt.Errorf("a bucket named %q has been previously created, you cannot rename it. Either revert 'spec.bucketName' back to %q or delete the bucket and recreate using a new name",
			bucket.Status.AtProvider.BucketName, bucket.Status.AtProvider.BucketName)
	}
	return nil
}

func fetchCredentialsSecret(ctx context.Context) error {
	kube := steps.GetClientFromContext(ctx)
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*cloudscalev1.Bucket)
	log := controllerruntime.LoggerFrom(ctx)

	secret := &corev1.Secret{}
	name := bucket.Spec.ForProvider.CredentialsSecretRef.Name
	namespace := bucket.Spec.ForProvider.CredentialsSecretRef.Namespace
	err := kube.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	pipeline.StoreInContext(ctx, CredentialsSecretKey{}, secret)
	return logIfNotError(err, log, 1, "Fetched credentials secret", "secret name", fmt.Sprintf("%s/%s", namespace, name))
}

func validateSecret(ctx context.Context) error {
	secret := steps.GetFromContextOrPanic(ctx, CredentialsSecretKey{}).(*corev1.Secret)

	if secret.Data == nil {
		return fmt.Errorf("secret %q does not have any data", secret.Name)
	}

	requiredKeys := []string{cloudscalev1.AccessKeyIDName, cloudscalev1.SecretAccessKeyName}
	for _, key := range requiredKeys {
		if val, exists := secret.Data[key]; !exists || string(val) == "" {
			return fmt.Errorf("secret %q is missing on of the following keys or content: %s", fmt.Sprintf("%s/%s", secret.Namespace, secret.Name), requiredKeys)
		}
	}
	return nil
}

func emitCreationEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	obj := steps.GetFromContextOrPanic(ctx, BucketKey{}).(client.Object)

	recorder.Event(obj, corev1.EventTypeNormal, "Created", "Bucket successfully created")
	return nil
}

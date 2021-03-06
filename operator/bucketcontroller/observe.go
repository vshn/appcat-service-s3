package bucketcontroller

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *ProvisioningPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	s3Client := p.minio
	bucket := fromManaged(mg)

	if err := preventBucketRename(bucket); err != nil {
		bucket.SetConditions(xpv1.Unavailable())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, err
	}
	if err := preventRegionChange(bucket); err != nil {
		bucket.SetConditions(xpv1.Unavailable())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, err
	}

	bucketName := bucket.GetBucketName()
	exists, err := s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot determine whether bucket exists")
	}
	if exists {
		bucket.Status.AtProvider.BucketName = bucketName
		bucket.Status.AtProvider.Region = bucket.Spec.ForProvider.Region
		bucket.SetConditions(xpv1.Available())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	}
	return managed.ExternalObservation{}, nil
}

func preventBucketRename(bucket *cloudscalev1.Bucket) error {
	if bucket.Status.AtProvider.BucketName == "" {
		// we don't know the previous properties
		return nil
	}
	if bucket.Status.AtProvider.BucketName != bucket.GetBucketName() {
		return fmt.Errorf("a bucket named %q has been previously created, you cannot rename it. Either revert 'spec.forProvider.bucketName' back to %q or delete the bucket and recreate using a new name",
			bucket.Status.AtProvider.BucketName, bucket.Status.AtProvider.BucketName)
	}
	return nil
}

func preventRegionChange(bucket *cloudscalev1.Bucket) error {
	if bucket.Status.AtProvider.Region == "" {
		// we don't know the previous property
		return nil
	}
	if bucket.Status.AtProvider.Region != bucket.Spec.ForProvider.Region {
		return fmt.Errorf("a bucket named %q has been previously created with region %q, you cannot change the region. Either revert 'spec.forProvider.region' back to %q or delete the bucket and recreate in new region",
			bucket.Status.AtProvider.BucketName, bucket.Status.AtProvider.Region, bucket.Status.AtProvider.Region)
	}
	return nil
}

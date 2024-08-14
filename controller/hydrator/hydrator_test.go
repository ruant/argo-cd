package hydrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

func Test_appNeedsHydration(t *testing.T) {
	t.Parallel()

	now := metav1.NewTime(time.Now())
	oneHourAgo := metav1.NewTime(now.Add(-1 * time.Hour))

	testCases := []struct {
		name           string
		app            *v1alpha1.Application
		timeout        time.Duration
		latestRevision string
		expected       string
	}{
		{
			name:     "source hydrator not configured",
			app:      &v1alpha1.Application{},
			expected: "source hydrator not configured",
		},
		{
			name: "hydrate requested",
			app: &v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{v1alpha1.AnnotationKeyHydrate: "normal"}},
				Spec:       v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "hydrate requested",
		},
		{
			name: "no previous hydrate operation",
			app: &v1alpha1.Application{
				Spec: v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "no previous hydrate operation",
		},
		{
			name: "spec.sourceHydrator differs",
			app: &v1alpha1.Application{
				Spec: v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
				Status: v1alpha1.ApplicationStatus{SourceHydrator: v1alpha1.SourceHydratorStatus{CurrentOperation: &v1alpha1.HydrateOperation{
					SourceHydrator: v1alpha1.SourceHydrator{DrySource: v1alpha1.DrySource{RepoURL: "something new"}},
				}}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "spec.sourceHydrator differs",
		},
		{
			name: "dry SHA has changed",
			app: &v1alpha1.Application{
				Spec:   v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
				Status: v1alpha1.ApplicationStatus{SourceHydrator: v1alpha1.SourceHydratorStatus{CurrentOperation: &v1alpha1.HydrateOperation{DrySHA: "xyz123"}}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "revision differs",
		},
		{
			name: "hydration failed more than two minutes ago",
			app: &v1alpha1.Application{
				Spec:   v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
				Status: v1alpha1.ApplicationStatus{SourceHydrator: v1alpha1.SourceHydratorStatus{CurrentOperation: &v1alpha1.HydrateOperation{DrySHA: "abc123", FinishedAt: &oneHourAgo, Phase: v1alpha1.HydrateOperationPhaseFailed}}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "previous hydrate operation failed more than 2 minutes ago",
		},
		{
			name: "timeout reached",
			app: &v1alpha1.Application{
				Spec:   v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
				Status: v1alpha1.ApplicationStatus{SourceHydrator: v1alpha1.SourceHydratorStatus{CurrentOperation: &v1alpha1.HydrateOperation{StartedAt: oneHourAgo}}},
			},
			timeout:        1 * time.Minute,
			latestRevision: "",
			expected:       "hydration expired",
		},
		{
			name: "hydrate not needed",
			app: &v1alpha1.Application{
				Spec:   v1alpha1.ApplicationSpec{SourceHydrator: &v1alpha1.SourceHydrator{}},
				Status: v1alpha1.ApplicationStatus{SourceHydrator: v1alpha1.SourceHydratorStatus{CurrentOperation: &v1alpha1.HydrateOperation{DrySHA: "abc123", StartedAt: now, FinishedAt: &now, Phase: v1alpha1.HydrateOperationPhaseFailed}}},
			},
			timeout:        1 * time.Hour,
			latestRevision: "abc123",
			expected:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := appNeedsHydration(tc.app, tc.timeout, tc.latestRevision)
			assert.Equal(t, tc.expected, result)
		})
	}
}

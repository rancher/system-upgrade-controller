package job_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	sucjob "github.com/rancher/system-upgrade-controller/pkg/upgrade/job"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJob(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Job Suite")
}

var _ = Describe("Jobs", func() {
	var plan *upgradev1.Plan
	var node *corev1.Node

	BeforeEach(func() {
		plan = &upgradev1.Plan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: "default",
			},
			Spec: upgradev1.PlanSpec{
				Concurrency:        1,
				ServiceAccountName: "system-upgrade-controller-foo",
				Upgrade: &upgradev1.ContainerSpec{
					Image: "test-image:latest",
				},
			},
		}

		node = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "prod.test.local",
			},
		}
	})

	Describe("Setting the batchv1.Job ActiveDeadlineSeconds field", func() {
		Context("When the Plan has a positive non-zero value for deadline", func() {
			It("Constructs the batchv1.Job with the Plan's given value", func() {
				plan.Spec.JobActiveDeadlineSecs = 12345
				job := sucjob.New(plan, node, "foo")
				Expect(*job.Spec.ActiveDeadlineSeconds).To(Equal(int64(12345)))
			})
		})

		Context("When the Plan has a zero-value given as its deadline", func() {
			It("Constructs the batchv1.Job with a global default", func() {
				oldActiveDeadlineSeconds := sucjob.ActiveDeadlineSeconds
				sucjob.ActiveDeadlineSeconds = 300
				defer func() { sucjob.ActiveDeadlineSeconds = oldActiveDeadlineSeconds }()

				plan.Spec.JobActiveDeadlineSecs = 0
				job := sucjob.New(plan, node, "bar")
				Expect(*job.Spec.ActiveDeadlineSeconds).To(Equal(int64(300)))
			})
		})

		Context("When the Plan has a negative value given as its deadline", func() {
			It("Constructs the batchv1.Job with a global default", func() {
				oldActiveDeadlineSeconds := sucjob.ActiveDeadlineSeconds
				sucjob.ActiveDeadlineSeconds = 3600
				defer func() { sucjob.ActiveDeadlineSeconds = oldActiveDeadlineSeconds }()

				plan.Spec.JobActiveDeadlineSecs = -1
				job := sucjob.New(plan, node, "baz")
				Expect(*job.Spec.ActiveDeadlineSeconds).To(Equal(int64(3600)))
			})
		})

		Context("When cluster has a maximum deadline and the Plan deadline exceeds that value", func() {
			It("Constructs the batchv1.Job with the cluster's maximum deadline value", func() {
				oldActiveDeadlineSecondsMax := sucjob.ActiveDeadlineSecondsMax
				sucjob.ActiveDeadlineSecondsMax = 300
				defer func() { sucjob.ActiveDeadlineSecondsMax = oldActiveDeadlineSecondsMax }()

				plan.Spec.JobActiveDeadlineSecs = 600
				job := sucjob.New(plan, node, "foobar")
				Expect(*job.Spec.ActiveDeadlineSeconds).To(Equal(int64(300)))
			})
		})

		Context("When the Plan has annotations and labels", func() {
			It("Copies the non-cattle.io metadata to the Job and Pod", func() {
				plan.Annotations = make(map[string]string)
				plan.Annotations["cattle.io/some-annotation"] = "foo"
				plan.Annotations["plan.cattle.io/some-annotation"] = "bar"
				plan.Annotations["some.other/annotation"] = "baz"
				plan.Labels = make(map[string]string)
				plan.Labels["cattle.io/some-label"] = "biz"
				plan.Labels["plan.cattle.io/some-label"] = "buz"
				plan.Labels["some.other/label"] = "bla"

				job := sucjob.New(plan, node, "foobar")
				Expect(job.Annotations).To(Not(HaveKey("cattle.io/some-annotation")))
				Expect(job.Annotations).To(Not(HaveKey("plan.cattle.io/some-annotation")))
				Expect(job.Annotations).To(HaveKeyWithValue("some.other/annotation", "baz"))
				Expect(job.Labels).To(Not(HaveKey("cattle.io/some-label")))
				Expect(job.Labels).To(Not(HaveKey("plan.cattle.io/some-label")))
				Expect(job.Labels).To(HaveKeyWithValue("some.other/label", "bla"))

				Expect(job.Spec.Template.Annotations).To(Not(HaveKey("cattle.io/some-annotation")))
				Expect(job.Spec.Template.Annotations).To(Not(HaveKey("plan.cattle.io/some-annotation")))
				Expect(job.Spec.Template.Annotations).To(HaveKeyWithValue("some.other/annotation", "baz"))
				Expect(job.Spec.Template.Labels).To(Not(HaveKey("cattle.io/some-label")))
				Expect(job.Spec.Template.Labels).To(Not(HaveKey("plan.cattle.io/some-label")))
				Expect(job.Spec.Template.Labels).To(HaveKeyWithValue("some.other/label", "bla"))
			})
		})
	})
})

package suite_test

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/utils/pointer"

	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Job Generation", func() {
	e2e := framework.New("generate")

	When("fails because of a bad plan", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
			jobs []batchv1.Job
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("fail-then-succeed-", "library/alpine:3.11", []string{"sh", "-c"}, "exit 1")
			plan.Spec.Version = "latest"
			plan.Spec.Concurrency = 1
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "node-role.kubernetes.io/master",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				}},
			}
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(BeTrue())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Status.Succeeded).To(BeNumerically("==", 0))
			Expect(jobs[0].Status.Failed).To(BeNumerically(">=", 1))

			plan, err = e2e.GetPlan(plan.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			plan.Spec.Upgrade.Args = []string{"exit 0"}
			plan, err = e2e.UpdatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
		})
		It("should apply successfully after edit", func() {
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Status.Succeeded).To(BeNumerically("==", 1))
			Expect(jobs[0].Status.Failed).To(BeNumerically("==", 0))
		})
	})

	When("fails because of conflicting drain options", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
			jobs []batchv1.Job
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("fail-drain-options-", "library/alpine:3.11", []string{"sh", "-c"}, "exit 0")
			plan.Spec.Version = "latest"
			plan.Spec.Concurrency = 1
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "node-role.kubernetes.io/master",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				}},
			}
			plan.Spec.Drain = &upgradeapiv1.DrainSpec{
				DisableEviction:    true,
				DeleteLocalData:    pointer.Bool(true),
				DeleteEmptydirData: pointer.Bool(true),
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "app",
						Values:   []string{"csi-attacher", "csi-provisioner"},
						Operator: metav1.LabelSelectorOpNotIn,
					}},
				},
			}
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanSpecValidated, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(upgradeapiv1.PlanSpecValidated.IsTrue(plan)).To(BeFalse())
			Expect(upgradeapiv1.PlanSpecValidated.GetMessage(plan)).To(ContainSubstring("cannot specify both deleteEmptydirData and deleteLocalData"))

			plan.Spec.Drain.DeleteLocalData = nil
			plan, err = e2e.UpdatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanSpecValidated, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(upgradeapiv1.PlanSpecValidated.IsTrue(plan)).To(BeTrue())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(jobs).To(HaveLen(1))
		})
		It("should apply successfully after edit", func() {
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Status.Succeeded).To(BeNumerically("==", 1))
			Expect(jobs[0].Status.Failed).To(BeNumerically("==", 0))
			Expect(jobs[0].Spec.Template.Spec.InitContainers).To(HaveLen(1))
			Expect(jobs[0].Spec.Template.Spec.InitContainers[0].Args).To(ContainSubstring("csi-attacher"))
		})
	})
})

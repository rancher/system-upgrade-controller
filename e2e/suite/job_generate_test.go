package suite_test

import (
	"context"
	"fmt"
	"io"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			plan = e2e.NewPlan("fail-then-succeed-", "library/alpine:3.18", []string{"sh", "-c"}, "exit 1")
			plan.Spec.Version = "latest"
			plan.Spec.Concurrency = 1
			plan.Spec.ServiceAccountName = e2e.Namespace.Name
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "node-role.kubernetes.io/control-plane",
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
			Expect(jobs[0].Status.Active).To(BeNumerically("==", 0))
			Expect(jobs[0].Status.Failed).To(BeNumerically(">=", 1))

			Eventually(e2e.GetPlan).
				WithArguments(plan.Name, metav1.GetOptions{}).
				WithTimeout(30 * time.Second).
				Should(WithTransform(upgradeapiv1.PlanComplete.IsTrue, BeFalse()))

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
			Expect(jobs[0].Status.Active).To(BeNumerically("==", 0))
			Expect(jobs[0].Status.Failed).To(BeNumerically("==", 0))

			Eventually(e2e.GetPlan).
				WithArguments(plan.Name, metav1.GetOptions{}).
				WithTimeout(30 * time.Second).
				Should(WithTransform(upgradeapiv1.PlanComplete.IsTrue, BeTrue()))
		})
	})

	When("fails because of conflicting drain options", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
			jobs []batchv1.Job
		)
		BeforeEach(func() {
			plan = e2e.NewPlan("fail-drain-options-", "library/alpine:3.18", []string{"sh", "-c"}, "exit 0")
			plan.Spec.Version = "latest"
			plan.Spec.Concurrency = 1
			plan.Spec.ServiceAccountName = e2e.Namespace.Name
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				}},
			}
			plan.Spec.Drain = &upgradeapiv1.DrainSpec{
				DisableEviction:    true,
				DeleteLocalData:    pointer.Bool(true),
				DeleteEmptydirData: pointer.Bool(true),
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "component",
						Values:   []string{"sonobuoy"},
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
			Expect(jobs[0].Status.Active).To(BeNumerically("==", 0))
			Expect(jobs[0].Status.Failed).To(BeNumerically("==", 0))
			Expect(jobs[0].Spec.Template.Spec.InitContainers).To(HaveLen(1))
			Expect(jobs[0].Spec.Template.Spec.InitContainers[0].Args).To(ContainElement(ContainSubstring("!upgrade.cattle.io/controller")))
			Expect(jobs[0].Spec.Template.Spec.InitContainers[0].Args).To(ContainElement(ContainSubstring("component notin (sonobuoy)")))
		})
		AfterEach(func() {
			if CurrentSpecReport().Failed() {
				podList, _ := e2e.PodClient().List(context.Background(), metav1.ListOptions{})
				for _, pod := range podList.Items {
					containerNames := []string{}
					for _, container := range pod.Spec.InitContainers {
						containerNames = append(containerNames, container.Name)
					}
					for _, container := range pod.Spec.Containers {
						containerNames = append(containerNames, container.Name)
					}
					for _, container := range containerNames {
						reportName := fmt.Sprintf("podlogs-%s-%s", pod.Name, container)
						logs := e2e.PodClient().GetLogs(pod.Name, &v1.PodLogOptions{Container: container})
						if logStreamer, err := logs.Stream(context.Background()); err == nil {
							if podLogs, err := io.ReadAll(logStreamer); err == nil {
								AddReportEntry(reportName, string(podLogs))
							}
						}
					}
				}
			}
		})
	})

	When("updated secret should not change hash", func() {
		var (
			err    error
			plan   *upgradeapiv1.Plan
			secret *v1.Secret
			hash   string
		)
		BeforeEach(func() {
			secret, err = e2e.CreateSecret(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: e2e.Namespace.Name,
				},
				Data: map[string][]byte{"config": []byte("test")},
			})
			Expect(err).ToNot(HaveOccurred())

			plan = e2e.NewPlan("updated-secret-", "library/alpine:3.18", []string{"sh", "-c"}, "exit 0")
			plan.Spec.Version = "latest"
			plan.Spec.Concurrency = 1
			plan.Spec.ServiceAccountName = e2e.Namespace.Name
			plan.Spec.NodeSelector = &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				}},
			}
			plan.Spec.Secrets = []upgradeapiv1.SecretSpec{{
				Name:          "test",
				Path:          "/test",
				IgnoreUpdates: true,
			}}
			plan, err = e2e.CreatePlan(plan)
			Expect(err).ToNot(HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(plan.Status.LatestHash).ToNot(BeEmpty())
			hash = plan.Status.LatestHash

			secret.Data = map[string][]byte{"config": []byte("test2")}
			secret, err = e2e.UpdateSecret(secret)
			Expect(err).ToNot(HaveOccurred())
		})
		It("hash should be equal", func() {
			Expect(plan.Status.LatestHash).Should(Equal(hash))
		})
	})
})

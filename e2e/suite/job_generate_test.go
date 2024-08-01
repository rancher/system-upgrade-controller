package suite_test

import (
	"context"
	"fmt"
	"io"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/rancher/system-upgrade-controller/e2e/framework"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Job Generation", func() {
	e2e := framework.New("generate")

	ginkgo.When("fails because of a bad plan", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
			jobs []batchv1.Job
		)
		ginkgo.BeforeEach(func() {
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
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(upgradeapiv1.PlanLatestResolved.IsTrue(plan)).To(gomega.BeTrue())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(jobs).To(gomega.HaveLen(1))
			gomega.Expect(jobs[0].Status.Succeeded).To(gomega.BeNumerically("==", 0))
			gomega.Expect(jobs[0].Status.Active).To(gomega.BeNumerically("==", 0))
			gomega.Expect(jobs[0].Status.Failed).To(gomega.BeNumerically(">=", 1))

			gomega.Eventually(e2e.GetPlan).
				WithArguments(plan.Name, metav1.GetOptions{}).
				WithTimeout(30 * time.Second).
				Should(gomega.WithTransform(upgradeapiv1.PlanComplete.IsTrue, gomega.BeFalse()))

			plan, err = e2e.GetPlan(plan.Name, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			plan.Spec.Upgrade.Args = []string{"exit 0"}
			plan, err = e2e.UpdatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(jobs).To(gomega.HaveLen(1))
		})
		ginkgo.It("should apply successfully after edit", func() {
			gomega.Expect(jobs).To(gomega.HaveLen(1))
			gomega.Expect(jobs[0].Status.Succeeded).To(gomega.BeNumerically("==", 1))
			gomega.Expect(jobs[0].Status.Active).To(gomega.BeNumerically("==", 0))
			gomega.Expect(jobs[0].Status.Failed).To(gomega.BeNumerically("==", 0))

			gomega.Eventually(e2e.GetPlan).
				WithArguments(plan.Name, metav1.GetOptions{}).
				WithTimeout(30 * time.Second).
				Should(gomega.WithTransform(upgradeapiv1.PlanComplete.IsTrue, gomega.BeTrue()))
		})
	})

	ginkgo.When("fails because of conflicting drain options", func() {
		var (
			err  error
			plan *upgradeapiv1.Plan
			jobs []batchv1.Job
		)
		ginkgo.BeforeEach(func() {
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
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanSpecValidated, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(upgradeapiv1.PlanSpecValidated.IsTrue(plan)).To(gomega.BeFalse())
			gomega.Expect(upgradeapiv1.PlanSpecValidated.GetMessage(plan)).To(gomega.ContainSubstring("cannot specify both deleteEmptydirData and deleteLocalData"))

			plan.Spec.Drain.DeleteLocalData = nil
			plan, err = e2e.UpdatePlan(plan)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanSpecValidated, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(upgradeapiv1.PlanSpecValidated.IsTrue(plan)).To(gomega.BeTrue())

			jobs, err = e2e.WaitForPlanJobs(plan, 1, 120*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(jobs).To(gomega.HaveLen(1))
		})
		ginkgo.It("should apply successfully after edit", func() {
			gomega.Expect(jobs).To(gomega.HaveLen(1))
			gomega.Expect(jobs[0].Status.Succeeded).To(gomega.BeNumerically("==", 1))
			gomega.Expect(jobs[0].Status.Active).To(gomega.BeNumerically("==", 0))
			gomega.Expect(jobs[0].Status.Failed).To(gomega.BeNumerically("==", 0))
			gomega.Expect(jobs[0].Spec.Template.Spec.InitContainers).To(gomega.HaveLen(1))
			gomega.Expect(jobs[0].Spec.Template.Spec.InitContainers[0].Args).To(gomega.ContainElement(gomega.ContainSubstring("!upgrade.cattle.io/controller")))
			gomega.Expect(jobs[0].Spec.Template.Spec.InitContainers[0].Args).To(gomega.ContainElement(gomega.ContainSubstring("component notin (sonobuoy)")))
		})
		ginkgo.AfterEach(func() {
			if ginkgo.CurrentSpecReport().Failed() {
				podList, _ := e2e.ClientSet.CoreV1().Pods(e2e.Namespace.Name).List(context.Background(), metav1.ListOptions{})
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
						logs := e2e.ClientSet.CoreV1().Pods(e2e.Namespace.Name).GetLogs(pod.Name, &v1.PodLogOptions{Container: container})
						if logStreamer, err := logs.Stream(context.Background()); err == nil {
							if podLogs, err := io.ReadAll(logStreamer); err == nil {
								ginkgo.AddReportEntry(reportName, string(podLogs))
							}
						}
					}
				}
			}
		})
	})

	ginkgo.When("updated secret should not change hash", func() {
		var (
			err    error
			plan   *upgradeapiv1.Plan
			secret *v1.Secret
			hash   string
		)
		ginkgo.BeforeEach(func() {
			secret, err = e2e.CreateSecret(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: e2e.Namespace.Name,
				},
				Data: map[string][]byte{"config": []byte("test")},
			})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

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
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			plan, err = e2e.WaitForPlanCondition(plan.Name, upgradeapiv1.PlanLatestResolved, 30*time.Second)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(plan.Status.LatestHash).ToNot(gomega.BeEmpty())
			hash = plan.Status.LatestHash

			secret.Data = map[string][]byte{"config": []byte("test2")}
			secret, err = e2e.UpdateSecret(secret)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgo.It("hash should be Equal", func() {
			gomega.Expect(plan.Status.LatestHash).Should(gomega.Equal(hash))
		})
	})
})

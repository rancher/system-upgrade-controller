package container_test

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/system-upgrade-controller/pkg/upgrade/container"
	corev1 "k8s.io/api/core/v1"
)

var _ = ginkgo.Describe("Container", func() {
	var (
		zeroContainer *corev1.Container
		testContainer corev1.Container
		testOption    container.Option
	)
	ginkgo.BeforeEach(func() {
		zeroContainer = &corev1.Container{}
		testContainer = *zeroContainer
		gomega.Expect(zeroContainer).To(gomega.Equal(&testContainer))
		gomega.Expect(testContainer).To(gomega.BeZero())
	})

	ginkgo.Context("WithImagePullPolicy", func() {
		const testPullPolicy = corev1.PullPolicy("Onlyginkgo.WhenTesting")
		ginkgo.BeforeEach(func() {
			testOption = container.WithImagePullPolicy(testPullPolicy)
			gomega.Expect(testContainer.ImagePullPolicy).To(gomega.BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.ImagePullPolicy = ""
		})
		ginkgo.It("should have ImagePullPolicy with no side effects", func() {
			gomega.Expect(testContainer.ImagePullPolicy).To(gomega.Equal(testPullPolicy))
			gomega.Expect(*zeroContainer).To(gomega.BeZero())
		})
	})

	ginkgo.Context("WithLatestTag", func() {
		const testImageRegistry = "img.example.com:5000"
		const testImagePath = "test/image"
		const testImageTag = "test"
		ginkgo.BeforeEach(func() {
			testOption = container.WithLatestTag(testImageTag)
		})
		ginkgo.When("image ref is without domain and tag, e.g. "+testImagePath, func() {
			ginkgo.BeforeEach(func() {
				testContainer.Image = testImagePath
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			ginkgo.It("should have tag override with no side effects", func() {
				gomega.Expect(testContainer.Image).To(gomega.Equal(testImagePath + `:` + testImageTag))
				gomega.Expect(*zeroContainer).To(gomega.BeZero())
			})
		})
		const imageWithoutDomainWithTag = testImagePath + `:test-with-image-tag`
		ginkgo.When("image ref is without domain but with tag, e.g. "+imageWithoutDomainWithTag, func() {
			ginkgo.BeforeEach(func() {
				testContainer.Image = imageWithoutDomainWithTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			ginkgo.It("should have no side effects", func() {
				gomega.Expect(testContainer.Image).To(gomega.Equal(imageWithoutDomainWithTag))
				gomega.Expect(*zeroContainer).To(gomega.BeZero())
			})
		})
		const imageWithDomainWithoutTag = testImageRegistry + `/` + testImagePath
		ginkgo.When("image ref with domain and without tag, e.g. "+imageWithDomainWithoutTag, func() {
			ginkgo.BeforeEach(func() {
				testContainer.Image = imageWithDomainWithoutTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			ginkgo.It("should have overridden tag with no side effects", func() {
				gomega.Expect(testContainer.Image).To(gomega.Equal(imageWithDomainWithoutTag + `:` + testImageTag))
				gomega.Expect(*zeroContainer).To(gomega.BeZero())
			})
		})
		const imageWithDomainAndTag = testImageRegistry + `/` + testImagePath + `:test-with-image-tag`
		ginkgo.When("image ref with domain and tag, e.g. "+imageWithDomainAndTag, func() {
			ginkgo.BeforeEach(func() {
				testContainer.Image = imageWithDomainAndTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			ginkgo.It("should have no side effects", func() {
				gomega.Expect(testContainer.Image).To(gomega.Equal(imageWithDomainAndTag))
				gomega.Expect(*zeroContainer).To(gomega.BeZero())
			})
		})
	})

	ginkgo.Context("WithPlanEnvironment", func() {
		var testPlanStatus = upgradeapiv1.PlanStatus{
			LatestVersion: "test",
			LatestHash:    "test-hash",
		}
		ginkgo.BeforeEach(func() {
			testOption = container.WithPlanEnvironment("test", testPlanStatus)
			gomega.Expect(testContainer.Env).To(gomega.BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.Env = nil
		})
		ginkgo.It("should have Env with no side effects", func() {
			gomega.Expect(testContainer.Env).ToNot(gomega.BeEmpty())
			gomega.Expect(*zeroContainer).To(gomega.BeZero())
		})

	})

	ginkgo.Context("WithSecrets", func() {
		var (
			testSecretHavingPath = upgradeapiv1.SecretSpec{
				Name: "having-path", Path: "/run/secret/having-path",
			}
			testSecretNotHavingPath = upgradeapiv1.SecretSpec{
				Name: "not-having-path",
			}
			testSecretHavingLongName = upgradeapiv1.SecretSpec{
				Name: "not-having-path-with-a-supercalifragilisticexpialidocious-name",
			}
			testSecrets = []upgradeapiv1.SecretSpec{
				testSecretHavingPath, testSecretNotHavingPath, testSecretHavingLongName,
			}
		)
		ginkgo.BeforeEach(func() {
			testOption = container.WithSecrets(testSecrets)
			gomega.Expect(testContainer.VolumeMounts).To(gomega.BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.VolumeMounts = nil
		})
		ginkgo.It("should have VolumeMounts with no side effects", func() {
			gomega.Expect(testContainer.VolumeMounts).To(gomega.HaveLen(len(testSecrets)))
			gomega.Expect(*zeroContainer).To(gomega.BeZero())
			for i, testSecret := range testSecrets {
				ginkgo.By("having VolumeMount for Secret `"+testSecret.Name+"`", func() {
					if len(testSecret.Name) > 50 {
						gomega.Expect(testContainer.VolumeMounts[i].Name).To(gomega.ContainSubstring(testSecret.Name[:50]))
					} else {
						gomega.Expect(testContainer.VolumeMounts[i].Name).To(gomega.HaveSuffix(testSecret.Name))
					}
					if testSecret.Path != "" {
						gomega.Expect(testContainer.VolumeMounts[i].MountPath).To(gomega.Equal(testSecret.Path))
					}
				})
			}
		})
	})

	ginkgo.Context("WithSecurityContext", func() {
		var privileged = true
		var testSecurityContext corev1.SecurityContext = corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"TEST_CONTAINER",
				},
			},
			Privileged: &privileged,
		}
		ginkgo.BeforeEach(func() {
			testOption = container.WithSecurityContext(&testSecurityContext)
			gomega.Expect(testContainer.SecurityContext).To(gomega.BeNil())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.SecurityContext = nil
		})
		ginkgo.It("should have SecurityContext with no side effects", func() {
			gomega.Expect(testContainer.SecurityContext).To(gomega.Equal(&testSecurityContext))
			gomega.Expect(*zeroContainer).To(gomega.BeZero())
		})
	})
})

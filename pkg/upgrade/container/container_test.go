package container_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upgradeapiv1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/system-upgrade-controller/pkg/upgrade/container"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Container", func() {
	var (
		zeroContainer *corev1.Container
		testContainer corev1.Container
		testOption    container.Option
	)
	BeforeEach(func() {
		zeroContainer = &corev1.Container{}
		testContainer = *zeroContainer
		Expect(zeroContainer).To(Equal(&testContainer))
		Expect(testContainer).To(BeZero())
	})

	Context("WithImagePullPolicy", func() {
		const testPullPolicy = corev1.PullPolicy("OnlyWhenTesting")
		BeforeEach(func() {
			testOption = container.WithImagePullPolicy(testPullPolicy)
			Expect(testContainer.ImagePullPolicy).To(BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.ImagePullPolicy = ""
		})
		It("should have ImagePullPolicy with no side effects", func() {
			Expect(testContainer.ImagePullPolicy).To(Equal(testPullPolicy))
			Expect(*zeroContainer).To(BeZero())
		})
	})

	Context("WithLatestTag", func() {
		const testImageRegistry = "img.example.com:5000"
		const testImagePath = "test/image"
		const testImageTag = "test"
		BeforeEach(func() {
			testOption = container.WithLatestTag(testImageTag)
		})
		When("image ref is without domain and tag, e.g. "+testImagePath, func() {
			BeforeEach(func() {
				testContainer.Image = testImagePath
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			It("should have tag override with no side effects", func() {
				Expect(testContainer.Image).To(Equal(testImagePath + `:` + testImageTag))
				Expect(*zeroContainer).To(BeZero())
			})
		})
		const imageWithoutDomainWithTag = testImagePath + `:test-with-image-tag`
		When("image ref is without domain but with tag, e.g. "+imageWithoutDomainWithTag, func() {
			BeforeEach(func() {
				testContainer.Image = imageWithoutDomainWithTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			It("should have no side effects", func() {
				Expect(testContainer.Image).To(Equal(imageWithoutDomainWithTag))
				Expect(*zeroContainer).To(BeZero())
			})
		})
		const imageWithDomainWithoutTag = testImageRegistry + `/` + testImagePath
		When("image ref with domain and without tag, e.g. "+imageWithDomainWithoutTag, func() {
			BeforeEach(func() {
				testContainer.Image = imageWithDomainWithoutTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			It("should have overridden tag with no side effects", func() {
				Expect(testContainer.Image).To(Equal(imageWithDomainWithoutTag + `:` + testImageTag))
				Expect(*zeroContainer).To(BeZero())
			})
		})
		const imageWithDomainAndTag = testImageRegistry + `/` + testImagePath + `:test-with-image-tag`
		When("image ref with domain and tag, e.g. "+imageWithDomainAndTag, func() {
			BeforeEach(func() {
				testContainer.Image = imageWithDomainAndTag
				testOption(&testContainer) // apply the option
				*zeroContainer = testContainer
				zeroContainer.Image = ""
			})
			It("should have no side effects", func() {
				Expect(testContainer.Image).To(Equal(imageWithDomainAndTag))
				Expect(*zeroContainer).To(BeZero())
			})
		})
	})

	Context("WithPlanEnvironment", func() {
		var testPlanStatus = upgradeapiv1.PlanStatus{
			LatestVersion: "test",
			LatestHash:    "test-hash",
		}
		BeforeEach(func() {
			testOption = container.WithPlanEnvironment("test", testPlanStatus)
			Expect(testContainer.Env).To(BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.Env = nil
		})
		It("should have Env with no side effects", func() {
			Expect(testContainer.Env).ToNot(BeEmpty())
			Expect(*zeroContainer).To(BeZero())
		})

	})

	Context("WithSecrets", func() {
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
		BeforeEach(func() {
			testOption = container.WithSecrets(testSecrets)
			Expect(testContainer.VolumeMounts).To(BeEmpty())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.VolumeMounts = nil
		})
		It("should have VolumeMounts with no side effects", func() {
			Expect(testContainer.VolumeMounts).To(HaveLen(len(testSecrets)))
			Expect(*zeroContainer).To(BeZero())
			for i, testSecret := range testSecrets {
				By("having VolumeMount for Secret `"+testSecret.Name+"`", func() {
					if len(testSecret.Name) > 50 {
						Expect(testContainer.VolumeMounts[i].Name).To(ContainSubstring(testSecret.Name[:50]))
					} else {
						Expect(testContainer.VolumeMounts[i].Name).To(HaveSuffix(testSecret.Name))
					}
					if testSecret.Path != "" {
						Expect(testContainer.VolumeMounts[i].MountPath).To(Equal(testSecret.Path))
					}
				})
			}
		})
	})

	Context("WithSecurityContext", func() {
		var privileged = true
		var testSecurityContext corev1.SecurityContext = corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"TEST_CONTAINER",
				},
			},
			Privileged: &privileged,
		}
		BeforeEach(func() {
			testOption = container.WithSecurityContext(&testSecurityContext)
			Expect(testContainer.SecurityContext).To(BeNil())
			testOption(&testContainer) // apply the option
			*zeroContainer = testContainer
			zeroContainer.SecurityContext = nil
		})
		It("should have SecurityContext with no side effects", func() {
			Expect(testContainer.SecurityContext).To(Equal(&testSecurityContext))
			Expect(*zeroContainer).To(BeZero())
		})
	})
})

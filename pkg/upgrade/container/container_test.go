package container_test

import (
	"path/filepath"

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
	Describe("Applying Options", func() {
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

		Context("WithImageTag", func() {
			const testRepository = "img.example.com:5000"
			const testImageInfix = "this/is/a/test/image"
			const testImageTag = "test"
			BeforeEach(func() {
				testOption = container.WithImageTag(testImageTag)
			})
			for _, image := range []struct {
				img    string
				suffix string
			}{
				{testImageInfix, testImageInfix + `:` + testImageTag},
				{testImageInfix + ":latest", testImageInfix + ":latest"},
				{filepath.Join(testRepository, testImageInfix), testImageInfix + `:` + testImageTag},
				{filepath.Join(testRepository, testImageInfix+":latest"), testImageInfix + ":latest"},
			} {
				When("image is `"+image.img+"` and tag is `"+testImageTag+"` ", func() {
					BeforeEach(func() {
						testContainer.Image = image.img
						testOption(&testContainer) // apply the option
						*zeroContainer = testContainer
						zeroContainer.Image = ""
					})
					It("should have correct image suffix with no other side effects", func() {
						Expect(testContainer.Image).To(HaveSuffix(image.suffix))
						Expect(*zeroContainer).To(BeZero())
					})
				})
			}
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
})

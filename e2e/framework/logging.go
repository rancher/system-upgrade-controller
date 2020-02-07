package framework

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"k8s.io/kubernetes/test/e2e/framework/ginkgowrapper"
)

// Logf logs the info.
func Logf(level, format string, args ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, time.Now().Format(time.StampMilli)+": "+level+": "+format+"\n", args...)
}

func Infof(format string, args ...interface{}) {
	Logf("INFO", format, args)
}

func Warnf(format string, args ...interface{}) {
	Logf("WARN", format, args)
}

func Errorf(format string, args ...interface{}) {
	Logf("ERRO", format, args)
}

// Failf logs the fail info.
func Failf(format string, args ...interface{}) {
	FailfWithOffset(1, format, args...)
}

// FailfWithOffset calls "Fail" and logs the error at "offset" levels above its caller
// (for example, for call chain f -> g -> FailfWithOffset(1, ...) error would be logged for "f").
func FailfWithOffset(offset int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Logf("FAIL", msg)
	ginkgowrapper.Fail(nowStamp()+": "+msg, 1+offset)
}

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

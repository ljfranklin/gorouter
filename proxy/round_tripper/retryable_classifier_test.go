package round_tripper_test

import (
	"crypto/x509"
	"errors"
	"net"

	"code.cloudfoundry.org/gorouter/proxy/round_tripper"
	"code.cloudfoundry.org/gorouter/test_util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("roundTripperRetryableClassifier", func() {
	Context("IsRetryable", func() {
		var retry round_tripper.RoundTripperRetryableClassifier
		var err error
		BeforeEach(func() {
			retry = round_tripper.RoundTripperRetryableClassifier{}
		})
		AfterEach(func() {
			err = nil
		})
		Context("When error is a dial error", func() {
			BeforeEach(func() {
				err = &net.OpError{
					Err: errors.New("error"),
					Op:  "dial",
				}
			})
			It("returns true", func() {
				Expect(retry.IsRetryable(err)).To(BeTrue())
			})
		})
		Context("When error is a 'read: connection reset by peer' error", func() {
			BeforeEach(func() {
				err = &net.OpError{
					Err: errors.New("read: connection reset by peer"),
					Op:  "read",
				}
			})
			It("returns true", func() {
				Expect(retry.IsRetryable(err)).To(BeTrue())
			})
		})
		Context("When error is a HostnameError", func() {
			BeforeEach(func() {
				_, c := test_util.CreateCertDER("foobaz.com")
				var cert *x509.Certificate
				cert, err = x509.ParseCertificate(c)
				Expect(err).NotTo(HaveOccurred())
				err = &x509.HostnameError{
					Certificate: cert,
					Host:        "foobar.com",
				}
			})
			It("returns true", func() {
				Expect(retry.IsRetryable(err)).To(BeTrue())
			})
		})
		Context("When error is anything else", func() {
			BeforeEach(func() {
				err = &net.OpError{
					Err: errors.New("other error"),
					Op:  "write",
				}
			})
			It("returns true", func() {
				Expect(retry.IsRetryable(err)).To(BeFalse())
			})
		})
	})
})

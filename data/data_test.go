package data_test

import (
	. "github.com/nojyerac/semaphore/data"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Data", func() {
	var (
		flag *FeatureFlag
	)
	It("is testable", func() {
		flag = &FeatureFlag{
			ID:          "123",
			Name:        "test-flag",
			Description: "A test flag",
			Enabled:     true,
			Strategies: []Strategy{
				{
					Type: "user_targeting",
					Payload: []byte(`{
						"user_ids": ["user1", "user2"]
					}`),
				},
			},
		}
		Expect(flag).ToNot(BeNil())
	})
})

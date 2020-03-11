package services_test

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/jamillosantos/go-services"
)

var _ = Describe("Retrier", func() {
	Context("Resource", func() {
		It("should start and stop a service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := NewMockResource(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Name().Return("Service A"),
				serviceA.EXPECT().Start(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			serviceARetrier := services.Retrier().Backoff(backoff.NewExponentialBackOff()).Build(serviceA)
			Expect(serviceARetrier.Name()).To(Equal("Service A"))
			manager := services.NewRunner()
			Expect(manager.Run(ctx, serviceARetrier)).To(Succeed())
			Expect(manager.Finish(ctx)).To(Succeed())
		})

		It("should start a service after the third try waiting between each", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := NewMockResource(ctrl)

			reporter := NewMockRetrierReporter(ctrl)

			randomErr := errors.New("any error")
			gomock.InOrder(
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 1),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 2),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 3),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 4),
				serviceA.EXPECT().Start(gomock.Any()).Return(nil),
			)

			serviceARetrier := services.Retrier().Reporter(reporter).Backoff(backoff.NewConstantBackOff(time.Millisecond * 50)).Build(serviceA)
			runner := services.NewRunner()
			startedAt := time.Now()
			Expect(runner.Run(ctx, serviceARetrier)).To(Succeed())
			Expect(time.Since(startedAt)).To(BeNumerically("~", time.Millisecond*150, time.Millisecond*20))
		})

		It("should load a Configurable service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := &struct {
				*MockResource
				*MockConfigurable
			}{
				MockResource:     NewMockResource(ctrl),
				MockConfigurable: NewMockConfigurable(ctrl),
			}

			gomock.InOrder(
				serviceA.MockConfigurable.EXPECT().Load(gomock.Any()),
				serviceA.MockResource.EXPECT().Start(gomock.Any()),
			)

			serviceARetrier := services.Retrier().Build(serviceA)
			runner := services.NewRunner()
			Expect(runner.Run(ctx, serviceARetrier)).To(Succeed())
		})
	})
})

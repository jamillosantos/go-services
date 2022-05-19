package services_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	signals "github.com/jamillosantos/go-os-signals"
	"github.com/jamillosantos/go-os-signals/signaltest"
	. "github.com/onsi/ginkgo" // nolint
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamillosantos/go-services"
)

func createController() *gomock.Controller {
	return gomock.NewController(GinkgoT(1))
}

func TestRunner_Run(t *testing.T) {
	t.Run("resources", func(t *testing.T) {
		t.Run("should start Resouce instances", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
			)

			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			require.NoError(t, err)
		})

		t.Run("should not stop started Resource instances when one fails", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			wantErr := errors.New("random")

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()).Return(wantErr),
			)

			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			assert.ErrorIs(t, err, wantErr)
		})

		t.Run("should interrupt starting Resource instances", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx, cancelFunc := context.WithCancel(context.TODO())
			defer cancelFunc()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 90)
				}),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()).Do(func(_ context.Context) {
					time.Sleep(time.Second)
				}),
			)

			// 2. Create and Run the Runner
			runner := services.NewRunner()
			go func() {
				defer GinkgoRecover()

				// 3. Triggers the goroutine to interrupt the starting process before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 50)
				cancelFunc()
			}()

			// 4. Checks if the start was cancelled.
			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			assert.ErrorIs(t, err, context.Canceled)
			now := time.Now()
			_ = runner.Finish(context.Background())
			assert.InDelta(t, time.Second, time.Since(now), float64(time.Millisecond*50))
		})

		t.Run("should interrupt starting Resource instances when Start fails", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 1. Create 3 resourceServices

			errB := errors.New("random error")

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()).Return(errB),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			// 4. Checks if the start was cancelled.
			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			assert.ErrorIs(t, err, errB)
			_ = runner.Finish(context.Background())
		})
	})

	t.Run("Run Server instances", func(t *testing.T) {
		t.Run("should start a Server instance", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			ctx, cancelFunc := context.WithCancel(context.TODO())
			defer cancelFunc()

			// 1. Create 3 resourceServices
			serviceA := NewMockServer(ctrl)
			serviceB := NewMockServer(ctrl)
			serviceC := NewMockServer(ctrl)

			serviceA.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceB.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceC.EXPECT().Listen(gomock.Any()).Return(nil)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			require.NoError(t, err)
		})

		t.Run("WHEN a Serve instance fail starting", func(t *testing.T) {
			t.Run("should interrupt starting a Serve instance", func(t *testing.T) {
				ctrl := gomock.NewController(t)

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockServer(ctrl)
				serviceB := NewMockServer(ctrl)
				serviceC := NewMockServer(ctrl)

				wantErr := errors.New("random error")

				serviceA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 10)
				}).AnyTimes()
				serviceB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 50)
				}).AnyTimes()
				serviceC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 100)
				}).Return(wantErr).AnyTimes()

				serviceA.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceB.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Run the Runner
				runner := services.NewRunner()

				err := runner.Run(ctx, serviceA, serviceB, serviceC)
				require.Equal(t, err, services.MultiErrors{
					nil, nil, wantErr,
				})
			})
		})

		t.Run("WHEN context canceled", func(t *testing.T) {
			t.Run("should interrupt starting services", func(t *testing.T) {
				ctrl := gomock.NewController(t)

				ctx, cancelFunc := context.WithCancel(context.TODO())
				defer cancelFunc()

				// 1. Create 3 resourceServices
				serverA := NewMockServer(ctrl)
				serverB := NewMockServer(ctrl)
				serverC := NewMockServer(ctrl)

				serverA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})
				serverB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})
				serverC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})

				serverA.EXPECT().Close(gomock.Any()).AnyTimes()
				serverB.EXPECT().Close(gomock.Any()).AnyTimes()
				serverC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Run the Runner
				listener := signaltest.NewMockListener(os.Interrupt)
				runner := services.NewRunner(services.WithListenerBuilder(func() signals.Listener {
					return listener
				}))

				go func() {
					err := runner.Run(ctx, serverA, serverB, serverC)
					require.ErrorIs(t, err, context.Canceled)
				}()

				time.Sleep(time.Millisecond * 100)
				cancelFunc()
				time.Sleep(time.Second)
			})
		})
	})
}

func TestRunner_Finish(t *testing.T) {
	t.Run("should stop all services", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.TODO()

		// 1. Create 3 resourceServices
		serviceA := NewMockResource(ctrl)
		serviceB := NewMockServer(ctrl)
		serviceC := NewMockResource(ctrl)
		serviceD := NewMockServer(ctrl)

		serviceA.EXPECT().Start(gomock.Any())
		serviceB.EXPECT().Listen(gomock.Any())
		serviceC.EXPECT().Start(gomock.Any())
		serviceD.EXPECT().Listen(gomock.Any())

		// 2. Triggers the runner
		runner := services.NewRunner()
		err := runner.Run(ctx, serviceA, serviceB, serviceC, serviceD)
		require.NoError(t, err)

		serviceD.EXPECT().Close(gomock.Any()).Do(func(_ context.Context) { time.Sleep(time.Millisecond * 100) })
		serviceC.EXPECT().Stop(gomock.Any()).Do(func(_ context.Context) { time.Sleep(time.Millisecond * 300) })
		serviceB.EXPECT().Close(gomock.Any()).Do(func(_ context.Context) { time.Sleep(time.Millisecond * 500) })
		serviceA.EXPECT().Stop(gomock.Any()).Do(func(_ context.Context) { time.Sleep(time.Millisecond * 700) })

		// 3. Close the resourceServices.
		n := time.Now()
		err = runner.Finish(ctx)
		require.NoError(t, err)
		assert.InDelta(t, time.Millisecond*1600, time.Since(n), float64(time.Millisecond*50))
	})
}

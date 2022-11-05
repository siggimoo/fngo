package fngo

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Pipeline is a connection between two processing stages working on type T.
type Pipeline[T any] struct {
	ctx    context.Context
	group  *errgroup.Group
	values chan T
}

// Filter is a processing stage that passes or blocks values of type T according to whether
// the given function returns true or false, respectively.
func Filter[T any](input Pipeline[T], f func(context.Context, T) (bool, error)) Pipeline[T] {
	output := make(chan T)

	input.group.Go(func() error {
		defer close(output)

		for value := range input.values {
			pass, err := f(input.ctx, value)
			if err != nil {
				return err
			} else if pass {
				select {
				case output <- value:
				case <-input.ctx.Done():
					return input.ctx.Err()
				}
			}
		}

		return nil
	})

	return Pipeline[T]{
		ctx:    input.ctx,
		group:  input.group,
		values: output,
	}
}

// Flatten is a processing stage that collapses a sequence of slices of type T into a single slice of the same type.
func Flatten[T any](input Pipeline[[]T]) Pipeline[T] {
	output := make(chan T)

	input.group.Go(func() error {
		for slice := range input.values {
			for _, value := range slice {
				select {
				case output <- value:
				case <-input.ctx.Done():
					return input.ctx.Err()
				}
			}
		}

		close(output)
		return nil
	})

	return Pipeline[T]{
		ctx:    input.ctx,
		group:  input.group,
		values: output,
	}
}

// Map is a processing stage that converts values of type I into values of type O using the given function.
func Map[I any, O any](input Pipeline[I], f func(context.Context, I) (O, error)) Pipeline[O] {
	output := make(chan O)

	input.group.Go(func() error {
		defer close(output)

		for value := range input.values {
			newValue, err := f(input.ctx, value)
			if err != nil {
				return err
			}

			select {
			case output <- newValue:
			case <-input.ctx.Done():
				return input.ctx.Err()
			}
		}

		return nil
	})

	return Pipeline[O]{
		ctx:    input.ctx,
		group:  input.group,
		values: output,
	}
}

// ParallelMap is identical to Map except the mapping operations are performed in parallel.
// This process is not guaranteed to maintain the order of the values.
func ParallelMap[I any, O any](input Pipeline[I], f func(context.Context, I) (O, error)) Pipeline[O] {
	output := make(chan O)

	input.group.Go(func() error {
		defer close(output)
		mappingGroup, mappingContext := errgroup.WithContext(input.ctx)

		for value := range input.values {
			value := value
			mappingGroup.Go(func() error {
				newValue, err := f(mappingContext, value)
				if err != nil {
					return err
				}

				select {
				case output <- newValue:
					return nil

				case <-mappingContext.Done():
					return mappingContext.Err()
				}
			})
		}

		return mappingGroup.Wait()
	})

	return Pipeline[O]{
		ctx:    input.ctx,
		group:  input.group,
		values: output,
	}
}

// Sink is a processing stage that consumes values of type T using the given function. Any error generated
// by the Pipeline's errgroup will be returned here.
func Sink[T any](input Pipeline[T], f func(context.Context, T) error) error {
	input.group.Go(func() error {
		for {
			select {
			case value, ok := <-input.values:
				if !ok {
					return nil
				}
				if err := f(input.ctx, value); err != nil {
					return err
				}

			case <-input.ctx.Done():
				return input.ctx.Err()
			}
		}
	})

	return input.group.Wait()
}

// SliceSource is a helper function around Source that generates values from the given slice.
func SliceSource[T any](ctx context.Context, slice []T) Pipeline[T] {
	return Source(ctx, func(_ context.Context, emit func(T) error) error {
		for _, value := range slice {
			if err := emit(value); err != nil {
				return err
			}
		}

		return nil
	})
}

// Source is a processing stage that generates values of type T using the given function.
// This and all subsequent stages will run within an errgroup created from the given Context.
//
// The channel passed to the generator function is automatically closed when the function returns.
func Source[T any](ctx context.Context, f func(context.Context, func(T) error) error) Pipeline[T] {
	group, groupContext := errgroup.WithContext(ctx)
	output := make(chan T)

	group.Go(func() error {
		defer close(output)

		emit := func(value T) error {
			select {
			case output <- value:
				return nil
			case <-groupContext.Done():
				return groupContext.Err()
			}
		}

		return f(groupContext, emit)
	})

	return Pipeline[T]{
		ctx:    groupContext,
		group:  group,
		values: output,
	}
}

package util

import (
	"ABIevm/pkg/uipool/aave"
	"sort"

	"golang.org/x/exp/constraints"
)

type PoolData interface {
	aave.IUiPoolDataProviderV3AggregatedReserveData
}

func FilterPools[T PoolData](input []T, f func(T) bool) []T {

	out := make([]T, 0, len(input))

	for _, v := range input {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func SortedPools[T constraints.Ordered](input []T, f func(a, b T) bool) []T {

	sort.Slice(input, func(i, j int) bool {
		return f(input[i], input[j])
	})
	return input
}
